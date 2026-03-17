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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/agents"
	aicheckpoint "github.com/cy77cc/OpsPilot/internal/ai/checkpoint"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
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
	CheckpointStore    adk.CheckPointStore
	AIRouter           adk.ResumableAgent
}

// NewAILogic 创建 Logic 实例。
func NewAILogic(svcCtx *svc.ServiceContext) *Logic {
	if svcCtx == nil || svcCtx.DB == nil {
		return &Logic{}
	}

	aiRouter, err := agents.NewRouter(svcCtx.GetContext())
	if err != nil {
		return &Logic{}
	}

	return &Logic{
		svcCtx:             svcCtx,
		ChatDAO:            aidao.NewAIChatDAO(svcCtx.DB),
		RunDAO:             aidao.NewAIRunDAO(svcCtx.DB),
		DiagnosisReportDAO: aidao.NewAIDiagnosisReportDAO(svcCtx.DB),
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
			projected := projector.Fail(run.ID, event.Err)
			emit(projected.Event, projected.Data)
			_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{
				Status:       "failed",
				ErrorMessage: event.Err.Error(),
			})
			return nil
		}

		if event.Output != nil && event.Output.MessageOutput != nil && event.Output.MessageOutput.IsStreaming && event.Output.MessageOutput.MessageStream != nil {
			for {
				msg, err := event.Output.MessageOutput.MessageStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
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
	if err := l.ChatDAO.UpdateMessage(ctx, assistantMessage.ID, map[string]any{
		"content": finalContent,
		"status":  "done",
	}); err != nil {
		return fmt.Errorf("update assistant message: %w", err)
	}

	// 更新 Run 状态
	runStatus := aidao.AIRunStatusUpdate{
		Status:             "completed",
		AssistantMessageID: assistantMessage.ID,
		ProgressSummary:    truncateString(finalContent, 500),
	}
	if err := l.RunDAO.UpdateRunStatus(ctx, run.ID, runStatus); err != nil {
		return fmt.Errorf("update run status: %w", err)
	}

	// Step 8: 发送 done 事件
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

// mapAgentNameToIntentType 将 Agent 名称映射为 intent_type。
func mapAgentNameToIntentType(agentName string) string {
	switch agentName {
	case "QAAgent":
		return "qa"
	case "DiagnosisAgent":
		return "diagnosis"
	case "ChangeAgent":
		return "change"
	default:
		return "unknown"
	}
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
	hidden := []string{fmt.Sprintf("scene=%s", scene)}

	if payload := stringifyJSON(sceneContext); payload != "" && payload != "{}" {
		hidden = append(hidden, fmt.Sprintf("scene_context=%s", payload))
	}

	if augmentation := l.loadSceneAugmentation(ctx, scene); augmentation != "" {
		hidden = append(hidden, augmentation)
	}

	return strings.Join([]string{
		"[Hidden scene-aware context for routing and tool selection]",
		strings.Join(hidden, "\n"),
		"",
		fmt.Sprintf("User request:\n%s", strings.TrimSpace(message)),
	}, "\n")
}

func (l *Logic) loadSceneAugmentation(ctx context.Context, scene string) string {
	if l == nil || l.svcCtx == nil || l.svcCtx.DB == nil || strings.TrimSpace(scene) == "" {
		return ""
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

	lines := make([]string, 0, 4)
	if len(prompts) > 0 {
		promptTexts := make([]string, 0, len(prompts))
		for _, item := range prompts {
			if text := strings.TrimSpace(item.PromptText); text != "" {
				promptTexts = append(promptTexts, text)
			}
		}
		if len(promptTexts) > 0 {
			lines = append(lines, fmt.Sprintf("scene_prompts=%s", stringifyJSON(promptTexts)))
		}
	}

	if hasConfig {
		if description := strings.TrimSpace(config.Description); description != "" {
			lines = append(lines, fmt.Sprintf("scene_description=%s", description))
		}
		if constraints := compactJSONString(config.ConstraintsJSON); constraints != "" {
			lines = append(lines, fmt.Sprintf("scene_constraints=%s", constraints))
		}
		if allowed := compactJSONString(config.AllowedToolsJSON); allowed != "" {
			lines = append(lines, fmt.Sprintf("allowed_tools=%s", allowed))
		}
		if blocked := compactJSONString(config.BlockedToolsJSON); blocked != "" {
			lines = append(lines, fmt.Sprintf("blocked_tools=%s", blocked))
		}
	}

	return strings.Join(lines, "\n")
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
