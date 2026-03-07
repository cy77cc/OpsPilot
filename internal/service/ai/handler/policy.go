package handler

import (
	"context"

	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/gin-gonic/gin"
)

func (h *AIHandler) toolPolicy(ctx context.Context, meta tools.ToolMeta, params map[string]any) error {
	return h.control.ToolPolicy(ctx, meta, params)
}

func (h *AIHandler) hasPermission(uid uint64, code string) bool {
	return h.control.HasPermission(uid, code)
}

func (h *AIHandler) isAdmin(uid uint64) bool {
	return h.control.IsAdmin(uid)
}

func uidFromContext(c *gin.Context) (uint64, bool) {
	v, ok := c.Get("uid")
	if !ok {
		return 0, false
	}
	switch x := v.(type) {
	case uint:
		return uint64(x), true
	case uint64:
		return x, true
	case int:
		return uint64(x), true
	case int64:
		return uint64(x), true
	case float64:
		return uint64(x), true
	default:
		return 0, false
	}
}
