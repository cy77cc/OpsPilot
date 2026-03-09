package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/core"
	"github.com/cy77cc/k8s-manage/internal/model"
)

// Input types

type MonitorAlertRuleListInput struct {
	Status  string `json:"status,omitempty" jsonschema_description:"optional rule state filter"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on name/metric"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max rules,default=50"`
}

type MonitorAlertActiveInput struct {
	Severity  string `json:"severity,omitempty" jsonschema_description:"optional severity filter"`
	ServiceID int    `json:"service_id,omitempty" jsonschema_description:"optional service id filter"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max alerts,default=50"`
}

type MonitorAlertInput struct {
	Severity  string `json:"severity,omitempty" jsonschema_description:"optional severity filter"`
	ServiceID int    `json:"service_id,omitempty" jsonschema_description:"optional service id filter"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max alerts,default=50"`
}

type MonitorMetricQueryInput struct {
	Query     string `json:"query" jsonschema_description:"required,metric query or metric name"`
	TimeRange string `json:"time_range,omitempty" jsonschema_description:"time range,default=1h"`
	Step      int    `json:"step,omitempty" jsonschema_description:"step seconds,default=60"`
}

type MonitorMetricInput struct {
	Query     string `json:"query" jsonschema_description:"required,metric query or metric name"`
	TimeRange string `json:"time_range,omitempty" jsonschema_description:"time range,default=1h"`
	Step      int    `json:"step,omitempty" jsonschema_description:"step seconds,default=60"`
}

// NewMonitorTools returns all monitor tools.
func NewMonitorTools(ctx context.Context, deps core.PlatformDeps) []tool.InvokableTool {
	return []tool.InvokableTool{
		MonitorAlertRuleList(ctx, deps),
		MonitorAlert(ctx, deps),
		MonitorAlertActive(ctx, deps),
		MonitorMetric(ctx, deps),
		MonitorMetricQuery(ctx, deps),
	}
}


type MonitorAlertRuleListOutput struct {
	Total int               `json:"total"`
	List  []model.AlertRule `json:"list"`
}

func MonitorAlertRuleList(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_alert_rule_list",
		"Query the list of alert rules configured in the monitoring system. Optional parameters: status filters by rule state (enabled/disabled), keyword searches by rule name or metric name, limit controls max results (default 50, max 200). Returns alert rules with threshold conditions, severity levels, and notification settings. Example: {\"status\":\"enabled\",\"keyword\":\"cpu\"}.",
		func(ctx context.Context, input *MonitorAlertRuleListInput, opts ...tool.Option) (*MonitorAlertRuleListOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := deps.DB.Model(&model.AlertRule{})
			if status := strings.TrimSpace(input.Status); status != "" {
				query = query.Where("state = ? OR status = ?", status, status)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ? OR metric LIKE ?", pattern, pattern)
			}
			var rules []model.AlertRule
			if err := query.Order("id desc").Limit(limit).Find(&rules).Error; err != nil {
				return nil, err
			}
			return &MonitorAlertRuleListOutput{
				Total: len(rules),
				List:  rules,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type MonitorAlertOutput struct {
	Total int                `json:"total"`
	List  []model.AlertEvent `json:"list"`
}

func MonitorAlert(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_alert",
		"Query active/firing alert events from the monitoring system. Optional parameters: severity filters by alert severity (critical/warning/info), service_id filters alerts related to a specific service, limit controls max results (default 50, max 200). Returns alerts currently in firing status with timestamps, labels, and annotations. Example: {\"severity\":\"critical\",\"limit\":20}.",
		func(ctx context.Context, input *MonitorAlertInput, opts ...tool.Option) (*MonitorAlertOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := deps.DB.Model(&model.AlertEvent{}).Where("status = ?", "firing")
			if severity := strings.TrimSpace(input.Severity); severity != "" {
				query = query.Where("severity = ?", severity)
			}
			if input.ServiceID > 0 {
				query = query.Where("source LIKE ?", fmt.Sprintf("%%service:%d%%", input.ServiceID))
			}
			var alerts []model.AlertEvent
			if err := query.Order("triggered_at desc").Limit(limit).Find(&alerts).Error; err != nil {
				return nil, err
			}
			return &MonitorAlertOutput{
				Total: len(alerts),
				List:  alerts,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type MonitorAlertActiveOutput struct {
	Total int                `json:"total"`
	List  []model.AlertEvent `json:"list"`
}

func MonitorAlertActive(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_alert_active",
		"Query all active/firing alerts currently affecting the system. Optional parameters: severity filters by alert level (critical/warning/info), service_id filters by specific service, limit controls max results (default 50, max 200). Use this to get a quick overview of all ongoing issues. Example: {\"severity\":\"critical\"}.",
		func(ctx context.Context, input *MonitorAlertActiveInput, opts ...tool.Option) (*MonitorAlertActiveOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := deps.DB.Model(&model.AlertEvent{}).Where("status = ?", "firing")
			if severity := strings.TrimSpace(input.Severity); severity != "" {
				query = query.Where("severity = ?", severity)
			}
			if input.ServiceID > 0 {
				query = query.Where("source LIKE ?", fmt.Sprintf("%%service:%d%%", input.ServiceID))
			}
			var alerts []model.AlertEvent
			if err := query.Order("triggered_at desc").Limit(limit).Find(&alerts).Error; err != nil {
				return nil, err
			}
			return &MonitorAlertActiveOutput{
				Total: len(alerts),
				List:  alerts,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type MonitorMetricOutput struct {
	Query     string              `json:"query"`
	TimeRange string              `json:"time_range"`
	Step      int                 `json:"step"`
	Points    []model.MetricPoint `json:"points"`
	Count     int                 `json:"count"`
}

func MonitorMetric(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_metric",
		"Query time-series metric data from the monitoring system. query is required and specifies the metric name or PromQL expression. Optional parameters: time_range sets the query duration (default 1h, accepts values like 5m, 1h, 24h), step sets the data point interval in seconds (default 60). Returns metric points with timestamps and values. Example: {\"query\":\"cpu_usage\",\"time_range\":\"1h\",\"step\":60}.",
		func(ctx context.Context, input *MonitorMetricInput, opts ...tool.Option) (*MonitorMetricOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			queryName := strings.TrimSpace(input.Query)
			if queryName == "" {
				return nil, fmt.Errorf("query is required")
			}
			rangeDuration := parseTimeRange(strings.TrimSpace(input.TimeRange), time.Hour)
			step := input.Step
			if step <= 0 {
				step = 60
			}
			since := time.Now().Add(-rangeDuration)
			var points []model.MetricPoint
			if err := deps.DB.Where("metric = ? AND collected_at >= ?", queryName, since).
				Order("collected_at asc").
				Limit(2000).
				Find(&points).Error; err != nil {
				return nil, err
			}
			return &MonitorMetricOutput{
				Query:     queryName,
				TimeRange: rangeDuration.String(),
				Step:      step,
				Points:    points,
				Count:     len(points),
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type MonitorMetricQueryOutput struct {
	Query     string              `json:"query"`
	TimeRange string              `json:"time_range"`
	Step      int                 `json:"step"`
	Points    []model.MetricPoint `json:"points"`
	Count     int                 `json:"count"`
}

func MonitorMetricQuery(ctx context.Context, deps core.PlatformDeps) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_metric_query",
		"Query metric data points over a time range for analysis and visualization. query is required and specifies the metric name to retrieve. Optional parameters: time_range controls how far back to look (default 1h, supports formats like 5m, 30m, 2h, 24h), step sets the resolution in seconds between data points (default 60). Returns an array of metric points with timestamps. Example: {\"query\":\"memory_usage\",\"time_range\":\"30m\"}.",
		func(ctx context.Context, input *MonitorMetricQueryInput, opts ...tool.Option) (*MonitorMetricQueryOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			queryName := strings.TrimSpace(input.Query)
			if queryName == "" {
				return nil, fmt.Errorf("query is required")
			}
			rangeDuration := parseTimeRange(strings.TrimSpace(input.TimeRange), time.Hour)
			step := input.Step
			if step <= 0 {
				step = 60
			}
			since := time.Now().Add(-rangeDuration)
			var points []model.MetricPoint
			if err := deps.DB.Where("metric = ? AND collected_at >= ?", queryName, since).
				Order("collected_at asc").
				Limit(2000).
				Find(&points).Error; err != nil {
				return nil, err
			}
			return &MonitorMetricQueryOutput{
				Query:     queryName,
				TimeRange: rangeDuration.String(),
				Step:      step,
				Points:    points,
				Count:     len(points),
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func parseTimeRange(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
