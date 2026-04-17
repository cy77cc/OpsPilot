package http

import (
	"context"
	"errors"
	"net/http"
	"strings"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/app/command"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	ssehandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/sse"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
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
		writeChatEvent(writer, c, "error", gin.H{"message": "AI service not initialized"})
		return
	}

	err := h.commandHandler.Handle(c.Request.Context(), &command.ChatRequest{
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
		if errors.Is(err, aidao.ErrRunEventCursorExpired) {
			writeChatEvent(writer, c, "error", gin.H{
				"code":    "AI_STREAM_CURSOR_EXPIRED",
				"message": "last_event_id is too old; refresh the stream snapshot",
			})
			return
		}
		writeChatEvent(writer, c, "error", gin.H{"message": err.Error()})
		return
	}
}
