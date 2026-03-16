package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	dashboardv1 "github.com/cy77cc/OpsPilot/api/dashboard/v1"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type Logic struct {
	svcCtx *svc.ServiceContext
}

func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{svcCtx: svcCtx}
}

func (l *Logic) GetOverview(ctx context.Context, timeRange string) (*dashboardv1.OverviewResponse, error) {
	now := time.Now()
	since, err := parseTimeRange(now, timeRange)
	if err != nil {
		return nil, err
	}

	out := &dashboardv1.OverviewResponse{}
	group, gctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		hostStats, err := l.aggregateHostStats(gctx)
		if err != nil {
			return err
		}
		out.Hosts = hostStats
		return nil
	})

	group.Go(func() error {
		clusterStats, err := l.aggregateClusterStats(gctx)
		if err != nil {
			return err
		}
		out.Clusters = clusterStats
		return nil
	})

	group.Go(func() error {
		serviceStats, err := l.aggregateServiceStats(gctx, now)
		if err != nil {
			return err
		}
		out.Services = serviceStats
		return nil
	})

	group.Go(func() error {
		alerts, err := l.getRecentAlerts(gctx)
		if err != nil {
			return err
		}
		out.Alerts = alerts
		return nil
	})

	group.Go(func() error {
		events, err := l.getRecentEvents(gctx)
		if err != nil {
			return err
		}
		out.Events = events
		return nil
	})

	group.Go(func() error {
		metrics, err := l.getMetricsSeries(gctx, since, now)
		if err != nil {
			return err
		}
		out.Metrics = metrics
		return nil
	})

	group.Go(func() error {
		aiActivity, err := l.getAIActivity(gctx, since, now)
		if err != nil {
			return err
		}
		out.AI = aiActivity
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

func (l *Logic) aggregateHostStats(ctx context.Context) (dashboardv1.HealthStats, error) {
	rows := make([]model.Node, 0, 256)
	if err := l.svcCtx.DB.WithContext(ctx).Select("status", "health_state").Find(&rows).Error; err != nil {
		return dashboardv1.HealthStats{}, err
	}

	out := dashboardv1.HealthStats{Total: len(rows)}
	for _, row := range rows {
		status := strings.ToLower(strings.TrimSpace(row.Status))
		health := strings.ToLower(strings.TrimSpace(row.HealthState))

		if status != "online" {
			out.Offline++
			continue
		}
		switch health {
		case "healthy":
			out.Healthy++
		case "degraded":
			out.Degraded++
		case "critical":
			out.Unhealthy++
		default:
			out.Degraded++
		}
	}
	return out, nil
}

func (l *Logic) aggregateClusterStats(ctx context.Context) (dashboardv1.HealthStats, error) {
	rows := make([]model.Cluster, 0, 128)
	if err := l.svcCtx.DB.WithContext(ctx).Select("status").Find(&rows).Error; err != nil {
		return dashboardv1.HealthStats{}, err
	}

	out := dashboardv1.HealthStats{Total: len(rows)}
	for _, row := range rows {
		status := strings.ToLower(strings.TrimSpace(row.Status))
		switch status {
		case "connected", "ready", "active":
			out.Healthy++
		default:
			out.Unhealthy++
		}
	}
	return out, nil
}

func (l *Logic) aggregateServiceStats(ctx context.Context, now time.Time) (dashboardv1.HealthStats, error) {
	serviceRows := make([]model.Service, 0, 512)
	if err := l.svcCtx.DB.WithContext(ctx).Select("id").Find(&serviceRows).Error; err != nil {
		return dashboardv1.HealthStats{}, err
	}
	if len(serviceRows) == 0 {
		return dashboardv1.HealthStats{}, nil
	}

	serviceIDs := make([]uint, 0, len(serviceRows))
	for _, row := range serviceRows {
		serviceIDs = append(serviceIDs, row.ID)
	}

	type releaseStat struct {
		ServiceID uint
		Status    string
	}
	releases := make([]releaseStat, 0, 2048)
	if err := l.svcCtx.DB.WithContext(ctx).
		Model(&model.ServiceReleaseRecord{}).
		Select("service_id", "status").
		Where("created_at >= ?", now.Add(-24*time.Hour)).
		Find(&releases).Error; err != nil {
		return dashboardv1.HealthStats{}, err
	}

	type serviceAgg struct {
		total   int
		success int
		failed  bool
	}
	agg := make(map[uint]*serviceAgg, len(serviceIDs))
	for _, id := range serviceIDs {
		agg[id] = &serviceAgg{}
	}
	for _, row := range releases {
		a, ok := agg[row.ServiceID]
		if !ok {
			continue
		}
		a.total++
		status := strings.ToLower(strings.TrimSpace(row.Status))
		if status == "failed" {
			a.failed = true
		}
		if status == "success" || status == "succeeded" || status == "applied" {
			a.success++
		}
	}

	out := dashboardv1.HealthStats{Total: len(serviceIDs)}
	for _, id := range serviceIDs {
		a := agg[id]
		if a.total == 0 {
			out.Degraded++
			continue
		}
		if a.failed {
			out.Unhealthy++
			continue
		}
		rate := float64(a.success) / float64(a.total) * 100
		switch {
		case rate >= 95:
			out.Healthy++
		case rate >= 80:
			out.Degraded++
		default:
			out.Unhealthy++
		}
	}
	return out, nil
}

func (l *Logic) getRecentAlerts(ctx context.Context) (dashboardv1.AlertSummary, error) {
	rows := make([]model.AlertEvent, 0, 5)
	if err := l.svcCtx.DB.WithContext(ctx).
		Where("status = ?", "firing").
		Order("created_at DESC").
		Limit(5).
		Find(&rows).Error; err != nil {
		return dashboardv1.AlertSummary{}, err
	}

	items := make([]dashboardv1.AlertItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dashboardv1.AlertItem{
			ID:        fmt.Sprintf("%d", row.ID),
			Title:     defaultString(row.Title, row.Message, "告警事件"),
			Severity:  row.Severity,
			Source:    defaultString(row.Source, row.Metric, "system"),
			CreatedAt: row.CreatedAt,
		})
	}

	return dashboardv1.AlertSummary{
		Firing: len(items),
		Recent: items,
	}, nil
}

func (l *Logic) getRecentEvents(ctx context.Context) ([]dashboardv1.EventItem, error) {
	nodeRows := make([]model.NodeEvent, 0, 16)
	if err := l.svcCtx.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(10).
		Find(&nodeRows).Error; err != nil {
		return nil, err
	}

	alertRows := make([]model.AlertEvent, 0, 16)
	if err := l.svcCtx.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(10).
		Find(&alertRows).Error; err != nil {
		return nil, err
	}

	events := make([]dashboardv1.EventItem, 0, len(nodeRows)+len(alertRows))
	for _, row := range nodeRows {
		events = append(events, dashboardv1.EventItem{
			ID:        fmt.Sprintf("node-%d", row.ID),
			Type:      defaultString(strings.TrimSpace(row.Type), "node_event"),
			Message:   defaultString(strings.TrimSpace(row.Message), "主机事件"),
			CreatedAt: row.CreatedAt,
		})
	}
	for _, row := range alertRows {
		events = append(events, dashboardv1.EventItem{
			ID:        fmt.Sprintf("alert-%d", row.ID),
			Type:      defaultString(strings.TrimSpace(row.Severity), "alert"),
			Message:   defaultString(strings.TrimSpace(row.Title), strings.TrimSpace(row.Message), "告警事件"),
			CreatedAt: row.CreatedAt,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})
	if len(events) > 10 {
		events = events[:10]
	}
	return events, nil
}

// getAIActivity 获取 AI 助手活动统计数据。
func (l *Logic) getAIActivity(ctx context.Context, since, now time.Time) (dashboardv1.AIActivity, error) {
	out := dashboardv1.AIActivity{
		ByScene: make(map[string]int),
	}

	// 查询时间范围内的会话统计
	type spanStats struct {
		TotalCount   int64
		TotalTokens  int64
		TotalMs      int64
		SuccessCount int64
	}

	var stats spanStats
	if err := l.svcCtx.DB.WithContext(ctx).
		Model(&model.AITraceSpan{}).
		Where("start_time >= ? AND start_time <= ?", since, now).
		Select("COUNT(*) as total_count, COALESCE(SUM(tokens), 0) as total_tokens, COALESCE(SUM(duration_ms), 0) as total_ms").
		Scan(&stats).Error; err != nil {
		return out, err
	}

	// 查询成功数量
	var successCount int64
	if err := l.svcCtx.DB.WithContext(ctx).
		Model(&model.AITraceSpan{}).
		Where("start_time >= ? AND start_time <= ?", since, now).
		Where("status = ?", "success").
		Count(&successCount).Error; err != nil {
		return out, err
	}

	// 计算成功率
	var successRate float64
	if stats.TotalCount > 0 {
		successRate = float64(successCount) / float64(stats.TotalCount) * 100
	}

	// 计算平均响应时间
	var avgDuration int64
	if stats.TotalCount > 0 {
		avgDuration = stats.TotalMs / stats.TotalCount
	}

	out.Stats = dashboardv1.AIStatsSummary{
		SessionCount:  stats.TotalCount,
		TokenCount:    stats.TotalTokens,
		AvgDurationMs: avgDuration,
		SuccessRate:   successRate,
	}

	// 查询按场景分组的会话数量
	var sceneCounts []struct {
		Scene string
		Count int64
	}
	if err := l.svcCtx.DB.WithContext(ctx).
		Model(&model.AIChatSession{}).
		Where("created_at >= ? AND created_at <= ?", since, now).
		Select("scene, COUNT(*) as count").
		Group("scene").
		Find(&sceneCounts).Error; err != nil {
		return out, err
	}
	for _, sc := range sceneCounts {
		if sc.Scene != "" {
			out.ByScene[sc.Scene] = int(sc.Count)
		}
	}

	// 查询最近的 AI 会话
	sessions := make([]model.AIChatSession, 0, 5)
	if err := l.svcCtx.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(5).
		Find(&sessions).Error; err != nil {
		return out, err
	}

	out.Sessions = make([]dashboardv1.AISessionItem, 0, len(sessions))
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			title = fmt.Sprintf("%s 场景对话", s.Scene)
		}
		out.Sessions = append(out.Sessions, dashboardv1.AISessionItem{
			ID:        s.ID,
			Scene:     s.Scene,
			Title:     title,
			Status:    "success", // 默认成功，后续可从消息状态推断
			CreatedAt: s.CreatedAt,
		})
	}

	return out, nil
}

func (l *Logic) getMetricsSeries(ctx context.Context, since, now time.Time) (dashboardv1.MetricsSeries, error) {
	// Prometheus 不可用时返回空数据
	if l.svcCtx.Prometheus == nil {
		return dashboardv1.MetricsSeries{}, nil
	}

	// 计算合适的 step
	duration := now.Sub(since)
	step := calculateStep(duration)

	// 查询 CPU 负载指标
	cpuSeries, err := l.queryHostMetricsFromPrometheus(ctx, "host_cpu_load", since, now, step)
	if err != nil {
		return dashboardv1.MetricsSeries{}, err
	}

	// 查询内存使用率指标
	memorySeries, err := l.queryHostMetricsFromPrometheus(ctx, "host_memory_usage_percent", since, now, step)
	if err != nil {
		return dashboardv1.MetricsSeries{}, err
	}

	return dashboardv1.MetricsSeries{
		CPUUsage:    cpuSeries,
		MemoryUsage: memorySeries,
	}, nil
}

// queryHostMetricsFromPrometheus 从 Prometheus 查询主机指标。
func (l *Logic) queryHostMetricsFromPrometheus(ctx context.Context, metric string, start, end time.Time, step time.Duration) ([]dashboardv1.MetricSeries, error) {
	result, err := l.svcCtx.Prometheus.QueryRange(ctx, metric, start, end, step)
	if err != nil {
		return nil, err
	}

	// 按 host_id 分组
	type hostKey struct {
		id   uint64
		name string
	}
	groups := make(map[hostKey][]dashboardv1.MetricPoint)

	// 处理范围查询结果
	for _, series := range result.Matrix {
		hostID := parseHostID(series.Metric["host_id"])
		hostName := series.Metric["host_name"]

		if hostID == 0 {
			continue
		}

		key := hostKey{id: hostID, name: hostName}
		for _, pair := range series.Values {
			if len(pair) >= 2 {
				ts := parseTimestamp(pair[0])
				val := parseFloatValue(pair[1])
				groups[key] = append(groups[key], dashboardv1.MetricPoint{
					Timestamp: ts,
					Value:     val,
				})
			}
		}
	}

	// 转换为切片
	out := make([]dashboardv1.MetricSeries, 0, len(groups))
	for key, points := range groups {
		if len(points) == 0 {
			continue
		}
		// 按时间排序
		sort.Slice(points, func(i, j int) bool {
			return points[i].Timestamp.Before(points[j].Timestamp)
		})
		out = append(out, dashboardv1.MetricSeries{
			HostID:   key.id,
			HostName: key.name,
			Data:     points,
		})
	}

	// 按主机名排序
	sort.Slice(out, func(i, j int) bool {
		return out[i].HostName < out[j].HostName
	})

	return out, nil
}

// calculateStep 根据时间范围计算合适的 step。
func calculateStep(duration time.Duration) time.Duration {
	switch {
	case duration <= 2*time.Hour:
		return 2 * time.Minute
	case duration <= 6*time.Hour:
		return 5 * time.Minute
	default:
		return 10 * time.Minute
	}
}

// parseHostID 解析 host_id 标签值。
func parseHostID(v string) uint64 {
	id, _ := strconv.ParseUint(v, 10, 64)
	return id
}

// parseTimestamp 解析时间戳。
func parseTimestamp(v any) time.Time {
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0)
	case json.Number:
		f, _ := t.Float64()
		return time.Unix(int64(f), 0)
	default:
		return time.Time{}
	}
}

// parseFloatValue 解析浮点值。
func parseFloatValue(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	case json.Number:
		f, _ := t.Float64()
		return f
	default:
		return 0
	}
}

func parseTimeRange(now time.Time, timeRange string) (time.Time, error) {
	switch strings.TrimSpace(timeRange) {
	case "", "1h":
		return now.Add(-1 * time.Hour), nil
	case "6h":
		return now.Add(-6 * time.Hour), nil
	case "24h":
		return now.Add(-24 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("invalid time_range: %s", timeRange)
	}
}

func defaultString(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// GetOverviewV2 返回增强版主控台概览。
//
// 参数:
//   - ctx: 上下文
//   - timeRange: 时间范围 (1h/6h/24h)
//
// 返回: 增强版概览响应，包含健康概览、资源使用、运行状态、告警、事件和 AI 活动
func (l *Logic) GetOverviewV2(ctx context.Context, timeRange string) (*dashboardv1.OverviewResponseV2, error) {
	now := time.Now()
	since, err := parseTimeRange(now, timeRange)
	if err != nil {
		return nil, err
	}

	out := &dashboardv1.OverviewResponseV2{}
	group, gctx := errgroup.WithContext(ctx)

	// 健康概览
	group.Go(func() error {
		hostStats, err := l.aggregateHostStats(gctx)
		if err != nil {
			return err
		}
		out.Health.Hosts = hostStats
		return nil
	})

	group.Go(func() error {
		clusterStats, err := l.aggregateClusterStats(gctx)
		if err != nil {
			return err
		}
		out.Health.Clusters = clusterStats
		return nil
	})

	group.Go(func() error {
		appStats, err := l.aggregateServiceStats(gctx, now)
		if err != nil {
			return err
		}
		out.Health.Applications = appStats
		return nil
	})

	group.Go(func() error {
		workloadStats, err := l.getWorkloadStats(gctx)
		if err != nil {
			return err
		}
		out.Health.Workloads = workloadStats
		return nil
	})

	// 资源使用
	group.Go(func() error {
		metrics, err := l.getMetricsSeries(gctx, since, now)
		if err != nil {
			return err
		}
		out.Resources.CPUUsage = metrics.CPUUsage
		out.Resources.MemoryUsage = metrics.MemoryUsage
		return nil
	})

	group.Go(func() error {
		clusterResources, err := l.getClusterResources(gctx)
		if err != nil {
			return err
		}
		out.Resources.Clusters = clusterResources
		return nil
	})

	// 运行状态
	group.Go(func() error {
		deployStats, err := l.getDeploymentStats(gctx)
		if err != nil {
			return err
		}
		out.Operations.Deployments = deployStats
		return nil
	})

	group.Go(func() error {
		cicdStats, err := l.getCICDStats(gctx)
		if err != nil {
			return err
		}
		out.Operations.CICD = cicdStats
		return nil
	})

	group.Go(func() error {
		issueStats, err := l.getIssuePodStats(gctx)
		if err != nil {
			return err
		}
		out.Operations.IssuePods = issueStats
		return nil
	})

	// 告警事件
	group.Go(func() error {
		alerts, err := l.getRecentAlerts(gctx)
		if err != nil {
			return err
		}
		out.Alerts = alerts
		return nil
	})

	// 事件流（增强版）
	group.Go(func() error {
		events, err := l.getEnrichedEvents(gctx)
		if err != nil {
			return err
		}
		out.Events = events
		return nil
	})

	// AI 活动
	group.Go(func() error {
		aiActivity, err := l.getAIActivity(gctx, since, now)
		if err != nil {
			return err
		}
		out.AI = aiActivity
		return nil
	})

	if err := group.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// getClusterResources 获取集群资源概览。
//
// 从缓存表读取最新的集群资源快照，返回每个集群的 CPU/内存/Pod 使用情况。
func (l *Logic) getClusterResources(ctx context.Context) ([]dashboardv1.ClusterResource, error) {
	type snapshotWithCluster struct {
		model.ClusterResourceSnapshot
		ClusterName string
	}
	var snapshots []snapshotWithCluster
	err := l.svcCtx.DB.WithContext(ctx).
		Table("cluster_resource_snapshots crs").
		Select("crs.*, c.name as cluster_name").
		Joins("JOIN clusters c ON c.id = crs.cluster_id").
		Where("crs.id IN (SELECT MAX(id) FROM cluster_resource_snapshots GROUP BY cluster_id)").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}

	out := make([]dashboardv1.ClusterResource, 0, len(snapshots))
	for _, s := range snapshots {
		cpuUsagePercent := float64(0)
		if s.CPUAllocatableCores > 0 {
			cpuUsagePercent = s.CPUUsageCores / s.CPUAllocatableCores * 100
		}
		memUsagePercent := float64(0)
		if s.MemoryAllocatableMB > 0 {
			memUsagePercent = float64(s.MemoryUsageMB) / float64(s.MemoryAllocatableMB) * 100
		}
		out = append(out, dashboardv1.ClusterResource{
			ClusterID:   s.ClusterID,
			ClusterName: s.ClusterName,
			CPU: dashboardv1.ResourceMetric{
				Allocatable:  s.CPUAllocatableCores,
				Requested:    s.CPURequestedCores,
				Usage:        s.CPUUsageCores,
				UsagePercent: cpuUsagePercent,
			},
			Memory: dashboardv1.ResourceMetric{
				Allocatable:  float64(s.MemoryAllocatableMB),
				Requested:    float64(s.MemoryRequestedMB),
				Usage:        float64(s.MemoryUsageMB),
				UsagePercent: memUsagePercent,
			},
			Pods: dashboardv1.PodStats{
				Total:   s.PodTotal,
				Running: s.PodRunning,
				Pending: s.PodPending,
				Failed:  s.PodFailed,
			},
		})
	}
	return out, nil
}

// getWorkloadStats 获取工作负载健康统计。
//
// 从缓存表读取最新的工作负载统计，返回 Deployment/StatefulSet/DaemonSet 健康状态。
func (l *Logic) getWorkloadStats(ctx context.Context) (dashboardv1.WorkloadStats, error) {
	var stats model.K8sWorkloadStats
	err := l.svcCtx.DB.WithContext(ctx).
		Where("id = (SELECT MAX(id) FROM k8s_workload_stats WHERE namespace = '')").
		First(&stats).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return dashboardv1.WorkloadStats{}, err
	}
	return dashboardv1.WorkloadStats{
		Deployments: dashboardv1.WorkloadHealth{
			Total:   stats.DeploymentTotal,
			Healthy: stats.DeploymentHealthy,
		},
		StatefulSets: dashboardv1.WorkloadHealth{
			Total:   stats.StatefulSetTotal,
			Healthy: stats.StatefulSetHealthy,
		},
		DaemonSets: dashboardv1.WorkloadHealth{
			Total:   stats.DaemonSetTotal,
			Healthy: stats.DaemonSetHealthy,
		},
		Services:  stats.ServiceCount,
		Ingresses: stats.IngressCount,
	}, nil
}

// getDeploymentStats 获取部署状态统计。
//
// 统计正在部署、待审批、今日发布成功/失败的数量。
func (l *Logic) getDeploymentStats(ctx context.Context) (dashboardv1.DeploymentStats, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var stats dashboardv1.DeploymentStats
	var running, pendingApproval, todayTotal, todaySuccess, todayFailed int64

	// 正在部署的数量
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("status IN ?", []string{"deploying", "applying"}).
		Count(&running)
	stats.Running = int(running)

	// 待审批的数量
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("status = ?", "pending_approval").
		Count(&pendingApproval)
	stats.PendingApproval = int(pendingApproval)

	// 今日发布统计
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("created_at >= ?", today).
		Count(&todayTotal)
	stats.TodayTotal = int(todayTotal)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("created_at >= ? AND status = ?", today, "success").
		Count(&todaySuccess)
	stats.TodaySuccess = int(todaySuccess)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("created_at >= ? AND status = ?", today, "failed").
		Count(&todayFailed)
	stats.TodayFailed = int(todayFailed)

	return stats, nil
}

// getCICDStats 获取 CI/CD 状态统计。
//
// 统计运行中、排队中、今日构建成功/失败的数量。
func (l *Logic) getCICDStats(ctx context.Context) (dashboardv1.CICDStats, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var stats dashboardv1.CICDStats
	var running, queued, todayTotal, success, failed int64

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("status = ?", "running").
		Count(&running)
	stats.Running = int(running)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("status = ?", "queued").
		Count(&queued)
	stats.Queued = int(queued)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("triggered_at >= ?", today).
		Count(&todayTotal)
	stats.TodayTotal = int(todayTotal)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("triggered_at >= ? AND status = ?", today, "success").
		Count(&success)
	stats.Success = int(success)

	l.svcCtx.DB.WithContext(ctx).
		Model(&model.CICDServiceCIRun{}).
		Where("triggered_at >= ? AND status = ?", today, "failed").
		Count(&failed)
	stats.Failed = int(failed)

	return stats, nil
}

// getIssuePodStats 获取异常 Pod 统计。
//
// 从缓存表统计异常 Pod 总数和按类型分组的数量。
func (l *Logic) getIssuePodStats(ctx context.Context) (dashboardv1.IssuePodStats, error) {
	var stats dashboardv1.IssuePodStats
	stats.ByType = make(map[string]int)

	var total int64
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.K8sIssuePod{}).
		Count(&total)
	stats.Total = int(total)

	var byType []struct {
		IssueType string
		Count     int64
	}
	l.svcCtx.DB.WithContext(ctx).
		Model(&model.K8sIssuePod{}).
		Select("issue_type, COUNT(*) as count").
		Group("issue_type").
		Find(&byType)

	for _, bt := range byType {
		stats.ByType[bt.IssueType] = int(bt.Count)
	}

	return stats, nil
}

// getEnrichedEvents 获取增强版事件流（包含部署事件）。
//
// 合并主机事件、告警事件和部署事件，按时间排序返回最近的 20 条。
func (l *Logic) getEnrichedEvents(ctx context.Context) ([]dashboardv1.EventItem, error) {
	events := make([]dashboardv1.EventItem, 0, 20)

	// 主机事件
	nodeEvents := make([]model.NodeEvent, 0, 10)
	l.svcCtx.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(10).
		Find(&nodeEvents)
	for _, e := range nodeEvents {
		events = append(events, dashboardv1.EventItem{
			ID:        fmt.Sprintf("node-%d", e.ID),
			Type:      "host_event",
			Message:   e.Message,
			CreatedAt: e.CreatedAt,
		})
	}

	// 告警事件
	alertEvents := make([]model.AlertEvent, 0, 10)
	l.svcCtx.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(10).
		Find(&alertEvents)
	for _, e := range alertEvents {
		events = append(events, dashboardv1.EventItem{
			ID:        fmt.Sprintf("alert-%d", e.ID),
			Type:      "alert",
			Message:   e.Title,
			CreatedAt: e.CreatedAt,
		})
	}

	// 部署事件
	releaseEvents := make([]model.DeploymentRelease, 0, 10)
	l.svcCtx.DB.WithContext(ctx).
		Order("created_at DESC").
		Limit(10).
		Find(&releaseEvents)
	for _, r := range releaseEvents {
		events = append(events, dashboardv1.EventItem{
			ID:        fmt.Sprintf("release-%d", r.ID),
			Type:      "deployment",
			Message:   fmt.Sprintf("发布状态: %s", r.Status),
			CreatedAt: r.CreatedAt,
		})
	}

	// 按时间排序
	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.After(events[j].CreatedAt)
	})

	// 取最近的 20 条
	if len(events) > 20 {
		events = events[:20]
	}

	return events, nil
}
