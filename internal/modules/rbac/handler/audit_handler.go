package handler

import (
	"log"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/gin-gonic/gin"
)

// RecordMigrationEvent 记录迁移事件。
//
// @Summary 记录迁移事件
// @Description 记录前端权限迁移相关事件用于审计
// @Tags RBAC
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body object{eventType=string,fromPath=string,toPath=string,action=string,status=string,durationMs=int} true "迁移事件请求"
// @Success 200 {object} httpx.Response{data=object{accepted=bool}}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /rbac/migration/events [post]
func (h *Handler) RecordMigrationEvent(c *gin.Context) {
	uid, ok := c.Get("uid")
	if !ok {
		httpx.Fail(c, xcode.Unauthorized, "unauthorized")
		return
	}
	var req struct {
		EventType  string `json:"eventType" binding:"required"`
		FromPath   string `json:"fromPath"`
		ToPath     string `json:"toPath"`
		Action     string `json:"action"`
		Status     string `json:"status"`
		DurationMs int64  `json:"durationMs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	userID := httpx.ToUint64(uid)
	timestamp := time.Now().UTC().Format(time.RFC3339)
	log.Printf("rbac migration event=%s actor=%d from=%s to=%s action=%s status=%s duration_ms=%d timestamp=%s",
		strings.TrimSpace(req.EventType),
		userID,
		strings.TrimSpace(req.FromPath),
		strings.TrimSpace(req.ToPath),
		strings.TrimSpace(req.Action),
		strings.TrimSpace(req.Status),
		req.DurationMs,
		timestamp,
	)
	httpx.OK(c, gin.H{"accepted": true})
}
