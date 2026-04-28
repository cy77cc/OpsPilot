// Package logic 实现 AI 模块的业务逻辑层入口。
//
// Logic 是门面结构体，所有实现委托给 chat/approval/stream/policy 子包。
package logic

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/orchestrator"
	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	aicheckpoint "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/checkpoint"
	aidaodiagnosis "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/diagnosis"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/approval"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/chat"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/event"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

// ── 类型别名（向后兼容） ──────────────────────────────────────

type (
	ChatInput                 = chat.ChatInput
	ChatShell                 = chat.ChatShell
	SessionSummary            = chat.SessionSummary
	ResumableCredentials      = chat.ResumableCredentials
	RunProjectionQuery        = chat.RunProjectionQuery
	SubmitApprovalInput       = approval.SubmitApprovalInput
	SubmitApprovalOutput      = approval.SubmitApprovalOutput
	RetryResumeApprovalInput  = approval.RetryResumeApprovalInput
	RetryResumeApprovalOutput = approval.RetryResumeApprovalOutput
	ApprovalWorker            = approval.Worker
	ApprovalWorkerOption      = approval.WorkerOption
	ApprovalExpirer           = approval.Expirer
	EventEmitter              = chat.EventEmitter
)

var ErrInvalidProjectionCursor = chat.ErrInvalidProjectionCursor

// ── 工厂函数 ─────────────────────────────────────────────────

var newOpsPilotAgent = func(ctx context.Context) (adk.ResumableAgent, error) {
	return orchestrator.NewOpsPilotAgentFromContext(ctx)
}

// NewAILogic 从 ServiceContext 创建 Logic 实例。
func NewAILogic(svcCtx *svc.ServiceContext) *Logic {
	if svcCtx == nil || svcCtx.DB == nil {
		return &Logic{}
	}
	var aiRouter adk.ResumableAgent
	if r, err := newOpsPilotAgent(runtimectx.WithServices(context.Background(), svcCtx)); err == nil {
		aiRouter = r
	}
	return New(Deps{
		ServiceContext:     svcCtx,
		ChatDAO:            aidaochat.NewAIChatDAO(svcCtx.DB),
		RunDAO:             aidao.NewAIRunDAO(svcCtx.DB),
		DiagnosisReportDAO: aidaodiagnosis.NewAIDiagnosisReportDAO(svcCtx.DB),
		ApprovalDAO:        aidaoapproval.NewAIApprovalTaskDAO(svcCtx.DB),
		RunEventDAO:        aidao.NewAIRunEventDAO(svcCtx.DB),
		RunProjectionDAO:   aidao.NewAIRunProjectionDAO(svcCtx.DB),
		RunContentDAO:      aidao.NewAIRunContentDAO(svcCtx.DB),
		CheckpointStore:    aicheckpoint.NewStore(aicheckpoint.NewAICheckpointDAO(svcCtx.DB), svcCtx.Rdb, ""),
		AIRouter:           aiRouter,
		MigrationFlags:     event.NewApprovalEventMigrationFlagsFromEnv(),
	})
}

// NewLogicWithDB 从 DB 和 Router 创建 Logic 实例（测试用）。
func NewLogicWithDB(db *gorm.DB, router adk.ResumableAgent) *Logic {
	if db == nil {
		return &Logic{}
	}
	return New(Deps{
		ServiceContext:     &svc.ServiceContext{DB: db},
		ChatDAO:            aidaochat.NewAIChatDAO(db),
		RunDAO:             aidao.NewAIRunDAO(db),
		DiagnosisReportDAO: aidaodiagnosis.NewAIDiagnosisReportDAO(db),
		ApprovalDAO:        aidaoapproval.NewAIApprovalTaskDAO(db),
		RunEventDAO:        aidao.NewAIRunEventDAO(db),
		RunProjectionDAO:   aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:      aidao.NewAIRunContentDAO(db),
		AIRouter:           router,
	})
}

// ── Logic 门面 ──────────────────────────────────────────────

// Logic 是 AI 模块的业务逻辑门面，所有方法委托给子包。
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
	chatLogic          *chat.Logic
	initOnce           sync.Once
	mu                 sync.RWMutex
}

// Chat 执行一次 AI 对话（委托给 chat 子包）。
func (l *Logic) Chat(ctx context.Context, input ChatInput, emit chat.EventEmitter) error {
	l.ensureChatLogic()
	return chat.Chat(ctx, l.chatLogic, input, emit)
}

func (l *Logic) ensureChatLogic() {
	l.initOnce.Do(func() {
		if l.svcCtx != nil && l.svcCtx.DB != nil {
			l.chatLogic = chat.New(l.svcCtx)
		}
	})
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.chatLogic != nil {
		// Keep chat sub-logic dependencies in sync when tests assign AIRouter after construction.
		l.chatLogic.AIRouter = l.AIRouter
		l.chatLogic.CheckpointStore = l.CheckpointStore
	}
}

func (l *Logic) BuildAugmentedMessage(ctx context.Context, scene string, sceneContext map[string]any, message string) string {
	l.ensureChatLogic()
	return chat.BuildAugmentedMessage(ctx, l.chatLogic, scene, sceneContext, message)
}

// ── Chat/Session 委托 ────────────────────────────────────────

func (l *Logic) CreateSession(ctx context.Context, userID uint64, title, scene string) (*ai.AIChatSession, error) {
	return l.chatLogic.CreateSession(ctx, userID, title, scene)
}

func (l *Logic) ListSessions(ctx context.Context, userID uint64, scene string) ([]SessionSummary, error) {
	return l.chatLogic.ListSessions(ctx, userID, scene)
}

func (l *Logic) GetSession(ctx context.Context, userID uint64, scene, sessionID string) (*ai.AIChatSession, []ai.AIChatMessage, error) {
	return l.chatLogic.GetSession(ctx, userID, scene, sessionID)
}

func (l *Logic) DeleteSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	return l.chatLogic.DeleteSession(ctx, userID, sessionID)
}

func (l *Logic) GetMessageWithOwnership(ctx context.Context, userID uint64, messageID string) (*ai.AIChatMessage, error) {
	return l.chatLogic.GetMessageWithOwnership(ctx, userID, messageID)
}

func (l *Logic) GetRun(ctx context.Context, userID uint64, runID string) (*ai.AIRun, *ai.AIDiagnosisReport, error) {
	run, err := l.chatLogic.GetRun(ctx, userID, runID)
	if err != nil || run == nil {
		return run, nil, err
	}
	var report *ai.AIDiagnosisReport
	if l.DiagnosisReportDAO != nil {
		report, err = l.DiagnosisReportDAO.GetReportByRunID(ctx, run.ID)
	}
	return run, report, err
}

func (l *Logic) BuildResumableCredentials(ctx context.Context, run *ai.AIRun) (*ResumableCredentials, error) {
	return l.chatLogic.BuildResumableCredentials(ctx, run)
}

func (l *Logic) GetRunProjection(ctx context.Context, userID uint64, runID string) (*ai.AIRunProjection, error) {
	return chat.GetRunProjection(ctx, l.chatLogic, userID, runID)
}

func (l *Logic) GetRunProjectionPayload(ctx context.Context, userID uint64, runID string, q RunProjectionQuery) (any, error) {
	return chat.GetRunProjectionPayload(ctx, l.chatLogic, userID, runID, q)
}

func (l *Logic) GetRunContent(ctx context.Context, userID uint64, contentID string) (*ai.AIRunContent, error) {
	return chat.GetRunContent(ctx, l.chatLogic, userID, contentID)
}

func (l *Logic) GetDiagnosisReport(ctx context.Context, userID uint64, reportID string) (*ai.AIDiagnosisReport, error) {
	if l.DiagnosisReportDAO == nil {
		return nil, nil
	}
	report, err := l.DiagnosisReportDAO.GetReport(ctx, reportID)
	if err != nil || report == nil {
		return report, err
	}
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, report.SessionID, userID, "")
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, fmt.Errorf("report ownership check failed")
		}
	}
	return report, nil
}

func (l *Logic) SubmitFeedback(ctx context.Context, userID uint64, messageID string, action string) error {
	if l.ChatDAO == nil {
		return fmt.Errorf("chat service not initialized")
	}
	msg, err := l.ChatDAO.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return fmt.Errorf("message not found")
	}
	// Verify user owns the session
	session, err := l.ChatDAO.GetSession(ctx, msg.SessionID, userID, "")
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("permission denied")
	}
	return l.ChatDAO.UpdateMessage(ctx, messageID, map[string]any{"feedback": action})
}

// ── Approval 委托 ────────────────────────────────────────────

func (l *Logic) SubmitApproval(ctx context.Context, input SubmitApprovalInput) (*SubmitApprovalOutput, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil, fmt.Errorf("approval service not initialized")
	}
	return approval.NewWriteModel(l.svcCtx.DB).SubmitApproval(ctx, input)
}

func (l *Logic) RetryResumeApproval(ctx context.Context, input RetryResumeApprovalInput) (*RetryResumeApprovalOutput, error) {
	if l == nil || l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil, fmt.Errorf("approval service not initialized")
	}
	return approval.NewWriteModel(l.svcCtx.DB).RetryResumeApproval(ctx, input)
}

func (l *Logic) GetApproval(ctx context.Context, approvalID string, userID uint64) (*ai.AIApprovalTask, error) {
	if l.ApprovalDAO == nil {
		return nil, nil
	}
	task, err := l.ApprovalDAO.GetByApprovalID(ctx, approvalID)
	if err != nil || task == nil {
		return task, err
	}
	if l.ChatDAO != nil && task.SessionID != "" {
		session, err := l.ChatDAO.GetSession(ctx, task.SessionID, userID, "")
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, fmt.Errorf("approval ownership check failed")
		}
	}
	return task, nil
}

func (l *Logic) ListPendingApprovals(ctx context.Context, userID uint64) ([]ai.AIApprovalTask, error) {
	if l.ApprovalDAO == nil {
		return []ai.AIApprovalTask{}, nil
	}
	return l.ApprovalDAO.ListPendingByUserID(ctx, userID, 50)
}

func (l *Logic) ListPendingApprovalsGlobal(ctx context.Context, page, pageSize int) ([]ai.AIApprovalTask, int64, error) {
	if l.ApprovalDAO == nil {
		return []ai.AIApprovalTask{}, 0, nil
	}
	return l.ApprovalDAO.ListPendingPage(ctx, page, pageSize)
}

func (l *Logic) GetApprovalGlobal(ctx context.Context, approvalID string) (*ai.AIApprovalTask, error) {
	if l.ApprovalDAO == nil {
		return nil, nil
	}
	return l.ApprovalDAO.GetByApprovalID(ctx, approvalID)
}

// ── Worker / Expirer 工厂 ────────────────────────────────────

func ApprovalDecidedEventTypes() []string { return approval.ApprovalDecidedEventTypes() }
func CanResumeSameRunStatus(status string) bool {
	return strings.EqualFold(strings.TrimSpace(status), "waiting_approval")
}

func NewApprovalWorker(l *Logic, opts ...ApprovalWorkerOption) *ApprovalWorker {
	if l != nil {
		opts = append([]ApprovalWorkerOption{
			approval.WithWorkerResume(func(ctx context.Context, task *ai.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
				l.ensureChatLogic()
				if l.chatLogic == nil {
					return nil, fmt.Errorf("chat resume logic not initialized")
				}
				return chat.ResumeApprovedTask(ctx, l.chatLogic, task, params)
			}),
		}, opts...)
	}
	return approval.NewWorker(&approval.Logic{
		SvcCtx: l.svcCtx, ChatDAO: l.ChatDAO, RunDAO: l.RunDAO,
		RunEventDAO: l.RunEventDAO, ApprovalDAO: l.ApprovalDAO,
		AIRouter: l.AIRouter, CheckpointStore: l.CheckpointStore,
	}, opts...)
}

func NewApprovalExpirer(l *Logic) *ApprovalExpirer {
	return approval.NewExpirer(&approval.Logic{
		SvcCtx: l.svcCtx, ChatDAO: l.ChatDAO, RunDAO: l.RunDAO,
		RunEventDAO: l.RunEventDAO, ApprovalDAO: l.ApprovalDAO,
		AIRouter: l.AIRouter, CheckpointStore: l.CheckpointStore,
	})
}

func WithApprovalWorkerResume(fn approval.ResumeFunc) ApprovalWorkerOption {
	return approval.WithWorkerResume(fn)
}
func WithApprovalWorkerClock(now func() time.Time) ApprovalWorkerOption {
	return approval.WithWorkerClock(now)
}
func WithApprovalWorkerLeaseWindow(d time.Duration) ApprovalWorkerOption {
	return approval.WithWorkerLeaseWindow(d)
}
func WithApprovalWorkerRetryDelay(d time.Duration) ApprovalWorkerOption {
	return approval.WithWorkerRetryDelay(d)
}

// ── 测试兼容 ─────────────────────────────────────────────────

func (l *Logic) AppendRunEvent(ctx context.Context, runID, sessionID string, seq *int, eventName string, payload any) error {
	l.ensureChatLogic()
	_, err := chat.AppendRunEventWithID(ctx, l.chatLogic, runID, sessionID, seq, eventName, payload)
	return err
}
