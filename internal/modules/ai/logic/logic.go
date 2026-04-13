// Package logic 实现 AI 模块的业务逻辑层。
//
// 核心职责:
//   - 接收 HTTP Handler 的请求
//   - 调用 AIRouter (adk.ResumableAgent) 执行对话
//   - 消费 AsyncIterator 事件并转换为 SSE 推送
//   - 管理 Session/Message/Run 的持久化
package logic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	aimodule "github.com/cy77cc/OpsPilot/internal/modules/ai"
	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	aicheckpoint "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/checkpoint"
	aidaocheckpoint "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/checkpoint"
	aidaodiagnosis "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/diagnosis"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/event"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// EventEmitter 定义 SSE 事件发送接口。
type EventEmitter func(event string, data any)

// ChatInput 是 Chat 方法的输入参数。
type ChatInput struct {
	SessionID       string
	ClientRequestID string
	LastEventID     string
	Message         string
	Scene           string
	Context         map[string]any
	UserID          uint64
}

type projectedRunUpdate struct {
	AssistantType string
	IntentType    string
}

type ChatShell struct {
	SessionID        string
	Scene            string
	Run              *ai.AIRun
	UserMessage      *ai.AIChatMessage
	AssistantMessage *ai.AIChatMessage
	Reused           bool
}

type RunProjectionQuery struct {
	Cursor   string
	Limit    int
	Paginate bool
}

type projectionBlockPage struct {
	airuntime.RunProjection
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type projectionBlockMeta struct {
	Block     airuntime.ProjectionBlock
	CreatedAt time.Time
}

var ErrInvalidProjectionCursor = errors.New("invalid projection cursor")

var newOpsPilotAgent = func(ctx context.Context) (adk.ResumableAgent, error) {
	return aimodule.InitDeepAgent(ctx)
}

// Logic 封装 AI 模块的核心业务逻辑。
type Logic struct {
	svcCtx             *svc.ServiceContext
	ChatDAO            *aidaochat.AIChatDAO
	RunDAO             *aidao.AIRunDAO
	DiagnosisReportDAO *aidaodiagnosis.AIDiagnosisReportDAO
	ApprovalDAO        *aidaoapproval.AIApprovalTaskDAO
	RunEventDAO        *aidao.AIRunEventDAO
	RunProjectionDAO   *aidao.AIRunProjectionDAO
	RunContentDAO      *aidao.AIRunContentDAO
	CheckpointStore    adk.CheckPointStore
	AIRouter           adk.ResumableAgent
	MigrationFlags     event.ApprovalEventMigrationFlags
	projectionGroup    singleflight.Group
}

// NewAILogic 创建 Logic 实例。
func NewAILogic(svcCtx *svc.ServiceContext) *Logic {
	if svcCtx == nil || svcCtx.DB == nil {
		return &Logic{}
	}

	var aiRouter adk.ResumableAgent
	router, err := newOpsPilotAgent(runtimectx.WithServices(context.Background(), svcCtx))
	if err == nil {
		aiRouter = router
	}

	return &Logic{
		svcCtx:             svcCtx,
		ChatDAO:            aidaochat.NewAIChatDAO(svcCtx.DB),
		RunDAO:             aidao.NewAIRunDAO(svcCtx.DB),
		DiagnosisReportDAO: aidaodiagnosis.NewAIDiagnosisReportDAO(svcCtx.DB),
		ApprovalDAO:        aidaoapproval.NewAIApprovalTaskDAO(svcCtx.DB),
		RunEventDAO:        aidao.NewAIRunEventDAO(svcCtx.DB),
		RunProjectionDAO:   aidao.NewAIRunProjectionDAO(svcCtx.DB),
		RunContentDAO:      aidao.NewAIRunContentDAO(svcCtx.DB),
		CheckpointStore:    aicheckpoint.NewStore(aidaocheckpoint.NewAICheckpointDAO(svcCtx.DB), svcCtx.Rdb, ""),
		AIRouter:           aiRouter,
		MigrationFlags:     event.NewApprovalEventMigrationFlagsFromEnv(),
	}
}

func NewLogicWithDB(db *gorm.DB, router adk.ResumableAgent) *Logic {
	if db == nil {
		return &Logic{}
	}
	return &Logic{
		svcCtx:             &svc.ServiceContext{DB: db},
		ChatDAO:            aidaochat.NewAIChatDAO(db),
		RunDAO:             aidao.NewAIRunDAO(db),
		DiagnosisReportDAO: aidaodiagnosis.NewAIDiagnosisReportDAO(db),
		ApprovalDAO:        aidaoapproval.NewAIApprovalTaskDAO(db),
		RunEventDAO:        aidao.NewAIRunEventDAO(db),
		RunProjectionDAO:   aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:      aidao.NewAIRunContentDAO(db),
		AIRouter:           router,
	}
}

// CanResumeSameRunStatus reports whether an approval resume transition should
// remain within the same run attempt for the provided run status.
func (l *Logic) CanResumeSameRunStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "waiting_approval")
}

// Chat 执行一次 AI 对话，通过 SSE 流式返回结果。
//
// 流程:
//  1. 创建或复用 Session
//  2. 创建 User Message 和 Run 记录
//  3. 发送 A2UI meta 事件
//  4. 调用 AIRouter.Run() 获取 AsyncIterator
//  5. 消费事件，投影为 A2UI 事件后推送
//  6. 持久化结果
func (l *Logic) Chat(ctx context.Context, input ChatInput, emit EventEmitter) error {
	if l.ChatDAO == nil || l.RunDAO == nil || l.AIRouter == nil {
		projected := airuntime.NewErrorEvent("", fmt.Errorf("AI service not initialized"))
		emit(projected.Event, projected.Data)
		return nil
	}

	shell, err := l.ensureChatShell(ctx, input)
	if err != nil {
		emit("error", map[string]any{"message": sanitizeUserFacingError(err)})
		return nil
	}

	ctx = l.runtimeContext(ctx)
	ctx, runtime := runtimectx.Ensure(ctx)
	if requestID := strings.TrimSpace(input.ClientRequestID); requestID != "" {
		runtime.RequestID = requestID
	}
	ctx = runtimectx.WithAIMetadata(ctx, runtimectx.AIMetadata{
		SessionID:    shell.SessionID,
		RunID:        shell.Run.ID,
		CheckpointID: shell.Run.ID,
		UserID:       input.UserID,
		Scene:        shell.Scene,
	})
	ctx = aicheckpoint.ContextWithMetadata(ctx, aicheckpoint.Metadata{
		SessionID:    shell.SessionID,
		RunID:        shell.Run.ID,
		CheckpointID: shell.Run.ID,
		UserID:       input.UserID,
		Scene:        shell.Scene,
	})

	// Step 4: 发送 A2UI meta 事件
	if shell.Reused && strings.TrimSpace(input.LastEventID) != "" {
		tailer := &RunTailer{
			RunDAO:      l.RunDAO,
			RunEventDAO: l.RunEventDAO,
		}
		return tailer.ReplayThenTail(ctx, shell.Run.ID, input.LastEventID, emit, TailOptions{})
	}
	meta := airuntime.NewMetaEvent(shell.SessionID, shell.Run.ID, 1)
	seqCounter := 0
	if !shell.Reused {
		eventID, err := l.appendRunEventWithID(ctx, shell.Run.ID, shell.SessionID, &seqCounter, meta.Event, meta.Data)
		if err != nil {
			return fmt.Errorf("append meta event: %w", err)
		}
		meta.Data = withEventID(meta.Data, eventID)
	}
	emit(meta.Event, meta.Data)
	if shell.Reused {
		l.emitExistingShellTerminal(ctx, shell, emit)
		return nil
	}

	// Step 5: 调用 AIRouter
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           l.AIRouter,
		EnableStreaming: true,
		CheckPointStore: l.CheckpointStore,
	})

	agentInput := []*schema.Message{
		schema.UserMessage(l.buildAugmentedMessage(ctx, shell.Scene, input.Context, input.Message)),
	}

	iterator := runner.Run(ctx, agentInput, adk.WithCheckPointID(shell.Run.ID))

	// Step 6: 消费事件
	var (
		projector = airuntime.NewStreamProjector()
	)

	result, err := processAgentIterator(ctx, iteratorProcessInput{
		Iterator:  iterator,
		Projector: projector,
		Emit:      emit,
		ConsumeProjected: func(_ iteratorConsumeKind, events []airuntime.PublicStreamEvent) error {
			_, consumeErr := l.consumeProjectedEvents(ctx, shell.Run.ID, shell.SessionID, &seqCounter, events, emit, nil)
			return consumeErr
		},
		HandleRunUpdate: func(update projectedRunUpdate) {
			if update.AssistantType != "" || update.IntentType != "" {
				_ = l.RunDAO.UpdateRunStatus(ctx, shell.Run.ID, aidao.AIRunStatusUpdate{
					IntentType:    update.IntentType,
					AssistantType: update.AssistantType,
				})
			}
		},
	})
	if err != nil {
		return err
	}
	if result.FatalErr != nil {
		snapshot := result.AssistantSnapshot
		if !shouldRetainPartialStreamSnapshot(result.FatalErr) {
			snapshot = ""
		}
		if err := l.emitTerminalFailure(ctx, shell, &seqCounter, result.FatalErr, result.SummaryText, snapshot, emit); err != nil {
			return fmt.Errorf("finalize iterator error: %w", err)
		}
		return nil
	}
	if persisted := projector.GetPersistedState(); persisted != nil && !persisted.CanFinalizeDone() {
		runStatus := aidao.AIRunStatusUpdate{
			Status:             "waiting_approval",
			AssistantMessageID: shell.AssistantMessage.ID,
		}
		if err := l.finalizeRunCritical(ctx, shell, runStatus, result.SummaryText); err != nil {
			return fmt.Errorf("persist waiting approval state: %w", err)
		}
		_ = l.persistRunEnhancementsBestEffort(ctx, shell.Run.ID, shell.SessionID, runStatus.Status, result.SummaryText)
		emit("run_state", map[string]any{
			"run_id":  shell.Run.ID,
			"status":  "waiting_approval",
			"agent":   "executor",
			"summary": result.SummaryText,
		})
		return nil
	}

	done := projector.Finish(shell.Run.ID)
	if payload, ok := done.Data.(map[string]any); ok {
		ensureDoneSummary(payload, result.SummaryText, result.HasToolErrors)
		done.Data = payload
	}
	eventID, err := l.appendRunEventWithID(ctx, shell.Run.ID, shell.SessionID, &seqCounter, done.Event, done.Data)
	if err != nil {
		return fmt.Errorf("append meta event: %w", err)
	}
	runStatus := aidao.AIRunStatusUpdate{
		Status:             "completed",
		AssistantMessageID: shell.AssistantMessage.ID,
	}
	if result.HasToolErrors {
		runStatus.Status = "completed_with_tool_errors"
	}
	if err := l.finalizeRunCritical(ctx, shell, runStatus, ""); err != nil {
		return fmt.Errorf("finalize run critical: %w", err)
	}
	_ = l.persistRunEnhancementsBestEffort(ctx, shell.Run.ID, shell.SessionID, runStatus.Status, result.SummaryText)

	emit(done.Event, withEventID(done.Data, eventID))

	return nil
}
