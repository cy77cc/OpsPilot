// Package handler 提供主机管理服务的 HTTP 处理器。
package handler

import (
	"encoding/json"
	"math"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// List 获取主机列表。
//
// @Summary 获取主机列表
// @Description 获取所有主机信息
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts [get]
func (h *Handler) List(c *gin.Context) {
	list, err := h.hostService.List(c.Request.Context())
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// Get 获取单个主机详情。
//
// @Summary 获取主机详情
// @Description 根据主机 ID 获取主机详细信息
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /hosts/{id} [get]
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

// Facts 获取主机系统信息。
//
// @Summary 获取主机系统信息
// @Description 获取主机操作系统、架构、内核、CPU、内存、磁盘等系统信息
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /hosts/{id}/facts [get]
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

// Tags 获取主机标签列表。
//
// @Summary 获取主机标签
// @Description 获取指定主机的所有标签
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /hosts/{id}/tags [get]
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

// Metrics 获取主机健康指标历史。
//
// @Summary 获取主机健康指标
// @Description 获取主机最近 50 条健康检查快照，包括 CPU、内存、磁盘、网络等指标
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/metrics [get]
func (h *Handler) Metrics(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	snapshots, err := h.hostService.ListHealthSnapshots(c.Request.Context(), id, 50)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	rows := make([]gin.H, 0, len(snapshots))
	for _, s := range snapshots {
		cpuPct := math.Min(100, s.CpuLoad*20)
		memoryPct := 0.0
		if s.MemoryTotalMB > 0 {
			memoryPct = math.Min(100, float64(s.MemoryUsedMB)*100/float64(s.MemoryTotalMB))
		}
		extra := map[string]any{}
		if s.SummaryJSON != "" {
			_ = json.Unmarshal([]byte(s.SummaryJSON), &extra)
		}
		rows = append(rows, gin.H{
			"id":            s.ID,
			"cpu":           int(cpuPct),
			"memory":        int(memoryPct),
			"disk":          int(s.DiskUsedPct),
			"network":       0,
			"latency_ms":    s.LatencyMS,
			"health_state":  s.State,
			"error_message": s.ErrorMessage,
			"summary":       extra,
			"created_at":    s.CheckedAt,
		})
	}
	httpx.OK(c, rows)
}

// HealthCheck 执行主机健康检查。
//
// @Summary 执行主机健康检查
// @Description 立即执行一次主机健康检查，返回检查结果快照
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/health/check [post]
func (h *Handler) HealthCheck(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	snapshot, err := h.hostService.RunHealthCheck(c.Request.Context(), id, getUID(c))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, snapshot)
}

// Audits 获取主机审计日志。
//
// @Summary 获取主机审计日志
// @Description 获取指定主机的操作审计记录（MVP 阶段返回模拟数据）
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /hosts/{id}/audits [get]
func (h *Handler) Audits(c *gin.Context) {
	rows := []gin.H{{"id": 1, "action": "query", "operator": "system", "detail": "host detail viewed", "created_at": time.Now()}}
	httpx.OK(c, rows)
}
