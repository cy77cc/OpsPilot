package handler

import (
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	hostpluginlogic "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *hostpluginlogic.Service
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{
		service: hostpluginlogic.NewService(svcCtx),
	}
}

func (h *Handler) ListCatalog(c *gin.Context) {
	plugins, err := h.service.ListCatalog(c.Request.Context())
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": plugins, "total": len(plugins)})
}

func (h *Handler) ListHostInstances(c *gin.Context) {
	httpx.OK(c, gin.H{"list": []any{}, "total": 0, "host_id": c.Param("id")})
}

func (h *Handler) RunInstanceAction(c *gin.Context) {
	httpx.OK(c, gin.H{
		"instance_id": c.Param("instance_id"),
		"status":      "pending",
		"message":     "host plugin instance actions are not implemented yet",
	})
}

func (h *Handler) GetTask(c *gin.Context) {
	httpx.OK(c, gin.H{
		"task_id": c.Param("task_id"),
		"status":  "pending",
		"message": "host plugin tasks are not implemented yet",
	})
}

func (h *Handler) ListTaskLogs(c *gin.Context) {
	httpx.OK(c, gin.H{"list": []any{}, "total": 0, "task_id": c.Param("task_id")})
}
