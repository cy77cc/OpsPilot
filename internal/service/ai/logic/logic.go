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
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/agents"
	aicheckpoint "github.com/cy77cc/OpsPilot/internal/ai/checkpoint"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/google/uuid"
)

// EventEmitter 定义 SSE 事件发送接口。
type EventEmitter func(event string, data any)

// ChatInput 是 Chat 方法的输入参数。
type ChatInput struct {
	SessionID string
	Message   string
	Scene     string
	Context   map[string]any
	UserID    uint64
}

type projectedRunUpdate struct {
	AssistantType string
	IntentType    string
}

// Logic 封装 AI 模块的核心业务逻辑。
type Logic struct {
	svcCtx             *svc.ServiceContext
	ChatDAO            *aidao.AIChatDAO
	RunDAO             *aidao.AIRunDAO
	DiagnosisReportDAO *aidao.AIDiagnosisReportDAO
	ApprovalDAO        *aidao.AIApprovalTaskDAO
	CheckpointStore    adk.CheckPointStore
	AIRouter           adk.ResumableAgent
}

// NewAILogic 创建 Logic 实例。
func NewAILogic(svcCtx *svc.ServiceContext) *Logic {
	if svcCtx == nil || svcCtx.DB == nil {
		return &Logic{}
	}

	aiRouter, err := agents.NewRouter(runtimectx.WithServices(context.Background(), svcCtx))
	if err != nil {
		return &Logic{}
	}

	return &Logic{
		svcCtx:             svcCtx,
		ChatDAO:            aidao.NewAIChatDAO(svcCtx.DB),
		RunDAO:             aidao.NewAIRunDAO(svcCtx.DB),
		DiagnosisReportDAO: aidao.NewAIDiagnosisReportDAO(svcCtx.DB),
		ApprovalDAO:        aidao.NewAIApprovalTaskDAO(svcCtx.DB),
		CheckpointStore:    aicheckpoint.NewStore(aidao.NewAICheckpointDAO(svcCtx.DB), svcCtx.Rdb, ""),
		AIRouter:           aiRouter,
	}
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

	// Step 1: 创建或复用 Session，新对话就创建，旧对话前端带上了
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	session, err := l.ChatDAO.GetSession(ctx, sessionID, input.UserID, input.Scene)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	scene := resolveChatScene(input.Scene, session)
	if session == nil {
		if err := l.ChatDAO.CreateSession(ctx, &model.AIChatSession{
			ID:     sessionID,
			UserID: input.UserID,
			Scene:  scene,
			Title:  buildSessionTitle(input.Message),
		}); err != nil {
			return fmt.Errorf("create session: %w", err)
		}
	}

	// Step 2: 创建 User Message
	userMessage := &model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      "user",
		Content:   input.Message,
		Status:    "done",
	}
	if err := l.ChatDAO.CreateMessage(ctx, userMessage); err != nil {
		return fmt.Errorf("create user message: %w", err)
	}

	// Step 3: 创建 Run 记录
	run := &model.AIRun{
		ID:            uuid.NewString(),
		SessionID:     sessionID,
		UserMessageID: userMessage.ID,
		Status:        "running",
		TraceJSON:     "{}",
	}
	if err := l.RunDAO.CreateRun(ctx, run); err != nil {
		return fmt.Errorf("create run: %w", err)
	}
	ctx = l.runtimeContext(ctx)
	ctx = aicheckpoint.ContextWithMetadata(ctx, aicheckpoint.Metadata{
		SessionID: sessionID,
		RunID:     run.ID,
		UserID:    input.UserID,
		Scene:     scene,
	})

	// Step 4: 发送 A2UI meta 事件
	meta := airuntime.NewMetaEvent(sessionID, run.ID, 1)
	emit(meta.Event, meta.Data)

	// Step 5: 调用 AIRouter
	//
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           l.AIRouter,
		EnableStreaming: true,
		CheckPointStore: l.CheckpointStore,
	})

	agentInput := []*schema.Message{
		schema.UserMessage(l.buildAugmentedMessage(ctx, scene, input.Context, input.Message)),
	}

	iterator := runner.Run(ctx, agentInput)

	// Step 6: 消费事件
	var (
		assistantMessage *model.AIChatMessage
		assistantContent strings.Builder
		projector        = airuntime.NewStreamProjector()
		streamError      error
	)

	// 创建 assistant message 占位
	assistantMessage = &model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      "assistant",
		Status:    "streaming",
	}
	if err := l.ChatDAO.CreateMessage(ctx, assistantMessage); err != nil {
		return fmt.Errorf("create assistant message: %w", err)
	}

	for {
		event, ok := iterator.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			streamError = event.Err
			if event.Output != nil && event.Output.MessageOutput.Message.Role == schema.Tool {
				projected := projector.Fail(run.ID, event.Err)
				emit(projected.Event, projected.Data)
				_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{
					Status:       "failed",
					ErrorMessage: event.Err.Error(),
				})
			} else {
				projected := projector.Fail(run.ID, event.Err)
				emit(projected.Event, projected.Data)
				_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{
					Status:       "failed",
					ErrorMessage: event.Err.Error(),
				})
				// 更新消息状态为 error
				_ = l.ChatDAO.UpdateMessage(ctx, assistantMessage.ID, map[string]any{
					"content": assistantContent.String(),
					"status":  "error",
				})
				return nil
			}
		}

		if event.Output != nil && event.Output.MessageOutput != nil && event.Output.MessageOutput.IsStreaming && event.Output.MessageOutput.MessageStream != nil {
			for {
				msg, err := event.Output.MessageOutput.MessageStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					streamError = err
					projected := projector.Fail(run.ID, err)
					emit(projected.Event, projected.Data)
					break
				}
				if msg == nil {
					continue
				}

				chunkEvent := adk.EventFromMessage(msg, nil, msg.Role, msg.ToolName)
				chunkEvent.AgentName = event.AgentName
				update := consumeProjectedEvents(projector.Consume(chunkEvent), emit, &assistantContent)
				if update.AssistantType != "" || update.IntentType != "" {
					_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{
						IntentType:    update.IntentType,
						AssistantType: update.AssistantType,
					})
				}
			}
			continue
		}

		update := consumeProjectedEvents(projector.Consume(event), emit, &assistantContent)
		if update.AssistantType != "" || update.IntentType != "" {
			_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{
				IntentType:    update.IntentType,
				AssistantType: update.AssistantType,
			})
		}
	}

	// Step 7: 持久化结果
	finalContent := assistantContent.String()
	finalStatus := "done"
	if streamError != nil {
		finalStatus = "error"
	}

	// 提取持久化状态
	runtimeJSON := ""
	if persisted := projector.GetPersistedState(); persisted != nil && len(persisted.Activities) > 0 {
		if data, err := json.Marshal(persisted); err == nil {
			runtimeJSON = string(data)
		}
	}

	if err := l.ChatDAO.UpdateMessage(ctx, assistantMessage.ID, map[string]any{
		"content":      finalContent,
		"status":       finalStatus,
		"runtime_json": runtimeJSON,
	}); err != nil {
		return fmt.Errorf("update assistant message: %w", err)
	}

	// 更新 Run 状态
	runStatus := aidao.AIRunStatusUpdate{
		Status:             "completed",
		AssistantMessageID: assistantMessage.ID,
		ProgressSummary:    truncateString(finalContent, 500),
	}
	if streamError != nil {
		runStatus.Status = "failed"
		runStatus.ErrorMessage = streamError.Error()
	}
	if err := l.RunDAO.UpdateRunStatus(ctx, run.ID, runStatus); err != nil {
		return fmt.Errorf("update run status: %w", err)
	}

	// Step 8: 发送 done 事件前先刷新缓冲
	if remaining := projector.FlushBuffer(); len(remaining) > 0 {
		for _, e := range remaining {
			emit(e.Event, e.Data)
		}
	}
	done := projector.Finish(run.ID)
	emit(done.Event, done.Data)

	return nil
}

func consumeProjectedEvents(events []airuntime.PublicStreamEvent, emit EventEmitter, assistantContent *strings.Builder) projectedRunUpdate {
	update := projectedRunUpdate{}
	for _, projected := range events {
		if projected.Event == "delta" {
			if data, ok := projected.Data.(map[string]any); ok {
				if content, ok := data["content"].(string); ok {
					assistantContent.WriteString(content)
				}
			}
		}
		if projected.Event == "agent_handoff" {
			if data, ok := projected.Data.(map[string]any); ok {
				if assistantType, ok := data["to"].(string); ok {
					update.AssistantType = assistantType
				}
				if intentType, ok := data["intent"].(string); ok {
					update.IntentType = intentType
				}
			}
		}
		emit(projected.Event, projected.Data)
	}
	return update
}

func (l *Logic) runtimeContext(ctx context.Context) context.Context {
	if l == nil || l.svcCtx == nil {
		return ctx
	}
	return runtimectx.WithServices(ctx, l.svcCtx)
}

// CreateSession 创建新的 AI 会话。
func (l *Logic) CreateSession(ctx context.Context, userID uint64, title, scene string) (*model.AIChatSession, error) {
	if l.ChatDAO == nil {
		return nil, nil
	}

	s := &model.AIChatSession{
		ID:     uuid.NewString(),
		UserID: userID,
		Title:  title,
		Scene:  normalizeScene(scene),
	}
	if err := l.ChatDAO.CreateSession(ctx, s); err != nil {
		return nil, err
	}
	return s, nil
}

// ListSessions 列出用户的所有会话。
func (l *Logic) ListSessions(ctx context.Context, userID uint64, scene string) ([]model.AIChatSession, map[string][]model.AIChatMessage, error) {
	if l.ChatDAO == nil {
		return []model.AIChatSession{}, map[string][]model.AIChatMessage{}, nil
	}

	filtered, err := l.ChatDAO.ListSessions(ctx, userID, scene)
	if err != nil {
		return nil, nil, err
	}

	// 加载消息
	messagesBySession := make(map[string][]model.AIChatMessage, len(filtered))
	for _, session := range filtered {
		messages, err := l.ChatDAO.ListMessagesBySession(ctx, session.ID)
		if err != nil {
			return nil, nil, err
		}
		messagesBySession[session.ID] = messages
	}

	return filtered, messagesBySession, nil
}

// GetSession 获取会话详情。
func (l *Logic) GetSession(ctx context.Context, userID uint64, scene, sessionID string) (*model.AIChatSession, []model.AIChatMessage, error) {
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
	if err != nil {
		return false, err
	}
	if session == nil {
		return false, nil
	}

	if err := l.ChatDAO.DeleteSession(ctx, session.ID, userID); err != nil {
		return false, err
	}
	return true, nil
}

// GetMessageWithOwnership 获取消息并验证所有权。
//
// 验证消息所属会话是否属于当前用户。
// 返回消息或 nil（不存在或无权限时）。
func (l *Logic) GetMessageWithOwnership(ctx context.Context, userID uint64, messageID string) (*model.AIChatMessage, error) {
	if l.ChatDAO == nil {
		return nil, nil
	}

	// 获取消息
	message, err := l.ChatDAO.GetMessage(ctx, messageID)
	if err != nil || message == nil {
		return nil, err
	}

	// 验证会话所有权
	session, err := l.ChatDAO.GetSession(ctx, message.SessionID, userID, "")
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil // 无权限
	}

	return message, nil
}

// GetRun 获取 Run 状态。
func (l *Logic) GetRun(ctx context.Context, userID uint64, runID string) (*model.AIRun, *model.AIDiagnosisReport, error) {
	if l.RunDAO == nil {
		return nil, nil, nil
	}

	run, err := l.RunDAO.GetRun(ctx, runID)
	if err != nil || run == nil {
		return run, nil, err
	}

	// 验证用户权限
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, run.SessionID, userID, "")
		if err != nil {
			return nil, nil, err
		}
		if session == nil {
			return nil, nil, nil
		}
	}

	// 获取关联的诊断报告
	var report *model.AIDiagnosisReport
	if l.DiagnosisReportDAO != nil {
		report, err = l.DiagnosisReportDAO.GetReportByRunID(ctx, run.ID)
		if err != nil {
			return nil, nil, err
		}
	}

	return run, report, nil
}

// GetDiagnosisReport 获取诊断报告。
func (l *Logic) GetDiagnosisReport(ctx context.Context, userID uint64, reportID string) (*model.AIDiagnosisReport, error) {
	if l.DiagnosisReportDAO == nil {
		return nil, nil
	}

	report, err := l.DiagnosisReportDAO.GetReport(ctx, reportID)
	if err != nil || report == nil {
		return report, err
	}

	// 验证用户权限
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, report.SessionID, userID, "")
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, nil
		}
	}

	return report, nil
}

// buildSessionTitle 从首条消息生成会话标题。
func buildSessionTitle(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "New AI session"
	}
	if len(trimmed) > 48 {
		return trimmed[:48]
	}
	return trimmed
}

func normalizeScene(scene string) string {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return "ai"
	}
	return scene
}

func resolveChatScene(requestScene string, session *model.AIChatSession) string {
	if strings.TrimSpace(requestScene) != "" {
		return normalizeScene(requestScene)
	}
	if session != nil && strings.TrimSpace(session.Scene) != "" {
		return normalizeScene(session.Scene)
	}
	return "ai"
}

func (l *Logic) buildAugmentedMessage(ctx context.Context, scene string, sceneContext map[string]any, message string) string {
	scene = normalizeScene(scene)
	sections := []string{
		"[Hidden platform context for routing, tool selection, and safety policy]",
		"[Scene]",
		fmt.Sprintf("scene=%s", scene),
	}

	if payload := stringifyJSON(sceneContext); payload != "" && payload != "{}" {
		sections = append(sections,
			"",
			"[Scene Context]",
			fmt.Sprintf("scene_context=%s", payload),
		)
	}

	sceneSections := l.loadSceneAugmentation(ctx, scene)
	if len(sceneSections) > 0 {
		for _, section := range sceneSections {
			if len(section) == 0 {
				continue
			}
			sections = append(sections, "", strings.Join(section, "\n"))
		}
	}

	sections = append(sections,
		"",
		fmt.Sprintf("User request:\n%s", strings.TrimSpace(message)),
	)

	return strings.Join(sections, "\n")
}

func (l *Logic) loadSceneAugmentation(ctx context.Context, scene string) [][]string {
	if l == nil || l.svcCtx == nil || l.svcCtx.DB == nil || strings.TrimSpace(scene) == "" {
		return nil
	}

	var prompts []model.AIScenePrompt
	_ = l.svcCtx.DB.WithContext(ctx).
		Where("scene = ? AND is_active = ?", scene, true).
		Order("display_order ASC, id ASC").
		Find(&prompts).Error

	var config model.AISceneConfig
	hasConfig := l.svcCtx.DB.WithContext(ctx).
		Where("scene = ?", scene).
		First(&config).Error == nil

	sceneLines := make([]string, 0, 4)
	if len(prompts) > 0 {
		promptTexts := make([]string, 0, len(prompts))
		for _, item := range prompts {
			if text := strings.TrimSpace(item.PromptText); text != "" {
				promptTexts = append(promptTexts, text)
			}
		}
		if len(promptTexts) > 0 {
			sceneLines = append(sceneLines, fmt.Sprintf("scene_prompts=%s", stringifyJSON(promptTexts)))
		}
	}

	if hasConfig {
		if description := strings.TrimSpace(config.Description); description != "" {
			sceneLines = append(sceneLines, fmt.Sprintf("scene_description=%s", description))
		}
		if constraints := compactJSONString(config.ConstraintsJSON); constraints != "" {
			sceneLines = append(sceneLines, fmt.Sprintf("scene_constraints=%s", constraints))
		}
	}

	sections := make([][]string, 0, 2)
	if len(sceneLines) > 0 {
		sections = append(sections, append([]string{"[Scene Prompts & Constraints]"}, sceneLines...))
	}

	toolLines := make([]string, 0, 3)
	if hasConfig {
		if allowed := compactJSONString(config.AllowedToolsJSON); allowed != "" {
			toolLines = append(toolLines, fmt.Sprintf("allowed_tools=%s", allowed))
		}
		if blocked := compactJSONString(config.BlockedToolsJSON); blocked != "" {
			toolLines = append(toolLines, fmt.Sprintf("blocked_tools=%s", blocked))
		}
	}
	if len(toolLines) > 0 {
		toolLines = append(toolLines, "These tool constraints are mandatory.")
		sections = append(sections, append([]string{"[Tool Constraints]"}, toolLines...))
	}

	return sections
}

func stringifyJSON(value any) string {
	if value == nil {
		return ""
	}
	b, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(b)
}

func compactJSONString(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var payload any
	if err := json.Unmarshal([]byte(value), &payload); err != nil {
		return value
	}
	return stringifyJSON(payload)
}

// truncateString 截断字符串到指定长度。
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// =============================================================================
// 审批相关方法
// =============================================================================

// SubmitApprovalInput 提交审批结果的输入参数。
type SubmitApprovalInput struct {
	ApprovalID       string
	Approved         bool
	DisapproveReason string
	Comment          string
	UserID           uint64
}

// SubmitApprovalOutput 提交审批结果的输出。
type SubmitApprovalOutput struct {
	ApprovalID string `json:"approval_id"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

// SubmitApproval 提交审批结果。
//
// 该方法仅更新审批任务状态，不恢复执行。
// 用户在前端审批界面点击"批准"或"拒绝"后调用。
func (l *Logic) SubmitApproval(ctx context.Context, input SubmitApprovalInput) (*SubmitApprovalOutput, error) {
	if l.ApprovalDAO == nil {
		return nil, fmt.Errorf("approval service not initialized")
	}

	// 获取审批任务
	task, err := l.ApprovalDAO.GetByApprovalID(ctx, input.ApprovalID)
	if err != nil {
		return nil, fmt.Errorf("get approval task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("approval task not found")
	}

	// 检查状态
	if task.Status != "pending" {
		return &SubmitApprovalOutput{
			ApprovalID: input.ApprovalID,
			Status:     task.Status,
			Message:    fmt.Sprintf("approval already %s", task.Status),
		}, nil
	}

	// 检查是否过期
	if task.ExpiresAt != nil && task.ExpiresAt.Before(time.Now()) {
		_ = l.ApprovalDAO.UpdateStatus(ctx, input.ApprovalID, "expired", input.UserID, "", "")
		return &SubmitApprovalOutput{
			ApprovalID: input.ApprovalID,
			Status:     "expired",
			Message:    "approval has expired",
		}, nil
	}

	// 更新状态
	status := "approved"
	if !input.Approved {
		status = "rejected"
	}

	if err := l.ApprovalDAO.UpdateStatus(ctx, input.ApprovalID, status, input.UserID, input.DisapproveReason, input.Comment); err != nil {
		return nil, fmt.Errorf("update approval status: %w", err)
	}

	return &SubmitApprovalOutput{
		ApprovalID: input.ApprovalID,
		Status:     status,
		Message:    fmt.Sprintf("approval %s successfully", status),
	}, nil
}

// ResumeApprovalInput 恢复审批执行的输入参数。
type ResumeApprovalInput struct {
	SessionID  string
	ApprovalID string
	Approved   bool
	Reason     string
	Comment    string
	UserID     uint64
}

// ResumeApproval 恢复审批执行（SSE 流式）。
//
// 该方法通过 Runner.ResumeWithParams 恢复 AI Agent 执行，
// 并通过 SSE 流式返回后续执行结果。
func (l *Logic) ResumeApproval(ctx context.Context, input ResumeApprovalInput, emit EventEmitter) error {
	if l.ApprovalDAO == nil || l.CheckpointStore == nil || l.AIRouter == nil {
		emit(airuntime.NewErrorEvent("", fmt.Errorf("AI service not initialized")).Event, nil)
		return nil
	}

	// 获取审批任务
	task, err := l.ApprovalDAO.GetByApprovalID(ctx, input.ApprovalID)
	if err != nil {
		return fmt.Errorf("get approval task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("approval task not found")
	}

	// 验证用户权限
	if l.ChatDAO != nil {
		session, err := l.ChatDAO.GetSession(ctx, task.SessionID, input.UserID, "")
		if err != nil {
			return fmt.Errorf("verify session: %w", err)
		}
		if session == nil {
			return fmt.Errorf("session not found or no permission")
		}
	}

	// 更新审批状态
	if task.Status == "pending" {
		status := "approved"
		if !input.Approved {
			status = "rejected"
		}
		if err := l.ApprovalDAO.UpdateStatus(ctx, input.ApprovalID, status, input.UserID, input.Reason, input.Comment); err != nil {
			return fmt.Errorf("update approval status: %w", err)
		}
	}

	// 构建恢复参数
	approvalResult := map[string]any{
		"approved":          input.Approved,
		"disapprove_reason": input.Reason,
		"comment":           input.Comment,
		"approved_by":       input.UserID,
		"approved_at":       time.Now().Format(time.RFC3339),
	}

	// 发送 meta 事件
	meta := airuntime.NewMetaEvent(task.SessionID, task.RunID, 1)
	emit(meta.Event, meta.Data)

	// 创建 Runner 并恢复执行
	ctx = l.runtimeContext(ctx)
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           l.AIRouter,
		EnableStreaming: true,
		CheckPointStore: l.CheckpointStore,
	})

	// 使用 ResumeWithParams 恢复执行
	resumeParams := &adk.ResumeParams{
		Targets: map[string]any{
			task.ToolCallID: approvalResult,
		},
	}

	iterator, err := runner.ResumeWithParams(ctx, task.CheckpointID, resumeParams)
	if err != nil {
		return fmt.Errorf("resume execution: %w", err)
	}

	// 消费事件
	var assistantContent strings.Builder
	projector := airuntime.NewStreamProjector()

	for {
		event, ok := iterator.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			projected := projector.Fail(task.RunID, event.Err)
			emit(projected.Event, projected.Data)
			return nil
		}

		if event.Output != nil && event.Output.MessageOutput != nil && event.Output.MessageOutput.IsStreaming && event.Output.MessageOutput.MessageStream != nil {
			for {
				msg, err := event.Output.MessageOutput.MessageStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					projected := projector.Fail(task.RunID, err)
					emit(projected.Event, projected.Data)
					break
				}
				if msg == nil {
					continue
				}

				chunkEvent := adk.EventFromMessage(msg, nil, msg.Role, msg.ToolName)
				chunkEvent.AgentName = event.AgentName
				consumeProjectedEvents(projector.Consume(chunkEvent), emit, &assistantContent)
			}
			continue
		}

		consumeProjectedEvents(projector.Consume(event), emit, &assistantContent)
	}

	// 刷新缓冲区
	if remaining := projector.FlushBuffer(); len(remaining) > 0 {
		for _, e := range remaining {
			emit(e.Event, e.Data)
		}
	}
	done := projector.Finish(task.RunID)
	emit(done.Event, done.Data)

	return nil
}

// GetApproval 获取审批详情。
func (l *Logic) GetApproval(ctx context.Context, approvalID string, userID uint64) (*model.AIApprovalTask, error) {
	if l.ApprovalDAO == nil {
		return nil, nil
	}

	task, err := l.ApprovalDAO.GetByApprovalID(ctx, approvalID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, nil
	}

	// 验证用户权限
	if l.ChatDAO != nil && task.SessionID != "" {
		session, err := l.ChatDAO.GetSession(ctx, task.SessionID, userID, "")
		if err != nil {
			return nil, err
		}
		if session == nil {
			return nil, nil
		}
	}

	return task, nil
}

// ListPendingApprovals 列出用户的待处理审批。
func (l *Logic) ListPendingApprovals(ctx context.Context, userID uint64) ([]model.AIApprovalTask, error) {
	if l.ApprovalDAO == nil {
		return []model.AIApprovalTask{}, nil
	}

	return l.ApprovalDAO.ListPendingByUserID(ctx, userID, 50)
}
