package handler

import (
	"fmt"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
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

	if err := h.logic.Chat(c.Request.Context(), logic.ChatInput{
		SessionID: req.SessionID,
		Message:   req.Message,
		UserID:    httpx.UIDFromCtx(c),
	}, func(event string, data any) {
		writeChatEvent(c, event, data)
	}); err != nil {
		httpx.ServerErr(c, err)
		return
	}
}

func writeChatEvent(c *gin.Context, event string, data any) {
	payload, err := airuntime.EncodePublicEvent(event, data)
	if err != nil {
		payload = []byte(fmt.Sprintf(`{"event":"error","data":{"message":%q}}`, err.Error()))
	}
	_, _ = c.Writer.Write([]byte("data: "))
	_, _ = c.Writer.Write(payload)
	_, _ = c.Writer.Write([]byte("\n\n"))
	c.Writer.Flush()
}
