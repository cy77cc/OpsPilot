package handler

import (
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

func (h *AIHandler) sceneTools(c *gin.Context) {
	uid, ok := uidFromContext(c)
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	scene := c.Param("scene")
	meta, exists := sceneMetaByKey(scene)
	if !exists {
		httpx.Fail(c, xcode.NotFound, "scene not found")
		return
	}
	recommended := h.sceneRecommendedTools(scene)
	out := make([]any, 0, len(recommended))
	for _, item := range recommended {
		if h.hasPermission(uid, item.Permission) {
			out = append(out, item)
		}
	}
	httpx.OK(c, gin.H{
		"scene":         meta.Scene,
		"description":   meta.Description,
		"keywords":      meta.Keywords,
		"context_hints": meta.ContextHints,
		"tools":         out,
	})
}
