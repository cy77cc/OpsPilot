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
		httpx.ServerErr(c, err)
		return
	}
}

func writeChatEvent(writer *SSEWriter, c *gin.Context, event string, data any) {
	normalizedEvent, normalizedData := normalizeChatEvent(event, data)
	if err := writer.WriteEvent(normalizedEvent, normalizedData); err == nil {
		c.Writer.Flush()
	}
}

func normalizeChatEvent(event string, data any) (string, any) {
	if event != "init" {
		return event, data
	}

	payload, ok := data.(map[string]any)
	if !ok {
		return "meta", map[string]any{"turn": 1}
	}

	normalized := make(map[string]any, len(payload)+1)
	for key, value := range payload {
		normalized[key] = value
	}
	if _, exists := normalized["turn"]; !exists {
		normalized["turn"] = 1
	}
	return "meta", normalized
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
