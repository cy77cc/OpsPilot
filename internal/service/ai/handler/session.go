package handler

import (
	"context"
	"strings"
	"time"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateSession(c *gin.Context) {
	var req aiv1.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	session, err := h.logic.CreateSession(c.Request.Context(), httpx.UIDFromCtx(c), req.Title, req.Scene)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if session == nil {
		httpx.OK(c, gin.H{})
		return
	}
	httpx.OK(c, sessionSummaryFromModel(*session))
}

func (h *Handler) ListSessions(c *gin.Context) {
	sessions, messagesBySession, err := h.logic.ListSessions(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	runBySessionAndAssistantMessageID := h.runBySessionAndAssistantMessageID(c.Request.Context(), sessions)
	items := make([]gin.H, 0, len(sessions))
	for _, session := range sessions {
		summary := sessionSummaryFromModel(session)
		if messages, ok := messagesBySession[session.ID]; ok {
			messageItems := make([]gin.H, 0, len(messages))
			for _, message := range messages {
				item := sessionMessageItem(message, runBySessionAndAssistantMessageID[session.ID][message.ID])
				messageItems = append(messageItems, item)
			}
			summary["messages"] = messageItems
		}
		items = append(items, summary)
	}
	httpx.OK(c, items)
}

func (h *Handler) GetSession(c *gin.Context) {
	session, messages, err := h.logic.GetSession(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if session == nil {
		httpx.NotFound(c, "session not found")
		return
	}
	runByAssistantMessageID := h.runByAssistantMessageID(c.Request.Context(), session.ID)
	messageItems := make([]gin.H, 0, len(messages))
	for _, message := range messages {
		item := sessionMessageItem(message, runByAssistantMessageID[message.ID])
		messageItems = append(messageItems, item)
	}
	httpx.OK(c, gin.H{
		"id":         session.ID,
		"title":      session.Title,
		"scene":      session.Scene,
		"messages":   messageItems,
		"created_at": formatTime(session.CreatedAt),
		"updated_at": formatTime(session.UpdatedAt),
	})
}

func (h *Handler) DeleteSession(c *gin.Context) {
	ok, err := h.logic.DeleteSession(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !ok {
		httpx.NotFound(c, "session not found")
		return
	}
	httpx.OK(c, nil)
}

func sessionSummaryFromModel(session model.AIChatSession) gin.H {
	return gin.H{
		"id":         session.ID,
		"title":      session.Title,
		"scene":      session.Scene,
		"created_at": formatTime(session.CreatedAt),
		"updated_at": formatTime(session.UpdatedAt),
	}
}

const terminalAssistantErrorMessage = "生成中断，请稍后重试。"

func sessionMessageItem(message model.AIChatMessage, run *model.AIRun) gin.H {
	item := gin.H{
		"id":             message.ID,
		"session_id_num": message.SessionIDNum,
		"role":           message.Role,
		"status":         message.Status,
		"created_at":     formatTime(message.CreatedAt),
		"content":        message.Content,
	}
	if run != nil {
		item["run_id"] = run.ID
		if isTerminalAssistantRun(run.Status) {
			item["status"] = "error"
			item["error_message"] = terminalAssistantErrorMessage
		}
	}
	return item
}

func isTerminalAssistantRun(status string) bool {
	switch strings.TrimSpace(status) {
	case "failed", "failed_runtime", "expired":
		return true
	default:
		return false
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (h *Handler) runByAssistantMessageID(ctx context.Context, sessionID string) map[string]*model.AIRun {
	result := map[string]*model.AIRun{}
	if h == nil || h.logic == nil || h.logic.RunDAO == nil {
		return result
	}
	runs, err := h.logic.RunDAO.ListBySession(ctx, sessionID)
	if err != nil {
		return result
	}
	for _, run := range runs {
		if strings.TrimSpace(run.AssistantMessageID) != "" {
			runCopy := run
			result[run.AssistantMessageID] = &runCopy
		}
	}
	return result
}

func (h *Handler) runBySessionAndAssistantMessageID(ctx context.Context, sessions []model.AIChatSession) map[string]map[string]*model.AIRun {
	result := map[string]map[string]*model.AIRun{}
	if h == nil || h.logic == nil || h.logic.RunDAO == nil || len(sessions) == 0 {
		return result
	}
	sessionIDs := make([]string, 0, len(sessions))
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.ID)
		result[session.ID] = map[string]*model.AIRun{}
	}
	runs, err := h.logic.RunDAO.ListBySessionIDs(ctx, sessionIDs)
	if err != nil {
		return result
	}
	for _, run := range runs {
		if strings.TrimSpace(run.AssistantMessageID) == "" {
			continue
		}
		if _, ok := result[run.SessionID]; !ok {
			result[run.SessionID] = map[string]*model.AIRun{}
		}
		runCopy := run
		result[run.SessionID][run.AssistantMessageID] = &runCopy
	}
	return result
}
