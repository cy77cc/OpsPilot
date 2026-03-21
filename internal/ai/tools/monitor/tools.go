// Package monitor 提供监控和告警相关的工具实现。
//
// 本文件实现监控操作工具集，包括：
//   - 告警规则列表查询
//   - 活跃告警查询
//   - 指标时间序列查询
package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// =============================================================================
// 输入类型定义
// =============================================================================

// MonitorAlertRuleListInput 告警规则列表查询输入。
type MonitorAlertRuleListInput struct {
	Status  string `json:"status,omitempty" jsonschema_description:"optional rule state filter"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on name/metric"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max rules,default=50"`
}

// MonitorAlertActiveInput 活跃告警查询输入。
type MonitorAlertActiveInput struct {
	Severity  string `json:"severity,omitempty" jsonschema_description:"optional severity filter"`
	ServiceID int    `json:"service_id,omitempty" jsonschema_description:"optional service id filter"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max alerts,default=50"`
}

// MonitorAlertInput 告警查询输入。
type MonitorAlertInput struct {
	Severity  string `json:"severity,omitempty" jsonschema_description:"optional severity filter"`
	ServiceID int    `json:"service_id,omitempty" jsonschema_description:"optional service id filter"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"max alerts,default=50"`
}

// MonitorMetricQueryInput 指标查询输入。
type MonitorMetricQueryInput struct {
	Query     string `json:"query" jsonschema_description:"required,metric query or metric name"`
	TimeRange string `json:"time_range,omitempty" jsonschema_description:"time range,default=1h"`
	Step      int    `json:"step,omitempty" jsonschema_description:"step seconds,default auto-calculated based on time_range"`
	HostID    int    `json:"host_id,omitempty" jsonschema_description:"optional host id filter"`
	HostName  string `json:"host_name,omitempty" jsonschema_description:"optional host name filter"`
}

// MonitorMetricInput 指标数据查询输入。
type MonitorMetricInput struct {
	Query     string `json:"query" jsonschema_description:"required,metric query or metric name"`
	TimeRange string `json:"time_range,omitempty" jsonschema_description:"time range,default=1h"`
	Step      int    `json:"step,omitempty" jsonschema_description:"step seconds,default auto-calculated based on time_range"`
	HostID    int    `json:"host_id,omitempty" jsonschema_description:"optional host id filter"`
	HostName  string `json:"host_name,omitempty" jsonschema_description:"optional host name filter"`
}

// NewMonitorTools 创建所有监控工具。
//
// 监控工具全部为只读工具，不修改任何状态。
func NewMonitorTools(ctx context.Context) []tool.InvokableTool {
	return NewMonitorReadonlyTools(ctx)
}

// NewMonitorReadonlyTools 创建监控只读工具子集。
//
// 返回只读工具列表，包括：
//   - 告警规则列表查询（monitor_alert_rule_list）
//   - 活跃告警查询（monitor_alert, monitor_alert_active）
//   - 指标时间序列查询（monitor_metric, monitor_metric_query）
//
// 这些工具不修改任何状态，可安全用于诊断和巡检场景。
func NewMonitorReadonlyTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		MonitorAlertRuleList(ctx),
		MonitorAlert(ctx),
		MonitorAlertActive(ctx),
		MonitorMetric(ctx),
		MonitorMetricQuery(ctx),
	}
}

func depsFromContextOrFallback(ctx context.Context) *svc.ServiceContext {
	svcCtx, _ := runtimectx.ServicesAs[*svc.ServiceContext](ctx)
	return svcCtx
}

type MonitorAlertRuleListOutput struct {
	Total int               `json:"total"`
	List  []model.AlertRule `json:"list"`
}

func MonitorAlertRuleList(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_alert_rule_list",
		"Query the list of alert rules configured in the monitoring system. Optional parameters: status filters by rule state (enabled/disabled), keyword searches by rule name or metric name, limit controls max results (default 50, max 200). Returns alert rules with threshold conditions, severity levels, and notification settings. Example: {\"status\":\"enabled\",\"keyword\":\"cpu\"}.",
		func(ctx context.Context, input *MonitorAlertRuleListInput, opts ...tool.Option) (*MonitorAlertRuleListOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := svcCtx.DB.Model(&model.AlertRule{})
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

func MonitorAlert(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_alert",
		"Query active/firing alert events from the monitoring system. Optional parameters: severity filters by alert severity (critical/warning/info), service_id filters alerts related to a specific service, limit controls max results (default 50, max 200). Returns alerts currently in firing status with timestamps, labels, and annotations. Example: {\"severity\":\"critical\",\"limit\":20}.",
		func(ctx context.Context, input *MonitorAlertInput, opts ...tool.Option) (*MonitorAlertOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := svcCtx.DB.Model(&model.AlertEvent{}).Where("status = ?", "firing")
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

func MonitorAlertActive(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_alert_active",
		"Query all active/firing alerts currently affecting the system. Optional parameters: severity filters by alert level (critical/warning/info), service_id filters by specific service, limit controls max results (default 50, max 200). Use this to get a quick overview of all ongoing issues. Example: {\"severity\":\"critical\"}.",
		func(ctx context.Context, input *MonitorAlertActiveInput, opts ...tool.Option) (*MonitorAlertActiveOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := svcCtx.DB.Model(&model.AlertEvent{}).Where("status = ?", "firing")
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

// MetricPoint 表示单个指标数据点。
type MetricPoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type MonitorMetricOutput struct {
	Query     string        `json:"query"`
	TimeRange string        `json:"time_range"`
	Step      int           `json:"step"`
	Points    []MetricPoint `json:"points"`
	Count     int           `json:"count"`
}

func MonitorMetric(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_metric",
		"Query time-series metric data from the monitoring system with optional host filtering. query is required and specifies the metric name or PromQL expression. Optional parameters: time_range sets the query duration (default 1h, accepts values like 5m, 1h, 24h), step sets the data point interval in seconds (auto-calculated if not specified to limit data to ~500 points), host_id and host_name filter results to specific hosts. Returns metric points with timestamps and values. Example: {\"query\":\"host_cpu_load\",\"time_range\":\"24h\",\"host_id\":123}.",
		func(ctx context.Context, input *MonitorMetricInput, opts ...tool.Option) (*MonitorMetricOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
			if svcCtx == nil || svcCtx.Prometheus == nil {
				return nil, fmt.Errorf("prometheus client unavailable")
			}
			queryName := strings.TrimSpace(input.Query)
			if queryName == "" {
				return nil, fmt.Errorf("query is required")
			}
			rangeDuration := parseTimeRange(strings.TrimSpace(input.TimeRange), time.Hour)
			step := autoCalculateStep(rangeDuration, input.Step)

			// 应用主机过滤
			hostFilter := buildHostFilter(input.HostID, input.HostName)
			if hostFilter != "" {
				// 如果 query 已经是一个完整的 PromQL 表达式，包含大括号，需要插入过滤条件
				if strings.Contains(queryName, "{") {
					// 在 { 后面插入主机过滤条件
					queryName = strings.Replace(queryName, "{", "{"+hostFilter+",", 1)
				} else {
					// 纯指标名称，添加过滤条件
					queryName = queryName + "{" + hostFilter + "}"
				}
			}

			start := time.Now().Add(-rangeDuration)
			end := time.Now()

			result, err := svcCtx.Prometheus.QueryRange(ctx, queryName, start, end, time.Duration(step)*time.Second)
			if err != nil {
				return nil, err
			}

			points := make([]MetricPoint, 0, 2000)
			for _, series := range result.Matrix {
				for _, pair := range series.Values {
					if len(pair) >= 2 {
						points = append(points, MetricPoint{
							Timestamp: parsePromTimestamp(pair[0]),
							Value:     parsePromValue(pair[1]),
							Labels:    series.Metric,
						})
					}
				}
			}

			// 按时间排序
			sort.Slice(points, func(i, j int) bool {
				return points[i].Timestamp.Before(points[j].Timestamp)
			})

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
	Query     string        `json:"query"`
	TimeRange string        `json:"time_range"`
	Step      int           `json:"step"`
	Points    []MetricPoint `json:"points"`
	Count     int           `json:"count"`
}

func MonitorMetricQuery(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"monitor_metric_query",
		"Query metric data points over a time range for analysis and visualization with optional host filtering. query is required and specifies the metric name to retrieve. Optional parameters: time_range controls how far back to look (default 1h, supports formats like 5m, 30m, 2h, 24h), step sets the resolution in seconds between data points (auto-calculated if not specified to limit data to ~500 points), host_id and host_name filter results to specific hosts. Returns an array of metric points with timestamps. Example: {\"query\":\"host_memory_usage_percent\",\"time_range\":\"24h\",\"host_name\":\"prod-server-01\"}.",
		func(ctx context.Context, input *MonitorMetricQueryInput, opts ...tool.Option) (*MonitorMetricQueryOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
			if svcCtx == nil || svcCtx.Prometheus == nil {
				return nil, fmt.Errorf("prometheus client unavailable")
			}
			queryName := strings.TrimSpace(input.Query)
			if queryName == "" {
				return nil, fmt.Errorf("query is required")
			}
			rangeDuration := parseTimeRange(strings.TrimSpace(input.TimeRange), time.Hour)
			step := autoCalculateStep(rangeDuration, input.Step)

			// 应用主机过滤
			hostFilter := buildHostFilter(input.HostID, input.HostName)
			if hostFilter != "" {
				// 如果 query 已经是一个完整的 PromQL 表达式，包含大括号，需要插入过滤条件
				if strings.Contains(queryName, "{") {
					// 在 { 后面插入主机过滤条件
					queryName = strings.Replace(queryName, "{", "{"+hostFilter+",", 1)
				} else {
					// 纯指标名称，添加过滤条件
					queryName = queryName + "{" + hostFilter + "}"
				}
			}

			start := time.Now().Add(-rangeDuration)
			end := time.Now()

			result, err := svcCtx.Prometheus.QueryRange(ctx, queryName, start, end, time.Duration(step)*time.Second)
			if err != nil {
				return nil, err
			}

			points := make([]MetricPoint, 0, 2000)
			for _, series := range result.Matrix {
				for _, pair := range series.Values {
					if len(pair) >= 2 {
						points = append(points, MetricPoint{
							Timestamp: parsePromTimestamp(pair[0]),
							Value:     parsePromValue(pair[1]),
							Labels:    series.Metric,
						})
					}
				}
			}

			// 按时间排序
			sort.Slice(points, func(i, j int) bool {
				return points[i].Timestamp.Before(points[j].Timestamp)
			})

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

// parsePromTimestamp 解析 Prometheus 时间戳。
func parsePromTimestamp(v any) time.Time {
	switch t := v.(type) {
	case float64:
		return time.Unix(int64(t), 0)
	case json.Number:
		f, _ := t.Float64()
		return time.Unix(int64(f), 0)
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return time.Unix(int64(f), 0)
	default:
		return time.Time{}
	}
}

// parsePromValue 解析 Prometheus 值。
func parsePromValue(v any) float64 {
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

// autoCalculateStep 根据时间范围自动计算合适的步长（秒）。
// 目标：保持返回的数据点数在 500 以内，避免数据量过大。
//
// 参数:
//   - rangeDuration: 查询时间范围
//   - userStep: 用户指定的步长（0 表示未指定）
//
// 返回: 计算后的步长（秒）
func autoCalculateStep(rangeDuration time.Duration, userStep int) int {
	if userStep > 0 {
		return userStep // 用户指定了步长，直接使用
	}

	// 自动计算步长，保持数据点数在 500 左右
	targetPoints := 500
	totalSeconds := int(rangeDuration.Seconds())
	calculatedStep := totalSeconds / targetPoints
	if calculatedStep < 1 {
		calculatedStep = 1
	}
	// 将步长向上舍入到合理的值（1s, 5s, 10s, 30s, 60s, 5m, 10m, 15m, 30m, 1h）
	switch {
	case calculatedStep <= 1:
		return 1
	case calculatedStep <= 5:
		return 5
	case calculatedStep <= 10:
		return 10
	case calculatedStep <= 30:
		return 30
	case calculatedStep <= 60:
		return 60
	case calculatedStep <= 300: // 5m
		return 300
	case calculatedStep <= 600: // 10m
		return 600
	case calculatedStep <= 900: // 15m
		return 900
	case calculatedStep <= 1800: // 30m
		return 1800
	default:
		return 3600 // 1h
	}
}

// buildHostFilter 构建主机过滤的 PromQL 条件。
//
// 参数:
//   - hostID: 主机 ID（0 表示未指定）
//   - hostName: 主机名（空表示未指定）
//
// 返回: PromQL 过滤条件字符串（如 {instance=~"host1|host2"} 或空字符串）
func buildHostFilter(hostID int, hostName string) string {
	var filters []string
	if hostID > 0 {
		filters = append(filters, fmt.Sprintf("host_id=\"%d\"", hostID))
	}
	if hostName != "" {
		filters = append(filters, fmt.Sprintf("hostname=\"%s\"", hostName))
	}
	if len(filters) == 0 {
		return ""
	}
	return strings.Join(filters, ",")
}
