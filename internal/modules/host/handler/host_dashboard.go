package handler

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	hostlogic "github.com/cy77cc/OpsPilot/internal/modules/host/logic"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type hostDashboardRow struct {
	Node            hostmodel.Node
	Environment     string
	MonitorStatus   string
	CPUUsagePct     int
	MemoryUsagePct  int
	DiskUsagePct    int
	LastHeartbeatAt *time.Time
	AlertCount      int
}

func parseHostFilterParams(c *gin.Context) (keyword, status, environment, region, osName string, tags []string) {
	keyword = strings.TrimSpace(c.Query("keyword"))
	status = strings.ToLower(strings.TrimSpace(c.Query("status")))
	environment = strings.ToLower(strings.TrimSpace(c.Query("environment")))
	region = strings.TrimSpace(c.Query("region"))
	osName = strings.TrimSpace(c.Query("os"))
	for _, raw := range c.QueryArray("tags") {
		for _, part := range strings.Split(raw, ",") {
			tag := strings.TrimSpace(part)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}
	return keyword, status, environment, region, osName, tags
}

func applyHostBaseFilters(db *gorm.DB, keyword, environment, region, osName string, tags []string) *gorm.DB {
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR ip LIKE ? OR labels LIKE ?", like, like, like)
	}
	if region != "" && !strings.EqualFold(region, "all") {
		db = db.Where("region = ?", region)
	}
	if osName != "" && !strings.EqualFold(osName, "all") {
		switch strings.ToLower(strings.TrimSpace(osName)) {
		case "ubuntu":
			db = db.Where("LOWER(os) LIKE ?", "%ubuntu%")
		case "centos":
			db = db.Where("LOWER(os) LIKE ?", "%centos%")
		case "rocky":
			db = db.Where("LOWER(os) LIKE ?", "%rocky%")
		case "windows":
			db = db.Where("LOWER(os) LIKE ?", "%windows%")
		case "other":
			db = db.Where("LOWER(os) NOT LIKE ? AND LOWER(os) NOT LIKE ? AND LOWER(os) NOT LIKE ? AND LOWER(os) NOT LIKE ?", "%ubuntu%", "%centos%", "%rocky%", "%windows%")
		default:
			db = db.Where("os = ?", osName)
		}
	}
	if environment != "" && !strings.EqualFold(environment, "all") {
		needle := map[string]string{
			"prod":    "prod",
			"staging": "stg",
			"test":    "test",
			"dev":     "dev",
			"ops":     "ops",
		}[environment]
		if needle != "" {
			db = db.Where("LOWER(labels) LIKE ?", "%"+needle+"%")
		}
	}
	for _, tag := range tags {
		db = db.Where("labels LIKE ?", "%"+tag+"%")
	}
	return db
}

func detectEnvironmentFromLabels(labels string) string {
	normalized := strings.ToLower(strings.TrimSpace(labels))
	switch {
	case strings.Contains(normalized, "prod"):
		return "prod"
	case strings.Contains(normalized, "stg") || strings.Contains(normalized, "stage") || strings.Contains(normalized, "pre"):
		return "staging"
	case strings.Contains(normalized, "test"):
		return "test"
	case strings.Contains(normalized, "dev"):
		return "dev"
	case strings.Contains(normalized, "ops"):
		return "ops"
	default:
		return "prod"
	}
}

func normalizeNodeOnlineStatus(status string) string {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case "online", "active":
		return "online"
	case "maintenance":
		return "maintenance"
	case "error":
		return "error"
	default:
		return "offline"
	}
}

func computeUsageAndMonitor(snapshot *hostmodel.HostHealthSnapshot) (cpuPct, memoryPct, diskPct int, monitorStatus string, heartbeat *time.Time) {
	if snapshot == nil {
		return 0, 0, 0, "unmanaged", nil
	}
	cpuPct = int(math.Min(100, snapshot.CpuLoad*20))
	if snapshot.MemoryTotalMB > 0 {
		memoryPct = int(math.Min(100, float64(snapshot.MemoryUsedMB)*100/float64(snapshot.MemoryTotalMB)))
	}
	diskPct = int(math.Min(100, snapshot.DiskUsedPct))
	heartbeat = &snapshot.CheckedAt
	state := strings.ToLower(strings.TrimSpace(snapshot.State))
	switch state {
	case "healthy":
		monitorStatus = "healthy"
	case "degraded", "critical":
		monitorStatus = "warning"
	default:
		monitorStatus = "unmanaged"
	}
	return cpuPct, memoryPct, diskPct, monitorStatus, heartbeat
}

func (h *Handler) queryHostDashboardRows(c *gin.Context) ([]hostDashboardRow, error) {
	ctx := c.Request.Context()
	keyword, status, environment, region, osName, tags := parseHostFilterParams(c)

	base := applyHostBaseFilters(h.svcCtx.DB.WithContext(ctx).Model(&hostmodel.Node{}), keyword, environment, region, osName, tags)
	var nodes []hostmodel.Node
	if err := base.Order("id DESC").Find(&nodes).Error; err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return []hostDashboardRow{}, nil
	}

	hostIDs := make([]uint64, 0, len(nodes))
	for _, node := range nodes {
		hostIDs = append(hostIDs, uint64(node.ID))
	}

	var snapshots []hostmodel.HostHealthSnapshot
	if err := h.svcCtx.DB.WithContext(ctx).
		Where("host_id IN ?", hostIDs).
		Order("host_id ASC, checked_at DESC").
		Find(&snapshots).Error; err != nil {
		if !isMissingHealthSnapshotsTableErr(err) {
			return nil, err
		}
	}

	latestByHost := make(map[uint64]*hostmodel.HostHealthSnapshot, len(hostIDs))
	for i := range snapshots {
		hostID := snapshots[i].HostID
		if _, exists := latestByHost[hostID]; !exists {
			latestByHost[hostID] = &snapshots[i]
		}
	}

	rows := make([]hostDashboardRow, 0, len(nodes))
	for _, node := range nodes {
		hostID := uint64(node.ID)
		snapshot := latestByHost[hostID]
		cpuPct, memoryPct, diskPct, monitorStatus, heartbeat := computeUsageAndMonitor(snapshot)
		env := detectEnvironmentFromLabels(node.Labels)
		onlineStatus := normalizeNodeOnlineStatus(node.Status)

		if status != "" && status != "all" {
			abnormal := onlineStatus == "offline" || onlineStatus == "error" || monitorStatus == "warning"
			if (status == "online" && onlineStatus != "online") ||
				(status == "offline" && onlineStatus == "online") ||
				(status == "abnormal" && !abnormal) {
				continue
			}
		}

		alertCount := 0
		if monitorStatus == "warning" {
			alertCount = 1
			if snapshot != nil && strings.EqualFold(snapshot.State, "critical") {
				alertCount = 2
			}
		}
		if onlineStatus == "offline" {
			alertCount++
		}

		rows = append(rows, hostDashboardRow{
			Node:            node,
			Environment:     env,
			MonitorStatus:   monitorStatus,
			CPUUsagePct:     cpuPct,
			MemoryUsagePct:  memoryPct,
			DiskUsagePct:    diskPct,
			LastHeartbeatAt: heartbeat,
			AlertCount:      alertCount,
		})
	}

	return rows, nil
}

func (h *Handler) queryHostDashboardRowsPaginated(c *gin.Context) ([]hostDashboardRow, int64, error) {
	ctx := c.Request.Context()
	keyword, status, environment, region, osName, tags := parseHostFilterParams(c)

	page := 1
	pageSize := 20
	if raw := c.Query("page"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			page = n
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			pageSize = n
		}
	}
	if pageSize > 100 {
		pageSize = 100
	}

	base := applyHostBaseFilters(h.svcCtx.DB.WithContext(ctx).Model(&hostmodel.Node{}), keyword, environment, region, osName, tags)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []hostDashboardRow{}, 0, nil
	}

	var nodes []hostmodel.Node
	offset := (page - 1) * pageSize
	if err := applyHostBaseFilters(h.svcCtx.DB.WithContext(ctx).Model(&hostmodel.Node{}), keyword, environment, region, osName, tags).
		Order("id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&nodes).Error; err != nil {
		return nil, 0, err
	}

	hostIDs := make([]uint64, 0, len(nodes))
	for _, node := range nodes {
		hostIDs = append(hostIDs, uint64(node.ID))
	}

	var snapshots []hostmodel.HostHealthSnapshot
	if err := h.svcCtx.DB.WithContext(ctx).
		Where("host_id IN ?", hostIDs).
		Order("host_id ASC, checked_at DESC").
		Find(&snapshots).Error; err != nil {
		if !isMissingHealthSnapshotsTableErr(err) {
			return nil, 0, err
		}
	}

	latestByHost := make(map[uint64]*hostmodel.HostHealthSnapshot, len(hostIDs))
	for i := range snapshots {
		hostID := snapshots[i].HostID
		if _, exists := latestByHost[hostID]; !exists {
			latestByHost[hostID] = &snapshots[i]
		}
	}

	rows := make([]hostDashboardRow, 0, len(nodes))
	for _, node := range nodes {
		hostID := uint64(node.ID)
		snapshot := latestByHost[hostID]
		cpuPct, memoryPct, diskPct, monitorStatus, heartbeat := computeUsageAndMonitor(snapshot)
		env := detectEnvironmentFromLabels(node.Labels)
		onlineStatus := normalizeNodeOnlineStatus(node.Status)

		if status != "" && status != "all" {
			abnormal := onlineStatus == "offline" || onlineStatus == "error" || monitorStatus == "warning"
			if (status == "online" && onlineStatus != "online") ||
				(status == "offline" && onlineStatus == "online") ||
				(status == "abnormal" && !abnormal) {
				continue
			}
		}

		alertCount := 0
		if monitorStatus == "warning" {
			alertCount = 1
			if snapshot != nil && strings.EqualFold(snapshot.State, "critical") {
				alertCount = 2
			}
		}
		if onlineStatus == "offline" {
			alertCount++
		}

		rows = append(rows, hostDashboardRow{
			Node:            node,
			Environment:     env,
			MonitorStatus:   monitorStatus,
			CPUUsagePct:     cpuPct,
			MemoryUsagePct:  memoryPct,
			DiskUsagePct:    diskPct,
			LastHeartbeatAt: heartbeat,
			AlertCount:      alertCount,
		})
	}

	return rows, total, nil
}

func isMissingAlertsTableErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table") && strings.Contains(msg, "alerts")
}

func isMissingHealthSnapshotsTableErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table") && strings.Contains(msg, "host_health_snapshots")
}

// Overview 获取主机管理概览聚合数据。
func (h *Handler) Overview(c *gin.Context) {
	rows, err := h.queryHostDashboardRows(c)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}

	totalHosts := len(rows)
	onlineHosts := 0
	abnormalHosts := 0
	cpuSum := 0
	memorySum := 0
	for _, row := range rows {
		status := normalizeNodeOnlineStatus(row.Node.Status)
		if status == "online" {
			onlineHosts++
		}
		if status == "offline" || status == "error" || row.MonitorStatus == "warning" {
			abnormalHosts++
		}
		cpuSum += row.CPUUsagePct
		memorySum += row.MemoryUsagePct
	}
	avgCPU := 0.0
	avgMemory := 0.0
	onlineRate := 0.0
	if totalHosts > 0 {
		avgCPU = float64(cpuSum) / float64(totalHosts)
		avgMemory = float64(memorySum) / float64(totalHosts)
		onlineRate = float64(onlineHosts) * 100 / float64(totalHosts)
	}

	startOfDay := time.Now().Truncate(24 * time.Hour)
	severeAlertCount := int64(0)
	warningAlertCount := int64(0)
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Model(&monitoringmodel.AlertEvent{}).
		Where("status = ? AND created_at >= ?", "firing", startOfDay).
		Where("LOWER(severity) = ?", "critical").
		Count(&severeAlertCount).Error; err != nil && !isMissingAlertsTableErr(err) {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Model(&monitoringmodel.AlertEvent{}).
		Where("status = ? AND created_at >= ?", "firing", startOfDay).
		Where("LOWER(severity) IN ?", []string{"warning", "info"}).
		Count(&warningAlertCount).Error; err != nil && !isMissingAlertsTableErr(err) {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}

	httpx.OK(c, gin.H{
		"total_hosts":         totalHosts,
		"online_hosts":        onlineHosts,
		"abnormal_hosts":      abnormalHosts,
		"avg_cpu_usage":       math.Round(avgCPU*10) / 10,
		"avg_memory_usage":    math.Round(avgMemory*10) / 10,
		"today_alert_count":   severeAlertCount + warningAlertCount,
		"severe_alert_count":  severeAlertCount,
		"warning_alert_count": warningAlertCount,
		"online_rate":         math.Round(onlineRate*10) / 10,
	})
}

// Distribution 获取主机分布数据。
func (h *Handler) Distribution(c *gin.Context) {
	rows, err := h.queryHostDashboardRows(c)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}

	ubuntu := 0
	centos := 0
	rocky := 0
	windows := 0
	other := 0
	for _, row := range rows {
		normalized := strings.ToLower(strings.TrimSpace(row.Node.OS))
		switch {
		case strings.Contains(normalized, "windows"):
			windows++
		case strings.Contains(normalized, "ubuntu"):
			ubuntu++
		case strings.Contains(normalized, "centos"):
			centos++
		case strings.Contains(normalized, "rocky"):
			rocky++
		default:
			other++
		}
	}
	total := len(rows)
	asItem := func(name string, value int) gin.H {
		pct := 0.0
		if total > 0 {
			pct = float64(value) * 100 / float64(total)
		}
		return gin.H{"name": name, "value": value, "percent": math.Round(pct*10) / 10}
	}

	items := []gin.H{
		asItem("Ubuntu", ubuntu),
		asItem("CentOS", centos),
		asItem("Rocky Linux", rocky),
		asItem("Windows", windows),
		asItem("Other", other),
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		if v, ok := item["value"].(int); ok && v == 0 && total > 0 {
			continue
		}
		out = append(out, item)
	}
	httpx.OK(c, gin.H{"list": out})
}

// UsageTrend 获取资源使用趋势。
func (h *Handler) UsageTrend(c *gin.Context) {
	rows, err := h.queryHostDashboardRows(c)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	if len(rows) == 0 {
		httpx.OK(c, gin.H{"list": []gin.H{}})
		return
	}

	hostIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		hostIDs = append(hostIDs, uint64(row.Node.ID))
	}

	windowHours := 6
	if v := strings.TrimSpace(c.Query("hours")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 24 {
			windowHours = n
		}
	}

	now := time.Now().Truncate(time.Hour)
	start := now.Add(time.Duration(-windowHours+1) * time.Hour)

	var snapshots []hostmodel.HostHealthSnapshot
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("host_id IN ? AND checked_at >= ?", hostIDs, start).
		Order("checked_at ASC").
		Find(&snapshots).Error; err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}

	type bucket struct {
		count        int
		cpuSum       float64
		memoryPctSum float64
	}
	bucketMap := make(map[time.Time]*bucket, windowHours)
	for i := 0; i < windowHours; i++ {
		ts := start.Add(time.Duration(i) * time.Hour)
		bucketMap[ts] = &bucket{}
	}
	for _, snapshot := range snapshots {
		ts := snapshot.CheckedAt.Truncate(time.Hour)
		b, ok := bucketMap[ts]
		if !ok {
			continue
		}
		b.count++
		b.cpuSum += math.Min(100, snapshot.CpuLoad*20)
		if snapshot.MemoryTotalMB > 0 {
			b.memoryPctSum += math.Min(100, float64(snapshot.MemoryUsedMB)*100/float64(snapshot.MemoryTotalMB))
		}
	}

	points := make([]gin.H, 0, windowHours)
	for i := 0; i < windowHours; i++ {
		ts := start.Add(time.Duration(i) * time.Hour)
		b := bucketMap[ts]
		cpu := 0.0
		memory := 0.0
		if b.count > 0 {
			cpu = b.cpuSum / float64(b.count)
			memory = b.memoryPctSum / float64(b.count)
		}
		points = append(points, gin.H{
			"time":         ts.Format("15:04"),
			"cpu_usage":    math.Round(cpu*10) / 10,
			"memory_usage": math.Round(memory*10) / 10,
		})
	}

	httpx.OK(c, gin.H{"list": points})
}

// PendingAlerts 获取待处理告警聚合。
func (h *Handler) PendingAlerts(c *gin.Context) {
	type alertGroup struct {
		Name  string `gorm:"column:name"`
		Level string `gorm:"column:level"`
		Count int64  `gorm:"column:count"`
	}
	groups := make([]alertGroup, 0, 8)
	err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Model(&monitoringmodel.AlertEvent{}).
		Select("COALESCE(NULLIF(title, ''), NULLIF(metric, ''), '未命名告警') AS name, CASE WHEN LOWER(severity) = 'critical' THEN 'critical' ELSE 'warning' END AS level, COUNT(1) AS count").
		Where("status = ?", "firing").
		Group("name, level").
		Order("count DESC").
		Limit(8).
		Scan(&groups).Error
	if err != nil && !isMissingAlertsTableErr(err) {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}

	if len(groups) == 0 {
		httpx.OK(c, gin.H{"list": []gin.H{}})
		return
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Count == groups[j].Count {
			return groups[i].Name < groups[j].Name
		}
		return groups[i].Count > groups[j].Count
	})

	items := make([]gin.H, 0, len(groups))
	for _, group := range groups {
		items = append(items, gin.H{
			"name":  group.Name,
			"level": group.Level,
			"count": group.Count,
		})
	}
	httpx.OK(c, gin.H{"list": items})
}

func enrichHostListRows(rows []hostDashboardRow) []gin.H {
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		healthState := strings.TrimSpace(row.Node.HealthState)
		if row.MonitorStatus == "warning" && (healthState == "" || strings.EqualFold(healthState, "unknown") || strings.EqualFold(healthState, "unhealthy")) {
			healthState = "degraded"
		}
		if healthState == "" {
			healthState = "unknown"
		}

		item := gin.H{
			"id":                     row.Node.ID,
			"name":                   row.Node.Name,
			"ip":                     row.Node.IP,
			"status":                 normalizeNodeOnlineStatus(row.Node.Status),
			"health_state":           healthState,
			"cpu_cores":              row.Node.CpuCores,
			"memory_mb":              row.Node.MemoryMB,
			"disk_gb":                row.Node.DiskGB,
			"labels":                 hostlogic.ParseLabels(row.Node.Labels),
			"region":                 row.Node.Region,
			"source":                 row.Node.Source,
			"provider":               row.Node.Provider,
			"provider_instance_id":   row.Node.ProviderID,
			"parent_host_id":         row.Node.ParentHostID,
			"maintenance_reason":     row.Node.MaintenanceReason,
			"maintenance_by":         row.Node.MaintenanceBy,
			"maintenance_started_at": row.Node.MaintenanceStartedAt,
			"maintenance_until":      row.Node.MaintenanceUntil,
			"created_at":             row.Node.CreatedAt,
			"updated_at":             row.Node.UpdatedAt,
			"os":                     row.Node.OS,
			"ssh_user":               row.Node.SSHUser,
			"port":                   row.Node.Port,
			"description":            row.Node.Description,
			"ssh_key_id":             row.Node.SSHKeyID,
			"cpu_usage_pct":          row.CPUUsagePct,
			"memory_usage_pct":       row.MemoryUsagePct,
			"disk_usage_pct":         row.DiskUsagePct,
			"monitor_status":         row.MonitorStatus,
			"environment":            row.Environment,
			"alert_count":            row.AlertCount,
		}
		if row.LastHeartbeatAt != nil {
			item["last_heartbeat_at"] = row.LastHeartbeatAt
		}
		out = append(out, item)
	}
	return out
}
