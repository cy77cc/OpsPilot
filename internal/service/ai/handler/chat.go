package handler

import (
	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Chat(c *gin.Context) {
	var req aiv1.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	c.Status(200)
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	writer := NewSSEWriter(c.Writer)

	if err := h.logic.Chat(c.Request.Context(), logic.ChatInput{
		SessionID: req.SessionID,
		Message:   req.Message,
		Scene:     req.Scene,
		Context:   mapFromAny(req.Context),
		UserID:    httpx.UIDFromCtx(c),
	}, func(event string, data any) {
		writeChatEvent(writer, c, event, data)
	}); err != nil {
		writeChatEvent(writer, c, "error", gin.H{
			"message": err.Error(),
		})
		return
	}
}

func writeChatEvent(writer *SSEWriter, c *gin.Context, event string, data any) {
	if err := writer.WriteEvent(event, data); err == nil {
		c.Writer.Flush()
	}
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return nil
	}
	if result, ok := value.(map[string]any); ok {
		return result
	}
	return nil
}
