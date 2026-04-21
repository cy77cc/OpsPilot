package http

import (
	"context"
	"errors"
	"net/http"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	ssehandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/sse"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/streaming"
	"github.com/gin-gonic/gin"
)

// FormAssistStreamer defines the interface for form assistance logic.
type FormAssistStreamer interface {
	StreamAssist(ctx context.Context, input logic.FormAssistInput, emit logic.EventEmitter) error
}

// FormAssistHandler handles form assistance requests via SSE.
type FormAssistHandler struct {
	streamer FormAssistStreamer
}

// NewFormAssistHandler creates a new FormAssistHandler.
func NewFormAssistHandler(streamer FormAssistStreamer) *FormAssistHandler {
	return &FormAssistHandler{streamer: streamer}
}

// HandleAssist handles the form assistance request.
func (h *FormAssistHandler) HandleAssist(c *gin.Context) {
	var req aiv1.FormAssistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	writer := ssehandler.NewSSEWriter(c.Writer)

	if h.streamer == nil {
		publicErr := streaming.MapStreamError(errors.New("AI service not initialized"))
		h.writeEvent(writer, c, "error", gin.H{
			"code":      publicErr.Code,
			"message":   publicErr.Message,
			"retryable": publicErr.Retryable,
		})
		return
	}

	err := h.streamer.StreamAssist(c.Request.Context(), logic.FormAssistInput{
		Scene:       req.Scene,
		UserPrompt:  req.UserPrompt,
		FieldMeta:   req.FieldMeta,
		FormContext: req.FormContext,
		UserID:      httpx.UIDFromCtx(c),
	}, func(event string, data any) {
		h.writeEvent(writer, c, event, data)
	})

	if err != nil {
		publicErr := streaming.MapStreamError(err)
		h.writeEvent(writer, c, "error", gin.H{
			"code":      publicErr.Code,
			"message":   publicErr.Message,
			"retryable": publicErr.Retryable,
		})
		return
	}
}

func (h *FormAssistHandler) writeEvent(writer *ssehandler.SSEWriter, c *gin.Context, event string, data any) {
	if err := writer.WriteEvent(event, data); err == nil {
		c.Writer.Flush()
	}
}
