package handler

import (
	"encoding/json"
	"strings"
	"time"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
)

// MessageRuntimeResponse 消息 runtime 响应结构。
type MessageRuntimeResponse struct {
	MessageID string         `json:"message_id"`
	Runtime   map[string]any `json:"runtime,omitempty"`
}

func (h *Handler) CreateSession(c *gin.Context) {
	var req aiv1.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	session, err := h.logic.CreateSession(c.Request.Context(), httpx.UIDFromCtx(c), req.Title, req.Scene)
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

func (h *Handler) ListSessions(c *gin.Context) {
	sessions, messagesBySession, err := h.logic.ListSessions(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	items := make([]gin.H, 0, len(sessions))
	for _, session := range sessions {
		summary := sessionSummaryFromModel(session)
		if messages, ok := messagesBySession[session.ID]; ok {
			messageItems := make([]gin.H, 0, len(messages))
			for _, message := range messages {
				messageItems = append(messageItems, gin.H{
					"id":             message.ID,
					"session_id_num": message.SessionIDNum,
					"role":           message.Role,
					"content":        message.Content,
					"status":         message.Status,
					"created_at":     formatTime(message.CreatedAt),
				})
			}
			summary["messages"] = messageItems
		}
		items = append(items, summary)
	}
	httpx.OK(c, items)
}

func (h *Handler) GetSession(c *gin.Context) {
	session, messages, err := h.logic.GetSession(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if session == nil {
		httpx.NotFound(c, "session not found")
		return
	}
	messageItems := make([]gin.H, 0, len(messages))
	for _, message := range messages {
		messageItems = append(messageItems, gin.H{
			"id":             message.ID,
			"session_id_num": message.SessionIDNum,
			"role":           message.Role,
			"content":        message.Content,
			"status":         message.Status,
			"has_runtime":    message.RuntimeJSON != "",
			"created_at":     formatTime(message.CreatedAt),
		})
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

// GetMessageRuntime 获取单条消息的运行时状态。
//
// 权限验证：检查消息所属会话是否属于当前用户。
func (h *Handler) GetMessageRuntime(c *gin.Context) {
	messageID := c.Param("id")
	userID := httpx.UIDFromCtx(c)

	// 获取消息并验证权限
	message, err := h.logic.GetMessageWithOwnership(c.Request.Context(), userID, messageID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if message == nil {
		httpx.NotFound(c, "消息不存在或无权限访问")
		return
	}

	// 解析 runtime JSON
	var runtime map[string]any
	if message.RuntimeJSON != "" {
		if err := json.Unmarshal([]byte(message.RuntimeJSON), &runtime); err != nil {
			// JSON 解析失败，返回 null
			runtime = nil
		}
	}

	httpx.OK(c, MessageRuntimeResponse{
		MessageID: messageID,
		Runtime:   runtime,
	})
}

func (h *Handler) DeleteSession(c *gin.Context) {
	ok, err := h.logic.DeleteSession(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id"))
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
