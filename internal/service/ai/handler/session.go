package handler

import (
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) CreateSession(c *gin.Context) {
	var req struct {
		Title string `json:"title"`
		Scene string `json:"scene"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	session := &model.AIChatSession{
		ID:     uuid.NewString(),
		UserID: httpx.UIDFromCtx(c),
		Title:  req.Title,
		Scene:  req.Scene,
	}
	if h.deps.ChatDAO != nil {
		if err := h.deps.ChatDAO.CreateSession(c.Request.Context(), session); err != nil {
			httpx.ServerErr(c, err)
			return
		}
	}
	httpx.OK(c, gin.H{"session": sessionSummaryFromModel(*session)})
}

func (h *Handler) ListSessions(c *gin.Context) {
	sessions := []model.AIChatSession{}
	if h.deps.ChatDAO != nil {
		rows, err := h.deps.ChatDAO.ListSessions(c.Request.Context(), httpx.UIDFromCtx(c))
		if err != nil {
			httpx.ServerErr(c, err)
			return
		}
		sessions = rows
	}
	items := make([]gin.H, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, sessionSummaryFromModel(session))
	}
	httpx.OK(c, gin.H{"sessions": items})
}

func (h *Handler) GetSession(c *gin.Context) {
	if h.deps.ChatDAO == nil {
		httpx.OK(c, gin.H{"session": gin.H{}})
		return
	}
	session, err := h.deps.ChatDAO.GetSession(c.Request.Context(), c.Param("id"), httpx.UIDFromCtx(c))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if session == nil {
		httpx.NotFound(c, "session not found")
		return
	}
	messages, err := h.deps.ChatDAO.ListMessagesBySession(c.Request.Context(), session.ID)
	if err != nil {
		httpx.ServerErr(c, err)
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
	httpx.OK(c, gin.H{"session": gin.H{
		"id":         session.ID,
		"title":      session.Title,
		"scene":      session.Scene,
		"messages":   messageItems,
		"created_at": formatTime(session.CreatedAt),
		"updated_at": formatTime(session.UpdatedAt),
	}})
}

func (h *Handler) DeleteSession(c *gin.Context) {
	if h.deps.ChatDAO != nil {
		if err := h.deps.ChatDAO.DeleteSession(c.Request.Context(), c.Param("id"), httpx.UIDFromCtx(c)); err != nil {
			httpx.ServerErr(c, err)
			return
		}
	}
	httpx.OK(c, gin.H{"deleted": true})
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
