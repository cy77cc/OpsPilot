package handler

import (
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
	items := make([]gin.H, 0, len(sessions))
	for _, session := range sessions {
		summary := sessionSummaryFromModel(session)
		if messages, ok := messagesBySession[session.ID]; ok {
			messageItems := make([]gin.H, 0, len(messages))
			for _, message := range messages {
				messageItems = append(messageItems, gin.H{
					"id":         message.ID,
					"role":       message.Role,
					"content":    message.Content,
					"status":     message.Status,
					"created_at": formatTime(message.CreatedAt),
				})
			}
			summary["messages"] = messageItems
		}
		items = append(items, summary)
	}
	httpx.OK(c, items)
}

func (h *Handler) GetSession(c *gin.Context) {
	session, messages, err := h.logic.GetSession(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if session == nil {
		httpx.NotFound(c, "session not found")
		return
	}
	messageItems := make([]gin.H, 0, len(messages))
	for _, message := range messages {
		messageItems = append(messageItems, gin.H{
			"id":         message.ID,
			"role":       message.Role,
			"content":    message.Content,
			"status":     message.Status,
			"created_at": formatTime(message.CreatedAt),
		})
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

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
