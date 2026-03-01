package handler

import (
	"time"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	hostlogic "github.com/cy77cc/k8s-manage/internal/service/host/logic"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

func (h *Handler) List(c *gin.Context) {
	list, err := h.hostService.List(c.Request.Context())
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	httpx.OK(c, node)
}

func (h *Handler) Facts(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	httpx.OK(c, gin.H{"os": node.OS, "arch": node.Arch, "kernel": node.Kernel, "cpu_cores": node.CpuCores, "memory_mb": node.MemoryMB, "disk_gb": node.DiskGB, "source": "node"})
}

func (h *Handler) Tags(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	httpx.OK(c, hostlogic.ParseLabels(node.Labels))
}

func (h *Handler) Metrics(c *gin.Context) {
	now := time.Now()
	rows := []gin.H{{"id": 1, "cpu": 10, "memory": 256, "disk": 20, "network": 5, "created_at": now.Add(-2 * time.Minute)}, {"id": 2, "cpu": 12, "memory": 260, "disk": 20, "network": 6, "created_at": now.Add(-time.Minute)}, {"id": 3, "cpu": 14, "memory": 262, "disk": 20, "network": 8, "created_at": now}}
	httpx.OK(c, rows)
}

func (h *Handler) Audits(c *gin.Context) {
	rows := []gin.H{{"id": 1, "action": "query", "operator": "system", "detail": "host detail viewed", "created_at": time.Now()}}
	httpx.OK(c, rows)
}
