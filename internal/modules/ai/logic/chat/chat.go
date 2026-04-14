// Package chat 实现 AI Chat/Session 相关的纯函数和类型。
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	aicheckpoint "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/checkpoint"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/stream"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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

// ChatShell 是 Chat 方法的输出壳。
type ChatShell struct {
	SessionID        string
	Scene            string
	Run              *ai.AIRun
	UserMessage      *ai.AIChatMessage
	AssistantMessage *ai.AIChatMessage
	Reused           bool
}

// SessionSummary 是会话摘要。
type SessionSummary struct {
	Session     ai.AIChatSession
	LastMessage *ai.AIChatMessage
}

// ResumableCredentials 包含恢复运行所需的凭证。
type ResumableCredentials struct {
	RunID           string `json:"run_id,omitempty"`
	ClientRequestID string `json:"client_request_id,omitempty"`
	LatestEventID   string `json:"latest_event_id,omitempty"`
	ApprovalID      string `json:"approval_id,omitempty"`
	Resumable       bool   `json:"resumable"`
}

// RunProjectionQuery 是投影查询参数。
type RunProjectionQuery struct {
	Cursor   string
	Limit    int
	Paginate bool
}

// Logic 提供 chat 子包所需的最小依赖视图。
type Logic struct {
	SvcCtx           *svc.ServiceContext
	ChatDAO          *aidaochat.AIChatDAO
	RunDAO           *aidao.AIRunDAO
	RunEventDAO      *aidao.AIRunEventDAO
	RunProjectionDAO *aidao.AIRunProjectionDAO
	RunContentDAO    *aidao.AIRunContentDAO
	AIRouter         adk.ResumableAgent
	CheckpointStore  adk.CheckPointStore
}

// New 创建 Logic 实例。
func New(svcCtx *svc.ServiceContext) *Logic {
	if svcCtx == nil || svcCtx.DB == nil {
		return &Logic{}
	}
	return &Logic{
		SvcCtx: svcCtx, ChatDAO: aidaochat.NewAIChatDAO(svcCtx.DB), RunDAO: aidao.NewAIRunDAO(svcCtx.DB),
		RunEventDAO: aidao.NewAIRunEventDAO(svcCtx.DB), RunProjectionDAO: aidao.NewAIRunProjectionDAO(svcCtx.DB),
		RunContentDAO: aidao.NewAIRunContentDAO(svcCtx.DB),
	}
}

// EnsureChatShell 创建或复用会话壳。
func EnsureChatShell(ctx context.Context, l *Logic, input ChatInput) (ChatShell, error) {
	shell := ChatShell{}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	session, err := l.ChatDAO.GetSession(ctx, sessionID, input.UserID, input.Scene)
	if err != nil {
		return shell, fmt.Errorf("get session: %w", err)
	}
	scene := ResolveChatScene(input.Scene, session)
	if session == nil {
		session = &ai.AIChatSession{
			ID:     sessionID,
			UserID: input.UserID,
			Scene:  scene,
			Title:  BuildSessionTitle(input.Message),
		}
		if err := l.ChatDAO.CreateSession(ctx, session); err != nil {
			return shell, fmt.Errorf("create session: %w", err)
		}
	}

	var createdUser, createdAssistant *ai.AIChatMessage
	run, created, err := l.RunDAO.CreateOrReuseRunShell(ctx, input.UserID, sessionID, input.ClientRequestID, func() (*ai.AIRun, *ai.AIChatMessage, *ai.AIChatMessage) {
		userMessageID := uuid.NewString()
		assistantMessageID := uuid.NewString()
		createdUser = &ai.AIChatMessage{ID: userMessageID, SessionID: sessionID, Role: "user", Content: input.Message, Status: "done"}
		createdAssistant = &ai.AIChatMessage{ID: assistantMessageID, SessionID: sessionID, Role: "assistant", Content: "", Status: "streaming"}
		return &ai.AIRun{ID: uuid.NewString(), SessionID: sessionID, ClientRequestID: strings.TrimSpace(input.ClientRequestID), UserMessageID: userMessageID, AssistantMessageID: assistantMessageID, Status: "running", TraceJSON: "{}"}, createdUser, createdAssistant
	})
	if err != nil {
		return shell, fmt.Errorf("create run shell: %w", err)
	}

	shell = ChatShell{SessionID: sessionID, Scene: scene, Run: run, Reused: !created}
	if created {
		shell.UserMessage = createdUser
		shell.AssistantMessage = createdAssistant
		return shell, nil
	}

	userMessage, err := l.ChatDAO.GetMessage(ctx, run.UserMessageID)
	if err != nil {
		return shell, fmt.Errorf("load user message shell: %w", err)
	}
	assistantMessage, err := l.ChatDAO.GetMessage(ctx, run.AssistantMessageID)
	if err != nil {
		return shell, fmt.Errorf("load assistant message shell: %w", err)
	}
	if userMessage == nil || assistantMessage == nil {
		return shell, fmt.Errorf("load reused shell messages")
	}
	shell.UserMessage = userMessage
	shell.AssistantMessage = assistantMessage
	return shell, nil
}

// CreateSession 创建会话。
func (l *Logic) CreateSession(ctx context.Context, userID uint64, title, scene string) (*ai.AIChatSession, error) {
	if l.ChatDAO == nil {
		return nil, nil
	}
	s := &ai.AIChatSession{ID: uuid.NewString(), UserID: userID, Title: title, Scene: NormalizeScene(scene)}
	if err := l.ChatDAO.CreateSession(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// ListSessions 列出用户的所有会话。
func (l *Logic) ListSessions(ctx context.Context, userID uint64, scene string) ([]SessionSummary, error) {
	if l.ChatDAO == nil {
		return []SessionSummary{}, nil
	}
	rows, err := l.ChatDAO.ListSessionSummaries(ctx, userID, scene)
	if err != nil {
		return nil, err
	}
	summaries := make([]SessionSummary, 0, len(rows))
	for _, row := range rows {
		summaries = append(summaries, SessionSummary{Session: row.Session(), LastMessage: row.LastMessage()})
	}
	return summaries, nil
}

// GetSession 获取会话详情。
func (l *Logic) GetSession(ctx context.Context, userID uint64, scene, sessionID string) (*ai.AIChatSession, []ai.AIChatMessage, error) {
	if l.ChatDAO == nil {
		return nil, nil, nil
	}
	session, err := l.ChatDAO.GetSession(ctx, sessionID, userID, scene)
	if err != nil || session == nil {
		return session, nil, err
	}
	messages, err := l.ChatDAO.ListMessagesBySession(ctx, session.ID)
	if err != nil {
		return nil, nil, err
	}
	return session, messages, nil
}

// DeleteSession 删除会话。
func (l *Logic) DeleteSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	if l.ChatDAO == nil {
		return false, nil
	}
	session, err := l.ChatDAO.GetSession(ctx, sessionID, userID, "")
	if err != nil || session == nil {
		return false, err
	}
	if err := l.ChatDAO.DeleteSession(ctx, session.ID, userID); err != nil {
		return false, err
	}
	return true, nil
}

// GetMessageWithOwnership 获取消息并验证所有权。
func (l *Logic) GetMessageWithOwnership(ctx context.Context, userID uint64, messageID string) (*ai.AIChatMessage, error) {
	if l.ChatDAO == nil {
		return nil, nil
	}
	message, err := l.ChatDAO.GetMessage(ctx, messageID)
	if err != nil || message == nil {
		return nil, err
	}
	session, err := l.ChatDAO.GetSession(ctx, message.SessionID, userID, "")
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}
	return message, nil
}

// GetRun 获取 Run 状态。
func (l *Logic) GetRun(ctx context.Context, userID uint64, runID string) (*ai.AIRun, error) {
	if l.RunDAO == nil {
		return nil, nil
	}
	run, err := l.RunDAO.GetRun(ctx, runID)
	if err != nil || run == nil {
		return run, err
	}
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, run.SessionID, userID, "")
		if err != nil || session == nil {
			return nil, err
		}
	}
	return run, nil
}

// BuildResumableCredentials 构建恢复运行所需的凭证。
func (l *Logic) BuildResumableCredentials(ctx context.Context, run *ai.AIRun) (*ResumableCredentials, error) {
	if run == nil || !isTailOpenStatus(run.Status) {
		return nil, nil
	}
	creds := &ResumableCredentials{
		RunID:           strings.TrimSpace(run.ID),
		ClientRequestID: strings.TrimSpace(run.ClientRequestID),
		Resumable:       true,
	}
	if creds.ClientRequestID == "" {
		creds.ClientRequestID = creds.RunID
	}
	if l != nil && l.RunEventDAO != nil {
		events, err := l.RunEventDAO.ListByRun(ctx, run.ID)
		if err != nil {
			return nil, err
		}
		if len(events) > 0 {
			creds.LatestEventID = strings.TrimSpace(events[len(events)-1].ID)
		}
	}
	approvalID, err := l.lookupActiveApprovalID(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	creds.ApprovalID = approvalID
	return creds, nil
}

func (l *Logic) lookupActiveApprovalID(ctx context.Context, runID string) (string, error) {
	if l == nil || l.SvcCtx == nil || l.SvcCtx.DB == nil || strings.TrimSpace(runID) == "" {
		return "", nil
	}
	var task ai.AIApprovalTask
	err := l.SvcCtx.DB.WithContext(ctx).Where("run_id = ? AND status = ?", strings.TrimSpace(runID), "pending").Order("created_at DESC, id DESC").First(&task).Error
	if err == nil {
		return strings.TrimSpace(task.ApprovalID), nil
	}
	if err != nil && !isRecordNotFound(err) {
		return "", err
	}
	err = l.SvcCtx.DB.WithContext(ctx).Where("run_id = ? AND status = ?", strings.TrimSpace(runID), "approved").Order("decided_at DESC, created_at DESC, id DESC").First(&task).Error
	if err == nil {
		return strings.TrimSpace(task.ApprovalID), nil
	}
	if isRecordNotFound(err) {
		return "", nil
	}
	return "", err
}

func isRecordNotFound(err error) bool {
	return err != nil && err.Error() == "record not found"
}

func isTailOpenStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "waiting_approval", "resuming", "running", "resume_failed_retryable":
		return true
	default:
		return false
	}
}

// BuildAugmentedMessage 构建增强后的用户消息。
func BuildAugmentedMessage(ctx context.Context, l *Logic, scene string, sceneContext map[string]any, message string) string {
	scene = NormalizeScene(scene)
	sections := []string{
		"[Hidden platform context for routing, tool selection, and safety policy]",
		"[Scene]",
		fmt.Sprintf("scene=%s", scene),
	}
	if payload := StringifyJSON(sceneContext); payload != "" && payload != "{}" {
		sections = append(sections, "", "[Scene Context]", fmt.Sprintf("scene_context=%s", payload))
	}
	sceneSections := loadSceneAugmentation(ctx, l, scene)
	if len(sceneSections) > 0 {
		for _, section := range sceneSections {
			if len(section) == 0 {
				continue
			}
			sections = append(sections, "", strings.Join(section, "\n"))
		}
	}
	sections = append(sections, "", fmt.Sprintf("User request:\n%s", strings.TrimSpace(message)))
	return strings.Join(sections, "\n")
}

func loadSceneAugmentation(ctx context.Context, l *Logic, scene string) [][]string {
	if l == nil || l.SvcCtx == nil || l.SvcCtx.DB == nil || strings.TrimSpace(scene) == "" {
		return nil
	}
	var prompts []ai.AIScenePrompt
	_ = l.SvcCtx.DB.WithContext(ctx).Where("scene = ? AND is_active = ?", scene, true).Order("display_order ASC, id ASC").Find(&prompts).Error
	var config ai.AISceneConfig
	hasConfig := l.SvcCtx.DB.WithContext(ctx).Where("scene = ?", scene).First(&config).Error == nil

	sceneLines := make([]string, 0, 4)
	if len(prompts) > 0 {
		promptTexts := make([]string, 0, len(prompts))
		for _, item := range prompts {
			if text := strings.TrimSpace(item.PromptText); text != "" {
				promptTexts = append(promptTexts, text)
			}
		}
		if len(promptTexts) > 0 {
			sceneLines = append(sceneLines, fmt.Sprintf("scene_prompts=%s", StringifyJSON(promptTexts)))
		}
	}
	if hasConfig {
		if description := strings.TrimSpace(config.Description); description != "" {
			sceneLines = append(sceneLines, fmt.Sprintf("scene_description=%s", description))
		}
		if constraints := CompactJSONString(config.ConstraintsJSON); constraints != "" {
			sceneLines = append(sceneLines, fmt.Sprintf("scene_constraints=%s", constraints))
		}
	}
	sections := make([][]string, 0, 2)
	if len(sceneLines) > 0 {
		sections = append(sections, append([]string{"[Scene Prompts & Constraints]"}, sceneLines...))
	}
	toolLines := make([]string, 0, 3)
	if hasConfig {
		if allowed := CompactJSONString(config.AllowedToolsJSON); allowed != "" {
			toolLines = append(toolLines, fmt.Sprintf("allowed_tools=%s", allowed))
		}
		if blocked := CompactJSONString(config.BlockedToolsJSON); blocked != "" {
			toolLines = append(toolLines, fmt.Sprintf("blocked_tools=%s", blocked))
		}
	}
	if len(toolLines) > 0 {
		toolLines = append(toolLines, "These tool constraints are mandatory.")
		sections = append(sections, append([]string{"[Tool Constraints]"}, toolLines...))
	}
	return sections
}

// BuildSessionTitle 从首条消息生成会话标题。
func BuildSessionTitle(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "New AI session"
	}
	return TruncateString(trimmed, 48)
}

// NormalizeScene 规范化场景名称。
func NormalizeScene(scene string) string {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return "ai"
	}
	return scene
}

// ResolveChatScene 解析聊天场景。
func ResolveChatScene(requestScene string, session *ai.AIChatSession) string {
	if strings.TrimSpace(requestScene) != "" {
		return NormalizeScene(requestScene)
	}
	if session != nil && strings.TrimSpace(session.Scene) != "" {
		return NormalizeScene(session.Scene)
	}
	return "ai"
}

// StringifyJSON 将值序列化为 JSON 字符串。
func StringifyJSON(value any) string {
	if value == nil {
		return ""
	}
	b, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(b)
}

// CompactJSONString 压缩 JSON 字符串。
func CompactJSONString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var payload any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return value
	}
	return StringifyJSON(payload)
}

// TruncateString 截断字符串到指定长度。
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	return string([]rune(s)[:maxLen])
}

// ============================================================
// Chat 核心流程
// ============================================================

type EventEmitter func(event string, data any)

// Chat 执行一次 AI 对话，通过 SSE 流式返回结果。
func Chat(ctx context.Context, l *Logic, input ChatInput, emit EventEmitter) error {
	if l.RunDAO == nil || l.AIRouter == nil {
		emit("error", map[string]any{"message": stream.SanitizeUserFacingError(fmt.Errorf("AI service not initialized"))})
		return nil
	}
	shell, err := EnsureChatShell(ctx, l, input)
	if err != nil {
		emit("error", map[string]any{"message": stream.SanitizeUserFacingError(err)})
		return nil
	}
	ctx = l.runtimeContext(ctx)
	ctx, runtime := runtimectx.Ensure(ctx)
	if rid := strings.TrimSpace(input.ClientRequestID); rid != "" {
		runtime.RequestID = rid
	}
	ctx = runtimectx.WithAIMetadata(ctx, runtimectx.AIMetadata{
		SessionID: shell.SessionID, RunID: shell.Run.ID, CheckpointID: shell.Run.ID,
		UserID: input.UserID, Scene: shell.Scene,
	})
	ctx = aicheckpoint.ContextWithMetadata(ctx, aicheckpoint.Metadata{
		SessionID: shell.SessionID, RunID: shell.Run.ID, CheckpointID: shell.Run.ID,
		UserID: input.UserID, Scene: shell.Scene,
	})
	if shell.Reused && strings.TrimSpace(input.LastEventID) != "" {
		tailer := &stream.RunTailer{RunDAO: l.RunDAO, RunEventDAO: l.RunEventDAO}
		return tailer.ReplayThenTail(ctx, shell.Run.ID, input.LastEventID, emit, stream.TailOptions{})
	}
	meta := airuntime.NewMetaEvent(shell.SessionID, shell.Run.ID, 1)
	seq := 0
	if !shell.Reused {
		eid, err := AppendRunEventWithID(ctx, l, shell.Run.ID, shell.SessionID, &seq, meta.Event, meta.Data)
		if err != nil {
			return fmt.Errorf("append meta event: %w", err)
		}
		meta.Data = withEventID(meta.Data, eid)
	}
	emit(meta.Event, meta.Data)
	if shell.Reused {
		EmitExistingShellTerminal(ctx, l, shell, emit)
		return nil
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: l.AIRouter, EnableStreaming: true, CheckPointStore: l.CheckpointStore})
	agentInput := []*schema.Message{schema.UserMessage(BuildAugmentedMessage(ctx, l, shell.Scene, input.Context, input.Message))}
	iterator := runner.Run(ctx, agentInput, adk.WithCheckPointID(shell.Run.ID))
	projector := airuntime.NewStreamProjector()
	result, err := stream.ProcessAgentIterator(ctx, stream.IteratorProcessInput{
		Iterator:  iterator,
		Projector: projector,
		Emit:      emit,
		ConsumeProjected: func(_ stream.IteratorConsumeKind, events []airuntime.PublicStreamEvent) error {
			_, consumeErr := ConsumeProjectedEvents(ctx, l, shell.Run.ID, shell.SessionID, &seq, events, emit)
			return consumeErr
		},
		HandleRunUpdate: func(update stream.RunUpdate) {
			if update.AssistantType != "" || update.IntentType != "" {
				_ = l.RunDAO.UpdateRunStatus(ctx, shell.Run.ID, aidao.AIRunStatusUpdate{IntentType: update.IntentType, AssistantType: update.AssistantType})
			}
		},
	})
	if err != nil {
		return err
	}
	if result.FatalErr != nil {
		snapshot := result.AssistantSnapshot
		if !stream.ShouldRetainPartialStreamSnapshot(result.FatalErr) {
			snapshot = ""
		}
		if err := EmitTerminalFailure(ctx, l, shell, &seq, result.FatalErr, result.SummaryText, snapshot, emit); err != nil {
			return fmt.Errorf("finalize iterator error: %w", err)
		}
		return nil
	}
	if persisted := projector.GetPersistedState(); persisted != nil && !persisted.CanFinalizeDone() {
		runStatus := aidao.AIRunStatusUpdate{Status: "waiting_approval", AssistantMessageID: shell.AssistantMessage.ID}
		if err := FinalizeRunCritical(ctx, l, shell, runStatus, result.SummaryText); err != nil {
			return fmt.Errorf("persist waiting approval state: %w", err)
		}
		_ = PersistRunEnhancementsBestEffort(ctx, l, shell.Run.ID, shell.SessionID, runStatus.Status, result.SummaryText)
		emit("run_state", map[string]any{"run_id": shell.Run.ID, "status": "waiting_approval", "agent": "executor", "summary": result.SummaryText})
		return nil
	}
	done := projector.Finish(shell.Run.ID)
	if payload, ok := done.Data.(map[string]any); ok {
		stream.EnsureDoneSummary(payload, result.SummaryText, result.HasToolErrors)
		done.Data = payload
	}
	eid, err := AppendRunEventWithID(ctx, l, shell.Run.ID, shell.SessionID, &seq, done.Event, done.Data)
	if err != nil {
		return fmt.Errorf("append meta event: %w", err)
	}
	runStatus := aidao.AIRunStatusUpdate{Status: "completed", AssistantMessageID: shell.AssistantMessage.ID}
	if result.HasToolErrors {
		runStatus.Status = "completed_with_tool_errors"
	}
	if err := FinalizeRunCritical(ctx, l, shell, runStatus, ""); err != nil {
		return fmt.Errorf("finalize run critical: %w", err)
	}
	_ = PersistRunEnhancementsBestEffort(ctx, l, shell.Run.ID, shell.SessionID, runStatus.Status, result.SummaryText)
	emit(done.Event, withEventID(done.Data, eid))
	return nil
}

func (l *Logic) runtimeContext(ctx context.Context) context.Context {
	if l == nil || l.SvcCtx == nil {
		return ctx
	}
	return runtimectx.WithServices(ctx, l.SvcCtx)
}

// EmitTerminalFailure 发送终端失败事件并持久化。
func EmitTerminalFailure(ctx context.Context, l *Logic, shell ChatShell, seq *int, internalErr error, summaryBody, assistantBody string, emit EventEmitter) error {
	publicErr := stream.SanitizeUserFacingError(internalErr)
	projected := airuntime.NewErrorEvent(shell.Run.ID, errors.New(publicErr))
	eid, err := AppendRunEventWithID(ctx, l, shell.Run.ID, shell.SessionID, seq, projected.Event, projected.Data)
	if err != nil {
		return err
	}
	emit(projected.Event, withEventID(projected.Data, eid))
	runUpdate := aidao.AIRunStatusUpdate{AssistantMessageID: shell.AssistantMessage.ID, Status: "failed_runtime", ErrorMessage: internalErr.Error()}
	snapshot := stream.BuildAssistantFailureSnapshot(summaryBody, assistantBody, publicErr)
	if err := FinalizeRunCritical(ctx, l, shell, runUpdate, snapshot); err != nil {
		return err
	}
	if err := PersistRunEnhancementsBestEffort(ctx, l, shell.Run.ID, shell.SessionID, runUpdate.Status, snapshot); err != nil {
		if strings.TrimSpace(summaryBody) == "" && strings.TrimSpace(assistantBody) == "" {
			return nil
		}
		return fmt.Errorf("persist run artifacts: %w", err)
	}
	return nil
}

// FinalizeRunCritical 事务性更新消息和运行状态。
func FinalizeRunCritical(ctx context.Context, l *Logic, shell ChatShell, runUpdate aidao.AIRunStatusUpdate, assistantContent string) error {
	if l.SvcCtx == nil || l.SvcCtx.DB == nil {
		return nil
	}
	return l.SvcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chatDAO := aidaochat.NewAIChatDAO(tx)
		runDAO := aidao.NewAIRunDAO(tx)
		if err := chatDAO.UpdateMessage(ctx, shell.AssistantMessage.ID, map[string]any{"content": assistantContent, "status": assistantStatusFromRunStatus(runUpdate.Status)}); err != nil {
			return err
		}
		runUpdate.AssistantMessageID = shell.AssistantMessage.ID
		runUpdate.ProgressSummary = TruncateString(assistantContent, 500)
		return runDAO.UpdateRunStatus(ctx, shell.Run.ID, runUpdate)
	})
}

// PersistRunEnhancementsBestEffort 持久化投影和内容。
func PersistRunEnhancementsBestEffort(ctx context.Context, l *Logic, runID, sessionID, status string, _ string) error {
	if l.RunEventDAO == nil || l.RunProjectionDAO == nil || l.RunContentDAO == nil {
		return nil
	}
	events, err := l.RunEventDAO.ListByRun(ctx, runID)
	if err != nil {
		return err
	}
	projection, contents, err := airuntime.BuildProjection(events)
	if err != nil {
		return err
	}
	projection.Status = status
	projectionJSON, err := json.Marshal(projection)
	if err != nil {
		return err
	}
	for _, content := range contents {
		if err := l.RunContentDAO.Create(ctx, content); err != nil {
			return err
		}
	}
	return l.RunProjectionDAO.Upsert(ctx, &ai.AIRunProjection{
		ID: uuid.NewString(), RunID: runID, SessionID: sessionID,
		Version: projection.Version, Status: projection.Status, ProjectionJSON: string(projectionJSON),
	})
}

// EmitExistingShellTerminal 对重用 shell 发送终态事件。
func EmitExistingShellTerminal(ctx context.Context, l *Logic, shell ChatShell, emit EventEmitter) {
	switch shell.Run.Status {
	case "failed", "failed_runtime":
		emit("error", map[string]any{"run_id": shell.Run.ID, "message": stream.SanitizeUserFacingError(errors.New(shell.Run.ErrorMessage))})
	case "waiting_approval":
		emit("run_state", map[string]any{"run_id": shell.Run.ID, "status": "waiting_approval", "agent": "executor", "summary": shell.AssistantMessage.Content})
	case "cancelled", "expired":
		emit("run_state", map[string]any{"run_id": shell.Run.ID, "status": shell.Run.Status, "agent": "executor", "summary": shell.AssistantMessage.Content})
	case "completed", "completed_with_tool_errors":
		emit("done", map[string]any{"run_id": shell.Run.ID, "status": shell.Run.Status, "summary": shell.AssistantMessage.Content})
	}
}

// AppendRunEventWithID 追加运行事件并返回事件 ID。
func AppendRunEventWithID(ctx context.Context, l *Logic, runID, sessionID string, seq *int, eventName string, payload any) (string, error) {
	if l.RunEventDAO == nil || seq == nil {
		return "", nil
	}
	eventType, raw, err := stream.MarshalProjectedEvent(eventName, payload)
	if err != nil {
		return "", err
	}
	if eventType == "" {
		return "", nil
	}
	eventID := uuid.NewString()
	*seq++
	return eventID, l.RunEventDAO.Create(ctx, &ai.AIRunEvent{
		ID: eventID, RunID: runID, SessionID: sessionID, Seq: *seq,
		EventType: string(eventType), AgentName: stream.EventAgentName(payload),
		ToolCallID: stream.EventToolCallID(payload), PayloadJSON: raw,
	})
}

// ConsumeProjectedEvents 消费投影事件并持久化。
func ConsumeProjectedEvents(ctx context.Context, l *Logic, runID, sessionID string, seq *int, events []airuntime.PublicStreamEvent, emit EventEmitter) (stream.RunUpdate, error) {
	update := stream.AccumulateProjectedEvents(events, nil)
	for _, projected := range events {
		eid, err := AppendRunEventWithID(ctx, l, runID, sessionID, seq, projected.Event, projected.Data)
		if err != nil {
			return update, err
		}
		emit(projected.Event, withEventID(projected.Data, eid))
	}
	return update, nil
}

func assistantStatusFromRunStatus(status string) string {
	switch status {
	case "failed_runtime":
		return "error"
	case "waiting_approval":
		return "streaming"
	default:
		return "done"
	}
}

func withEventID(payload any, eventID string) any {
	if strings.TrimSpace(eventID) == "" {
		return payload
	}
	data, ok := payload.(map[string]any)
	if !ok {
		return payload
	}
	cp := make(map[string]any, len(data)+1)
	for k, v := range data {
		cp[k] = v
	}
	cp["event_id"] = eventID
	return cp
}
