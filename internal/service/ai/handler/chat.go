package handler

import (
	"encoding/json"
	"fmt"

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

	if err := h.logic.Chat(c.Request.Context(), logic.ChatInput{
		SessionID: req.SessionID,
		Message:   req.Message,
		Scene:     req.Scene,
		Context:   mapFromAny(req.Context),
		UserID:    httpx.UIDFromCtx(c),
	}, func(event string, data any) {
		writeChatEvent(c, event, data)
	}); err != nil {
		httpx.ServerErr(c, err)
		return
	}
}

// writeChatEvent 写入标准 SSE 事件。
//
// 格式:
//
//	event: <event_type>
//	data: <json_data>
//
//	(空行)
func writeChatEvent(c *gin.Context, event string, data any) {
	// 写入 event 行
	_, _ = c.Writer.Write([]byte("event: "))
	_, _ = c.Writer.Write([]byte(event))
	_, _ = c.Writer.Write([]byte("\n"))

	// 写入 data 行
	payload, err := json.Marshal(data)
	if err != nil {
		payload = []byte(fmt.Sprintf(`{"message":%q}`, err.Error()))
	}
	_, _ = c.Writer.Write([]byte("data: "))
	_, _ = c.Writer.Write(payload)
	_, _ = c.Writer.Write([]byte("\n\n"))

	c.Writer.Flush()
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
