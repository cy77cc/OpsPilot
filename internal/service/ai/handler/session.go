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
				item := sessionMessageItem(message)
				if runID := runBySessionAndAssistantMessageID[session.ID][message.ID]; runID != "" {
					item["run_id"] = runID
				}
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
		item := sessionMessageItem(message)
		if runID := runByAssistantMessageID[message.ID]; runID != "" {
			item["run_id"] = runID
		}
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

func sessionMessageItem(message model.AIChatMessage) gin.H {
	item := gin.H{
		"id":             message.ID,
		"session_id_num": message.SessionIDNum,
		"role":           message.Role,
		"status":         message.Status,
		"created_at":     formatTime(message.CreatedAt),
	}
	if message.Role != "assistant" {
		item["content"] = message.Content
	}
	return item
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (h *Handler) runByAssistantMessageID(ctx context.Context, sessionID string) map[string]string {
	result := map[string]string{}
	if h == nil || h.logic == nil || h.logic.RunDAO == nil {
		return result
	}
	runs, err := h.logic.RunDAO.ListBySession(ctx, sessionID)
	if err != nil {
		return result
	}
	for _, run := range runs {
		if strings.TrimSpace(run.AssistantMessageID) != "" {
			result[run.AssistantMessageID] = run.ID
		}
	}
	return result
}

func (h *Handler) runBySessionAndAssistantMessageID(ctx context.Context, sessions []model.AIChatSession) map[string]map[string]string {
	result := map[string]map[string]string{}
	if h == nil || h.logic == nil || h.logic.RunDAO == nil || len(sessions) == 0 {
		return result
	}
	sessionIDs := make([]string, 0, len(sessions))
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.ID)
		result[session.ID] = map[string]string{}
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
			result[run.SessionID] = map[string]string{}
		}
		result[run.SessionID][run.AssistantMessageID] = run.ID
	}
	return result
}
