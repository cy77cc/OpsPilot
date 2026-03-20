package handler

import (
	"encoding/json"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetRunProjection(c *gin.Context) {
	projection, err := h.logic.GetRunProjection(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("runId"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if projection == nil {
		httpx.NotFound(c, "projection not found")
		return
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(projection.ProjectionJSON), &payload); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, payload)
}

func (h *Handler) GetRunContent(c *gin.Context) {
	content, err := h.logic.GetRunContent(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if content == nil {
		httpx.NotFound(c, "content not found")
		return
	}
	httpx.OK(c, gin.H{
		"id":           content.ID,
		"run_id":       content.RunID,
		"session_id":   content.SessionID,
		"content_kind": content.ContentKind,
		"encoding":     content.Encoding,
		"summary_text": content.SummaryText,
		"body_text":    content.BodyText,
		"body_json":    content.BodyJSON,
		"size_bytes":   content.SizeBytes,
		"created_at":   formatTime(content.CreatedAt),
	})
}
