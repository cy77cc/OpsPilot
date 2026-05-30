package http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/app/command"
	ssehandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/sse"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/streaming"
	"github.com/gin-gonic/gin"
)

type ChatRequest = aiv1.ChatRequest

type ChatCommandHandler interface {
	Handle(ctx context.Context, req *command.ChatRequest, emit logic.EventEmitter) error
}

type ChatHandler struct {
	commandHandler ChatCommandHandler
}

func NewChatHandler(commandHandler ChatCommandHandler) *ChatHandler {
	return &ChatHandler{commandHandler: commandHandler}
}

func (h *ChatHandler) HandleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	lastEventID := strings.TrimSpace(req.LastEventID)
	if queryLastEventID := strings.TrimSpace(c.Query("last_event_id")); queryLastEventID != "" {
		lastEventID = queryLastEventID
	}
	if headerLastEventID := strings.TrimSpace(c.GetHeader("Last-Event-ID")); headerLastEventID != "" {
		lastEventID = headerLastEventID
	}

	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	writer := ssehandler.NewSSEWriter(c.Writer)

	if h.commandHandler == nil {
		publicErr := streaming.MapStreamError(errors.New("AI service not initialized"))
		writeChatEvent(writer, c, "error", gin.H{
			"code":      publicErr.Code,
			"message":   publicErr.Message,
			"retryable": publicErr.Retryable,
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	if deadline, ok := ctx.Deadline(); ok {
		logger.L().Info("[AI-DEBUG] HandleChat: context timeout set",
			logger.String("deadline", deadline.Format(time.RFC3339)),
			logger.String("timeout", "2m"))
	}
	err := h.commandHandler.Handle(ctx, &command.ChatRequest{
		SessionID:       req.SessionID,
		ClientRequestID: req.ClientRequestID,
		LastEventID:     lastEventID,
		Message:         req.Message,
		Scene:           req.Scene,
		Context:         mapFromAny(req.Context),
		UserID:          httpx.UIDFromCtx(c),
	}, func(event string, data any) {
		writeChatEvent(writer, c, event, data)
	})
	if err != nil {
		publicErr := streaming.MapStreamError(err)
		writeChatEvent(writer, c, "error", gin.H{
			"code":      publicErr.Code,
			"message":   publicErr.Message,
			"retryable": publicErr.Retryable,
		})
		return
	}
}
