package handler

import (
	"fmt"
	"time"

	v1 "github.com/cy77cc/OpsPilot/api/host/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	monitormodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/gin-gonic/gin"
)

// ListProcesses 获取主机进程列表。
func (h *Handler) ListProcesses(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	// TODO: SSH 实时获取
	mockData := []v1.ProcessItem{
		{PID: 1, User: "root", CPU: 0.1, Memory: 0.1, VSZ: 168924, RSS: 12456, State: "S", Start: "May12", Time: "0:05", Command: "/sbin/init"},
		{PID: 1205, User: "nginx", CPU: 0.5, Memory: 2.1, VSZ: 245600, RSS: 45600, State: "S", Start: "10:15", Time: "1:22", Command: "nginx: worker process"},
		{PID: 3456, User: "root", CPU: 12.5, Memory: 15.4, VSZ: 4567890, RSS: 1234567, State: "R", Start: "14:20", Time: "5:10", Command: "/usr/bin/java -jar opspilot.jar"},
	}
	httpx.OK(c, mockData)
}

// KillProcess 终止主机进程。
func (h *Handler) KillProcess(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:execute", "host:*") {
		return
	}
	// TODO: 执行 SSH kill 命令
	httpx.OK(c, "process termination signal sent")
}

// ListServices 获取主机服务列表。
func (h *Handler) ListServices(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	// TODO: SSH systemctl list-units
	mockData := []v1.ServiceItem{
		{Name: "nginx.service", Status: "active", Startup: "enabled", Description: "The nginx HTTP and reverse proxy server"},
		{Name: "opspilot-agent.service", Status: "active", Startup: "enabled", Description: "OpsPilot host management agent"},
		{Name: "postgresql.service", Status: "active", Startup: "enabled", Description: "PostgreSQL database server"},
		{Name: "redis.service", Status: "failed", Startup: "disabled", Description: "Advanced key-value store"},
	}
	httpx.OK(c, mockData)
}

// ServiceAction 执行服务操作 (start/stop/restart)。
func (h *Handler) ServiceAction(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:execute", "host:*") {
		return
	}
	// TODO: 执行 SSH systemctl 命令
	httpx.OK(c, "service action triggered")
}

// ListDisks 获取主机磁盘分区信息。
func (h *Handler) ListDisks(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	// TODO: SSH df -h
	mockData := []v1.PartitionItem{
		{Filesystem: "/dev/sda1", Type: "ext4", Size: "40GB", Used: "12GB", Available: "28GB", UsagePct: 30, Mounted: "/"},
		{Filesystem: "/dev/sdb1", Type: "xfs", Size: "100GB", Used: "80GB", Available: "20GB", UsagePct: 80, Mounted: "/data"},
	}
	httpx.OK(c, mockData)
}

// ListNetworkInterfaces 获取主机网络接口信息。
func (h *Handler) ListNetworkInterfaces(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	// TODO: SSH ip addr
	mockData := []v1.InterfaceItem{
		{Name: "ens33", IP: "192.168.1.10", MAC: "00:50:56:af:3e:88", Status: "up", RX: "1.2 GB", TX: "450 MB", MTU: 1500},
		{Name: "docker0", IP: "172.17.0.1", MAC: "02:42:04:60:88:94", Status: "up", RX: "0 B", TX: "0 B", MTU: 1500},
	}
	httpx.OK(c, mockData)
}

// ListNetworkRoutes 获取主机路由表信息。
func (h *Handler) ListNetworkRoutes(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	// TODO: SSH route -n / ip route
	mockData := []v1.RouteItem{
		{Destination: "0.0.0.0", Gateway: "192.168.1.1", Mask: "0.0.0.0", Flags: "UG", Interface: "ens33", Metric: 100},
		{Destination: "192.168.1.0", Gateway: "0.0.0.0", Mask: "255.255.255.0", Flags: "U", Interface: "ens33", Metric: 100},
	}
	httpx.OK(c, mockData)
}

// ListPackages 获取主机已安装软件包。
func (h *Handler) ListPackages(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	// TODO: SSH dpkg -l / rpm -qa
	mockData := []v1.PackageItem{
		{Name: "nginx", Version: "1.18.0-6ubuntu1.4", Arch: "amd64", Status: "installed", Description: "high performance web server"},
		{Name: "openssl", Version: "1.1.1f-1ubuntu2.19", Arch: "amd64", Status: "upgradable", Description: "Secure Sockets Layer toolkit"},
	}
	httpx.OK(c, mockData)
}

// ListAlarms 获取主机告警历史。
func (h *Handler) ListAlarms(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:read", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}

	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}

	var alerts []monitormodel.AlertEvent
	// 根据 IP 过滤告警
	err = h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("source LIKE ?", "%"+node.IP+"%").
		Order("triggered_at DESC").
		Limit(100).
		Find(&alerts).Error
	if err != nil {
		httpx.Fail(c, xcode.DatabaseError, err.Error())
		return
	}

	resp := make([]v1.AlarmHistoryItem, 0, len(alerts))
	for _, a := range alerts {
		duration := "进行中"
		if a.ResolvedAt != nil {
			duration = a.ResolvedAt.Sub(a.TriggeredAt).Round(time.Second).String()
		}
		resp = append(resp, v1.AlarmHistoryItem{
			ID:         fmt.Sprintf("%d", a.ID),
			Level:      a.Severity,
			Title:      a.Title,
			Status:     a.Status,
			StartedAt:  a.TriggeredAt,
			ResolvedAt: a.ResolvedAt,
			Duration:   duration,
			Value:      fmt.Sprintf("%.2f", a.Value),
		})
	}

	// 如果数据库没数据，提供一些 mock 展示
	if len(resp) == 0 {
		now := time.Now()
		resp = []v1.AlarmHistoryItem{
			{ID: "m1", Level: "warning", Title: "磁盘使用率过高", Status: "resolved", StartedAt: now.Add(-1 * time.Hour), Duration: "45m", Value: "85%"},
			{ID: "m2", Level: "critical", Title: "内存使用率超过 90%", Status: "resolved", StartedAt: now.Add(-2 * time.Hour), Duration: "18m", Value: "92%"},
		}
	}

	httpx.OK(c, resp)
}
