package dashboard

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	dashboardv1 "github.com/cy77cc/k8s-manage/api/dashboard/v1"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"golang.org/x/sync/errgroup"
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

func (l *Logic) getMetricsSeries(ctx context.Context, since, now time.Time) (dashboardv1.MetricsSeries, error) {
	cpuRows, err := l.listMetricPoints(ctx, "cpu_usage", since, now)
	if err != nil {
		return dashboardv1.MetricsSeries{}, err
	}
	memoryRows, err := l.listMetricPoints(ctx, "memory_usage", since, now)
	if err != nil {
		return dashboardv1.MetricsSeries{}, err
	}
	return dashboardv1.MetricsSeries{
		CPUUsage:    cpuRows,
		MemoryUsage: memoryRows,
	}, nil
}

func (l *Logic) listMetricPoints(ctx context.Context, metric string, since, now time.Time) ([]dashboardv1.MetricPoint, error) {
	rows := make([]model.MetricPoint, 0, 128)
	if err := l.svcCtx.DB.WithContext(ctx).
		Where("metric = ? AND collected_at >= ? AND collected_at <= ?", metric, since, now).
		Order("collected_at DESC").
		Limit(60).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	// Query uses DESC for cheap limit; convert to ASC for chart display.
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Collected.Before(rows[j].Collected)
	})

	out := make([]dashboardv1.MetricPoint, 0, len(rows))
	for _, row := range rows {
		out = append(out, dashboardv1.MetricPoint{
			Timestamp: row.Collected,
			Value:     row.Value,
		})
	}
	return out, nil
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
