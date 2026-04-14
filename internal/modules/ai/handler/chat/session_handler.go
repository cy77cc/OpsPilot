package chathandler

import (
	"strings"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/gin-gonic/gin"
)

func (h *HTTPHandler) CreateSession(c *gin.Context) {
	var req aiv1.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	session, err := h.svc.CreateSession(c.Request.Context(), httpx.UIDFromCtx(c), req.Title, req.Scene)
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

func (h *HTTPHandler) ListSessions(c *gin.Context) {
	summaries, err := h.svc.ListSessions(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	sessions := make([]ai.AIChatSession, 0, len(summaries))
	for _, summary := range summaries {
		sessions = append(sessions, summary.Session)
	}
	runBySessionAndAssistantMessageID := h.runBySessionAndAssistantMessageID(c.Request.Context(), sessions)
	items := make([]gin.H, 0, len(summaries))
	for _, summary := range summaries {
		item := sessionSummaryFromModel(summary.Session)
		if summary.LastMessage != nil {
			run := runBySessionAndAssistantMessageID[summary.Session.ID][summary.LastMessage.ID]
			lastMessage := sessionMessageItem(*summary.LastMessage, run)
			mergeResumableCredentials(lastMessage, h.buildResumableCredentials(c.Request.Context(), run))
			item["last_message"] = lastMessage
		}
		items = append(items, item)
	}
	httpx.OK(c, items)
}

func (h *HTTPHandler) GetSession(c *gin.Context) {
	session, messages, err := h.svc.GetSession(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")), c.Param("id"))
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
		run := runByAssistantMessageID[message.ID]
		item := sessionMessageItem(message, run)
		mergeResumableCredentials(item, h.buildResumableCredentials(c.Request.Context(), run))
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

func (h *HTTPHandler) DeleteSession(c *gin.Context) {
	ok, err := h.svc.DeleteSession(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id"))
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
