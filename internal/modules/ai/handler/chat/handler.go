// Package chathandler 实现 AI 聊天的 HTTP Handler。
package chathandler

import (
	"errors"
	"strings"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	ssehandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/sse"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	runtimecontext "github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/context"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	svc *Service
}

var chatContextBudget = runtimecontext.Budget{
	Pinned:  1,
	Recent:  12,
	History: 6,
}

func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func (h *HTTPHandler) Chat(c *gin.Context) {
	var req aiv1.ChatRequest
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
	req.LastEventID = lastEventID

	c.Status(200)
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	writer := ssehandler.NewSSEWriter(c.Writer)

	if strings.TrimSpace(req.LastEventID) != "" {
		if err := h.svc.ValidateReplayCursor(c.Request.Context(), req.SessionID, req.ClientRequestID, req.LastEventID); err != nil {
			if errors.Is(err, aidao.ErrRunEventCursorExpired) {
				writeChatEvent(writer, c, "error", gin.H{
					"code":    "AI_STREAM_CURSOR_EXPIRED",
					"message": "last_event_id is too old; refresh the stream snapshot",
				})
				return
			}
			writeChatEvent(writer, c, "error", gin.H{
				"message": err.Error(),
			})
			return
		}
	}

	if err := h.svc.Chat(c.Request.Context(), logic.ChatInput{
		SessionID:       req.SessionID,
		ClientRequestID: req.ClientRequestID,
		LastEventID:     req.LastEventID,
		Message:         req.Message,
		Scene:           req.Scene,
		Context:         mapFromAny(req.Context),
		Budget:          chatContextBudget,
		UserID:          httpx.UIDFromCtx(c),
	}, func(event string, data any) {
		writeChatEvent(writer, c, event, data)
	}); err != nil {
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
