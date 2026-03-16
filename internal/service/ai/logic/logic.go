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
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/agents"
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
	UserID    uint64
}

// Logic 封装 AI 模块的核心业务逻辑。
type Logic struct {
	svcCtx             *svc.ServiceContext
	ChatDAO            *aidao.AIChatDAO
	RunDAO             *aidao.AIRunDAO
	DiagnosisReportDAO *aidao.AIDiagnosisReportDAO
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
		AIRouter:           aiRouter,
	}
}

// Chat 执行一次 AI 对话，通过 SSE 流式返回结果。
//
// 流程:
//  1. 创建或复用 Session
//  2. 创建 User Message 和 Run 记录
//  3. 发送 init 事件
//  4. 调用 AIRouter.Run() 获取 AsyncIterator
//  5. 消费事件，转换为 SSE 推送
//  6. 持久化结果
func (l *Logic) Chat(ctx context.Context, input ChatInput, emit EventEmitter) error {
	if l.ChatDAO == nil || l.RunDAO == nil || l.AIRouter == nil {
		emit("error", map[string]any{"message": "AI service not initialized"})
		return nil
	}

	// Step 1: 创建或复用 Session
	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	session, err := l.ChatDAO.GetSession(ctx, sessionID, input.UserID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		if err := l.ChatDAO.CreateSession(ctx, &model.AIChatSession{
			ID:     sessionID,
			UserID: input.UserID,
			Scene:  "ai",
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
	}
	if err := l.RunDAO.CreateRun(ctx, run); err != nil {
		return fmt.Errorf("create run: %w", err)
	}

	// Step 4: 发送 init 事件
	emit("init", map[string]any{
		"session_id": sessionID,
		"run_id":     run.ID,
	})

	// Step 5: 调用 AIRouter
	agentInput := &adk.AgentInput{
		Messages: []adk.Message{
			schema.UserMessage(input.Message),
		},
		EnableStreaming: true,
	}

	iterator := l.AIRouter.Run(ctx, agentInput)

	// Step 6: 消费事件
	var (
		assistantMessage *model.AIChatMessage
		assistantContent strings.Builder
		intentEmitted    bool
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

	emit("status", map[string]any{"status": "running"})

	for {
		event, ok := iterator.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			emit("error", map[string]any{"message": event.Err.Error()})
			_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{
				Status:       "failed",
				ErrorMessage: event.Err.Error(),
			})
			return nil
		}

		// 发送 intent 事件（首次进入子 Agent 时）
		if !intentEmitted && event.AgentName != "" && event.AgentName != "OpsPilotAgent" {
			intentType := mapAgentNameToIntentType(event.AgentName)
			emit("intent", map[string]any{
				"intent_type":    intentType,
				"assistant_type": event.AgentName,
			})
			intentEmitted = true

			// 更新 Run 的 IntentType
			_ = l.RunDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{
				IntentType:    intentType,
				AssistantType: event.AgentName,
			})
		}

		// 处理消息输出（Token 级流式）
		if event.Output != nil && event.Output.MessageOutput != nil {
			msgOutput := event.Output.MessageOutput

			if msgOutput.IsStreaming && msgOutput.MessageStream != nil {
				// 消费 MessageStream，逐 token 发送
				for {
					msg, err := msgOutput.MessageStream.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						emit("error", map[string]any{"message": err.Error()})
						break
					}

					if msg != nil && msg.Content != "" {
						assistantContent.WriteString(msg.Content)
						emit("delta", map[string]any{"contentChunk": msg.Content})
					}
				}
			} else if msgOutput.Message != nil {
				// 非流式消息
				content := msgOutput.Message.Content
				if content != "" {
					assistantContent.WriteString(content)
					emit("delta", map[string]any{"contentChunk": content})
				}
			}
		}

		// 处理 Action
		if event.Action != nil {
			// Exit: Agent 执行完成
			if event.Action.Exit {
				emit("status", map[string]any{"status": "completed"})
			}
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
	emit("done", map[string]any{
		"run_id": run.ID,
		"status": "completed",
	})

	return nil
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
		Scene:  scene,
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

	rows, err := l.ChatDAO.ListSessions(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	// 按场景过滤
	filtered := make([]model.AIChatSession, 0, len(rows))
	for _, row := range rows {
		if scene == "" || row.Scene == scene {
			filtered = append(filtered, row)
		}
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
func (l *Logic) GetSession(ctx context.Context, userID uint64, sessionID string) (*model.AIChatSession, []model.AIChatMessage, error) {
	if l.ChatDAO == nil {
		return nil, nil, nil
	}

	session, err := l.ChatDAO.GetSession(ctx, sessionID, userID)
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

	session, err := l.ChatDAO.GetSession(ctx, sessionID, userID)
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
		session, err := l.ChatDAO.GetSession(ctx, run.SessionID, userID)
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
		session, err := l.ChatDAO.GetSession(ctx, report.SessionID, userID)
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

// truncateString 截断字符串到指定长度。
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
