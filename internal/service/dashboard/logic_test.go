package dashboard

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDashboardTestLogic(t *testing.T) *Logic {
	t.Helper()
	dsn := fmt.Sprintf("file:dashboard-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Node{},
		&model.Cluster{},
		&model.Service{},
		&model.ServiceReleaseRecord{},
		&model.AlertEvent{},
		&model.NodeEvent{},
		&model.MetricPoint{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewLogic(&svc.ServiceContext{DB: db})
}

func TestGetOverviewAggregatesHealthStats(t *testing.T) {
	logic := newDashboardTestLogic(t)
	ctx := context.Background()

	nodes := []model.Node{
		{ID: 1, Name: "n1", IP: "10.0.0.1", SSHUser: "root", Status: "online", HealthState: "healthy"},
		{ID: 2, Name: "n2", IP: "10.0.0.2", SSHUser: "root", Status: "online", HealthState: "degraded"},
		{ID: 3, Name: "n3", IP: "10.0.0.3", SSHUser: "root", Status: "offline", HealthState: "critical"},
	}
	if err := logic.svcCtx.DB.WithContext(ctx).Create(&nodes).Error; err != nil {
		t.Fatalf("seed nodes: %v", err)
	}

	clusters := []model.Cluster{
		{Name: "c1", Status: "active", Type: "kubernetes"},
		{Name: "c2", Status: "degraded", Type: "kubernetes"},
	}
	if err := logic.svcCtx.DB.WithContext(ctx).Create(&clusters).Error; err != nil {
		t.Fatalf("seed clusters: %v", err)
	}

	services := []model.Service{
		{ID: 1, Name: "s1", Type: "stateless", Image: "nginx:1.25"},
		{ID: 2, Name: "s2", Type: "stateless", Image: "nginx:1.25"},
		{ID: 3, Name: "s3", Type: "stateless", Image: "nginx:1.25"},
		{ID: 4, Name: "s4", Type: "stateless", Image: "nginx:1.25"},
	}
	if err := logic.svcCtx.DB.WithContext(ctx).Create(&services).Error; err != nil {
		t.Fatalf("seed services: %v", err)
	}

	now := time.Now()
	releases := []model.ServiceReleaseRecord{
		{ServiceID: 1, Status: "success", CreatedAt: now.Add(-2 * time.Hour)},
		{ServiceID: 2, Status: "success", CreatedAt: now.Add(-3 * time.Hour)},
		{ServiceID: 2, Status: "success", CreatedAt: now.Add(-2 * time.Hour)},
		{ServiceID: 2, Status: "failed", CreatedAt: now.Add(-1 * time.Hour)},
		{ServiceID: 3, Status: "success", CreatedAt: now.Add(-2 * time.Hour)},
		{ServiceID: 3, Status: "success", CreatedAt: now.Add(-1 * time.Hour)},
		{ServiceID: 3, Status: "success", CreatedAt: now.Add(-30 * time.Minute)},
		{ServiceID: 3, Status: "success", CreatedAt: now.Add(-10 * time.Minute)},
		{ServiceID: 3, Status: "success", CreatedAt: now.Add(-5 * time.Minute)},
	}
	if err := logic.svcCtx.DB.WithContext(ctx).Create(&releases).Error; err != nil {
		t.Fatalf("seed releases: %v", err)
	}

	resp, err := logic.GetOverview(ctx, "1h")
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}

	if resp.Hosts.Total != 3 || resp.Hosts.Healthy != 1 || resp.Hosts.Degraded != 1 || resp.Hosts.Offline != 1 {
		t.Fatalf("unexpected host stats: %+v", resp.Hosts)
	}
	if resp.Clusters.Total != 2 || resp.Clusters.Healthy != 1 || resp.Clusters.Unhealthy != 1 {
		t.Fatalf("unexpected cluster stats: %+v", resp.Clusters)
	}
	if resp.Services.Total != 4 || resp.Services.Healthy != 2 || resp.Services.Degraded != 1 || resp.Services.Unhealthy != 1 {
		t.Fatalf("unexpected service stats: %+v", resp.Services)
	}
}

func TestGetOverviewMergesEventsAndMetricsLimit(t *testing.T) {
	logic := newDashboardTestLogic(t)
	ctx := context.Background()
	now := time.Now()

	alerts := []model.AlertEvent{
		{ID: 101, Title: "CPU high", Severity: "critical", Source: "node-1", Status: "firing", CreatedAt: now.Add(-2 * time.Minute)},
		{ID: 102, Title: "Memory high", Severity: "warning", Source: "node-2", Status: "firing", CreatedAt: now.Add(-4 * time.Minute)},
	}
	if err := logic.svcCtx.DB.WithContext(ctx).Create(&alerts).Error; err != nil {
		t.Fatalf("seed alerts: %v", err)
	}

	nodeEvents := []model.NodeEvent{
		{ID: 201, NodeID: 1, Type: "host_online", Message: "node-1 online", CreatedAt: now.Add(-1 * time.Minute)},
		{ID: 202, NodeID: 2, Type: "host_offline", Message: "node-2 offline", CreatedAt: now.Add(-3 * time.Minute)},
	}
	if err := logic.svcCtx.DB.WithContext(ctx).Create(&nodeEvents).Error; err != nil {
		t.Fatalf("seed node events: %v", err)
	}

	metricRows := make([]model.MetricPoint, 0, 140)
	for i := 0; i < 70; i++ {
		at := now.Add(-time.Duration(70-i) * time.Minute)
		metricRows = append(metricRows,
			model.MetricPoint{Metric: "cpu_usage", Source: "host", DimensionsJSON: `{"host_id":1,"host_name":"host-1"}`, Value: float64(i), Collected: at},
			model.MetricPoint{Metric: "memory_usage", Source: "host", DimensionsJSON: `{"host_id":1,"host_name":"host-1"}`, Value: float64(i) + 10, Collected: at},
		)
	}
	if err := logic.svcCtx.DB.WithContext(ctx).Create(&metricRows).Error; err != nil {
		t.Fatalf("seed metric points: %v", err)
	}

	resp, err := logic.GetOverview(ctx, "24h")
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}

	if len(resp.Alerts.Recent) != 2 || resp.Alerts.Firing != 2 {
		t.Fatalf("unexpected alerts payload: %+v", resp.Alerts)
	}
	if len(resp.Events) != 4 {
		t.Fatalf("unexpected events len: %d", len(resp.Events))
	}
	// Metrics are now grouped by host
	if len(resp.Metrics.CPUUsage) != 1 || len(resp.Metrics.MemoryUsage) != 1 {
		t.Fatalf("unexpected metric series count: cpu=%d mem=%d", len(resp.Metrics.CPUUsage), len(resp.Metrics.MemoryUsage))
	}
	// 24h range has limitPerHost=288, but we only have 70 points
	if len(resp.Metrics.CPUUsage[0].Data) != 70 || len(resp.Metrics.MemoryUsage[0].Data) != 70 {
		t.Fatalf("unexpected metric data size: cpu=%d mem=%d", len(resp.Metrics.CPUUsage[0].Data), len(resp.Metrics.MemoryUsage[0].Data))
	}
	if len(resp.Metrics.CPUUsage[0].Data) >= 2 {
		if !resp.Metrics.CPUUsage[0].Data[0].Timestamp.Before(resp.Metrics.CPUUsage[0].Data[len(resp.Metrics.CPUUsage[0].Data)-1].Timestamp) {
			t.Fatalf("cpu series should be ascending by timestamp")
		}
	}
}

func TestParseTimeRangeValidation(t *testing.T) {
	now := time.Now()
	if _, err := parseTimeRange(now, "2h"); err == nil {
		t.Fatalf("expected invalid time range to fail")
	}
	if _, err := parseTimeRange(now, "6h"); err != nil {
		t.Fatalf("expected 6h to be valid: %v", err)
	}
}
