// Package logic 提供监控告警服务的业务逻辑层。
//
// 本文件实现监控模块的核心业务逻辑，包括:
//   - 告警事件和规则的 CRUD 操作
//   - Prometheus 指标查询和聚合
//   - 通知渠道管理
//   - 告警投递记录查询
//   - 规则评估和告警触发
package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/service/notification"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// Logic 是监控服务的业务逻辑层。
//
// 提供告警规则、事件、指标和通知渠道的业务处理能力。
type Logic struct {
	svcCtx *svc.ServiceContext // 服务上下文
}

// MetricQuery 是指标查询请求结构。
//
// 定义从 Prometheus 查询时间序列数据的参数。
type MetricQuery struct {
	Metric         string    // 指标名称
	Start          time.Time // 查询开始时间
	End            time.Time // 查询结束时间
	GranularitySec int       // 采样粒度 (秒)
	Source         string    // 数据来源标签
}

// AggregationQuery 是聚合查询请求结构。
//
// 定义对指标进行聚合计算的参数。
type AggregationQuery struct {
	Metric    string    // 指标名称
	Func      string    // 聚合函数 (avg/sum/max/min)
	Start     time.Time // 查询开始时间
	End       time.Time // 查询结束时间
	Source    string    // 数据来源标签
	WindowMin int       // 聚合窗口 (分钟)
}

// MetricQueryResult 是指标查询结果结构。
//
// 包含时间窗口信息和查询到的数据序列。
type MetricQueryResult struct {
	Window struct {
		Start          time.Time `json:"start"`          // 窗口开始时间
		End            time.Time `json:"end"`            // 窗口结束时间
		GranularitySec int       `json:"granularity_sec"` // 采样粒度
	} `json:"window"`
	Dimensions map[string]any   `json:"dimensions"` // 维度信息
	Series     []map[string]any `json:"series"`     // 数据序列
}

// AggregationResult 是聚合查询结果结构。
//
// 包含聚合计算的结果值和时间戳。
type AggregationResult struct {
	Metric    string    `json:"metric"`    // 指标名称
	Func      string    `json:"func"`      // 聚合函数
	Source    string    `json:"source"`    // 数据来源
	Value     float64   `json:"value"`     // 聚合值
	Timestamp time.Time `json:"timestamp"` // 时间戳
}

// NewLogic 创建业务逻辑层实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库、缓存等依赖
//
// 返回: 初始化完成的 Logic 实例
func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{svcCtx: svcCtx}
}

// ListAlerts 查询告警事件列表。
//
// 支持按严重级别和状态筛选，返回分页结果。
//
// 参数:
//   - ctx: 上下文
//   - severity: 严重级别筛选 (可选)
//   - status: 状态筛选 (可选)
//   - page: 页码
//   - pageSize: 每页数量
//
// 返回: 告警事件列表、总数和可能的错误
func (l *Logic) ListAlerts(ctx context.Context, severity, status string, page, pageSize int) ([]model.AlertEvent, int64, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertEvent{})
	if severity != "" {
		q = q.Where("severity = ?", severity)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]model.AlertEvent, 0, pageSize)
	offset := (page - 1) * pageSize
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListRules 查询告警规则列表。
//
// 首次查询时会自动初始化默认规则和通知渠道。
// 返回分页结果。
//
// 参数:
//   - ctx: 上下文
//   - page: 页码
//   - pageSize: 每页数量
//
// 返回: 告警规则列表、总数和可能的错误
func (l *Logic) ListRules(ctx context.Context, page, pageSize int) ([]model.AlertRule, int64, error) {
	if err := l.ensureDefaultRules(ctx); err != nil {
		return nil, 0, err
	}
	if err := l.ensureDefaultChannels(ctx); err != nil {
		return nil, 0, err
	}
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertRule{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]model.AlertRule, 0, pageSize)
	offset := (page - 1) * pageSize
	if err := q.Order("id ASC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// CreateRule 创建告警规则。
//
// 设置默认值并持久化到数据库。
//
// 参数:
//   - ctx: 上下文
//   - rule: 告警规则数据
//
// 返回: 创建成功的规则和可能的错误
func (l *Logic) CreateRule(ctx context.Context, rule model.AlertRule) (*model.AlertRule, error) {
	rule.State = boolToRuleState(rule.Enabled)
	if rule.WindowSec <= 0 {
		rule.WindowSec = 3600
	}
	if rule.GranularitySec <= 0 {
		rule.GranularitySec = 60
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateRule 更新告警规则。
//
// 支持部分字段更新，会同步更新状态字段。
//
// 参数:
//   - ctx: 上下文
//   - id: 规则 ID
//   - payload: 更新字段映射
//
// 返回: 更新后的规则和可能的错误
func (l *Logic) UpdateRule(ctx context.Context, id uint, payload map[string]any) (*model.AlertRule, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty update payload")
	}
	if v, ok := payload["enabled"]; ok {
		if b, ok := v.(bool); ok {
			payload["state"] = boolToRuleState(b)
		}
	}
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertRule{}).Where("id = ?", id).Updates(payload).Error; err != nil {
		return nil, err
	}
	var row model.AlertRule
	if err := l.svcCtx.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// SetRuleEnabled 设置告警规则的启用状态。
//
// 同时更新 enabled 和 state 字段。
//
// 参数:
//   - ctx: 上下文
//   - id: 规则 ID
//   - enabled: 目标启用状态
//
// 返回: 更新后的规则和可能的错误
func (l *Logic) SetRuleEnabled(ctx context.Context, id uint, enabled bool) (*model.AlertRule, error) {
	payload := map[string]any{
		"enabled": enabled,
		"state":   boolToRuleState(enabled),
	}
	return l.UpdateRule(ctx, id, payload)
}

// GetMetrics 查询指标时间序列数据。
//
// 从 Prometheus 查询指定指标在时间范围内的数据点。
// 支持缓存，缓存有效期 30 秒。
//
// 参数:
//   - ctx: 上下文
//   - query: 查询参数
//
// 返回: 查询结果和可能的错误
func (l *Logic) GetMetrics(ctx context.Context, query MetricQuery) (*MetricQueryResult, error) {
	if query.Metric == "" {
		return nil, fmt.Errorf("metric is required")
	}
	if query.GranularitySec <= 0 {
		query.GranularitySec = 60
	}

	// 映射前端指标名到 Prometheus 指标名
	query.Metric = mapMetricName(query.Metric)

	if l.svcCtx.Prometheus == nil {
		return nil, fmt.Errorf("prometheus client unavailable")
	}
	return l.queryMetricsFromPrometheus(ctx, query)
}

// mapMetricName 将前端指标名映射到 Prometheus 指标名。
//
// 提供友好的指标名别名，便于前端使用。
//
// 参数:
//   - metric: 前端指标名
//
// 返回: Prometheus 指标名
func mapMetricName(metric string) string {
	switch strings.TrimSpace(metric) {
	case "cpu_usage":
		return "host_cpu_load"
	case "memory_usage":
		return "host_memory_usage_percent"
	case "disk_usage":
		return "host_disk_usage_percent"
	default:
		return metric
	}
}

// queryMetricsFromPrometheus 从 Prometheus 查询指标数据。
//
// 执行 range query 并将结果转换为前端友好的格式。
// 支持缓存以减少对 Prometheus 的压力。
//
// 参数:
//   - ctx: 上下文
//   - query: 查询参数
//
// 返回: 查询结果和可能的错误
func (l *Logic) queryMetricsFromPrometheus(ctx context.Context, query MetricQuery) (*MetricQueryResult, error) {
	cacheKey := fmt.Sprintf("monitoring:metrics:%s:%s:%d:%d:%d", query.Metric, strings.TrimSpace(query.Source), query.Start.Unix(), query.End.Unix(), query.GranularitySec)
	if l.svcCtx.CacheFacade != nil {
		if cached, ok := l.svcCtx.CacheFacade.Get(cacheKey); ok {
			var out MetricQueryResult
			if err := json.Unmarshal([]byte(cached), &out); err == nil {
				return &out, nil
			}
		}
	}

	qb := prominfra.NewQueryBuilder(query.Metric)
	if strings.TrimSpace(query.Source) != "" {
		qb.WithLabel("source", strings.TrimSpace(query.Source))
	}
	res, err := l.svcCtx.Prometheus.QueryRange(
		ctx,
		qb.Build(),
		query.Start,
		query.End,
		time.Duration(query.GranularitySec)*time.Second,
	)
	if err != nil {
		return nil, err
	}

	out := &MetricQueryResult{
		Dimensions: map[string]any{
			"metric": query.Metric,
			"source": strings.TrimSpace(query.Source),
		},
		Series: make([]map[string]any, 0, 1024),
	}
	out.Window.Start = query.Start
	out.Window.End = query.End
	out.Window.GranularitySec = query.GranularitySec

	appendPoint := func(tsVal any, valueVal any, labels map[string]string) {
		tsFloat, ok := toFloat64(tsVal)
		if !ok {
			return
		}
		valueFloat, ok := toFloat64(valueVal)
		if !ok {
			return
		}
		labelMap := make(map[string]any, len(labels))
		for k, v := range labels {
			if k == "__name__" {
				continue
			}
			labelMap[k] = v
		}
		item := map[string]any{
			"timestamp": time.Unix(int64(tsFloat), 0).UTC(),
			"value":     valueFloat,
		}
		if len(labelMap) > 0 {
			item["labels"] = labelMap
		}
		out.Series = append(out.Series, item)
	}

	for _, series := range res.Matrix {
		for _, pair := range series.Values {
			if len(pair) < 2 {
				continue
			}
			appendPoint(pair[0], pair[1], series.Metric)
		}
	}
	for _, point := range res.Vector {
		if len(point.Value) < 2 {
			continue
		}
		appendPoint(point.Value[0], point.Value[1], point.Metric)
	}
	if l.svcCtx.CacheFacade != nil {
		if b, err := json.Marshal(out); err == nil {
			l.svcCtx.CacheFacade.Set(ctx, cacheKey, string(b), 30*time.Second)
		}
	}

	return out, nil
}

// GetMetricAggregation 查询指标聚合值。
//
// 对指定指标在时间窗口内执行聚合计算。
//
// 参数:
//   - ctx: 上下文
//   - query: 聚合查询参数
//
// 返回: 聚合结果和可能的错误
func (l *Logic) GetMetricAggregation(ctx context.Context, query AggregationQuery) (*AggregationResult, error) {
	if l.svcCtx.Prometheus == nil {
		return nil, fmt.Errorf("prometheus client is unavailable")
	}
	if strings.TrimSpace(query.Metric) == "" {
		return nil, fmt.Errorf("metric is required")
	}
	if query.End.IsZero() {
		query.End = time.Now()
	}
	if query.Start.IsZero() {
		query.Start = query.End.Add(-5 * time.Minute)
	}
	if strings.TrimSpace(query.Func) == "" {
		query.Func = "avg"
	}
	if query.WindowMin <= 0 {
		query.WindowMin = 5
	}

	qb := prominfra.NewQueryBuilder(query.Metric).
		WithAggregation(query.Func).
		WithRange(fmt.Sprintf("%dm", query.WindowMin))
	if strings.TrimSpace(query.Source) != "" {
		qb.WithLabel("source", strings.TrimSpace(query.Source))
	}
	res, err := l.svcCtx.Prometheus.Query(ctx, qb.Build(), query.End)
	if err != nil {
		return nil, err
	}
	if len(res.Vector) == 0 || len(res.Vector[0].Value) < 2 {
		return &AggregationResult{
			Metric:    query.Metric,
			Func:      strings.TrimSpace(query.Func),
			Source:    strings.TrimSpace(query.Source),
			Timestamp: query.End,
		}, nil
	}
	v, ok := toFloat64(res.Vector[0].Value[1])
	if !ok {
		return nil, fmt.Errorf("invalid aggregation value")
	}
	ts, ok := toFloat64(res.Vector[0].Value[0])
	if !ok {
		ts = float64(query.End.Unix())
	}
	return &AggregationResult{
		Metric:    query.Metric,
		Func:      strings.TrimSpace(query.Func),
		Source:    strings.TrimSpace(query.Source),
		Value:     v,
		Timestamp: time.Unix(int64(ts), 0).UTC(),
	}, nil
}

// GetMetricMetadata 查询指标元数据。
//
// 从 Prometheus 获取指标的元信息，如帮助文本、类型等。
// 支持缓存，缓存有效期 60 秒。
//
// 参数:
//   - ctx: 上下文
//   - metric: 指标名称
//
// 返回: 元数据列表和可能的错误
func (l *Logic) GetMetricMetadata(ctx context.Context, metric string) ([]prominfra.MetadataItem, error) {
	if l.svcCtx.Prometheus == nil {
		return nil, fmt.Errorf("prometheus client is unavailable")
	}
	cacheKey := fmt.Sprintf("monitoring:metadata:%s", strings.TrimSpace(metric))
	if l.svcCtx.CacheFacade != nil {
		if cached, ok := l.svcCtx.CacheFacade.Get(cacheKey); ok {
			items := make([]prominfra.MetadataItem, 0)
			if err := json.Unmarshal([]byte(cached), &items); err == nil {
				return items, nil
			}
		}
	}
	items, err := l.svcCtx.Prometheus.Metadata(ctx, strings.TrimSpace(metric))
	if err != nil {
		return nil, err
	}
	if l.svcCtx.CacheFacade != nil {
		if b, err := json.Marshal(items); err == nil {
			l.svcCtx.CacheFacade.Set(ctx, cacheKey, string(b), 60*time.Second)
		}
	}
	return items, nil
}

// toFloat64 将任意类型转换为 float64。
//
// 支持多种数值类型和字符串的转换。
//
// 参数:
//   - v: 待转换的值
//
// 返回: 转换后的 float64 值和转换是否成功的标志
func toFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// ListChannels 查询通知渠道列表。
//
// 首次查询时会自动初始化默认渠道。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 通知渠道列表和可能的错误
func (l *Logic) ListChannels(ctx context.Context) ([]model.AlertNotificationChannel, error) {
	if err := l.ensureDefaultChannels(ctx); err != nil {
		return nil, err
	}
	rows := make([]model.AlertNotificationChannel, 0, 16)
	err := l.svcCtx.DB.WithContext(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

// CreateChannel 创建通知渠道。
//
// 验证渠道类型和名称后持久化到数据库。
//
// 参数:
//   - ctx: 上下文
//   - channel: 通知渠道数据
//
// 返回: 创建成功的渠道和可能的错误
func (l *Logic) CreateChannel(ctx context.Context, channel model.AlertNotificationChannel) (*model.AlertNotificationChannel, error) {
	channel.Type = strings.ToLower(strings.TrimSpace(channel.Type))
	if channel.Type == "" {
		channel.Type = "log"
	}
	if strings.TrimSpace(channel.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}
	if _, err := buildNotifier(channel.Type); err != nil {
		return nil, err
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&channel).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}

// UpdateChannel 更新通知渠道。
//
// 支持部分字段更新，更新类型时会验证有效性。
//
// 参数:
//   - ctx: 上下文
//   - id: 渠道 ID
//   - payload: 更新字段映射
//
// 返回: 更新后的渠道和可能的错误
func (l *Logic) UpdateChannel(ctx context.Context, id uint, payload map[string]any) (*model.AlertNotificationChannel, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty update payload")
	}
	if v, ok := payload["type"]; ok {
		if s, ok := v.(string); ok {
			if _, err := buildNotifier(strings.ToLower(strings.TrimSpace(s))); err != nil {
				return nil, err
			}
			payload["type"] = strings.ToLower(strings.TrimSpace(s))
		}
	}
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertNotificationChannel{}).Where("id = ?", id).Updates(payload).Error; err != nil {
		return nil, err
	}
	var row model.AlertNotificationChannel
	if err := l.svcCtx.DB.WithContext(ctx).First(&row, id).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// ListDeliveries 查询告警投递记录列表。
//
// 支持按告警 ID、渠道类型和状态筛选，返回分页结果。
//
// 参数:
//   - ctx: 上下文
//   - alertID: 告警 ID 筛选 (0 表示不筛选)
//   - channelType: 渠道类型筛选 (可选)
//   - status: 投递状态筛选 (可选)
//   - page: 页码
//   - pageSize: 每页数量
//
// 返回: 投递记录列表、总数和可能的错误
func (l *Logic) ListDeliveries(ctx context.Context, alertID uint, channelType, status string, page, pageSize int) ([]model.AlertNotificationDelivery, int64, error) {
	q := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertNotificationDelivery{})
	if alertID > 0 {
		q = q.Where("alert_id = ?", alertID)
	}
	if strings.TrimSpace(channelType) != "" {
		q = q.Where("channel_type = ?", strings.TrimSpace(channelType))
	}
	if strings.TrimSpace(status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(status))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]model.AlertNotificationDelivery, 0, pageSize)
	offset := (page - 1) * pageSize
	if err := q.Order("id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// evaluateRules 评估告警规则。
//
// 根据传入的指标值评估所有启用的规则，
// 触发告警事件并发送通知。
//
// 参数:
//   - ctx: 上下文
//   - values: 指标值映射 (指标名 -> 值)
//
// 返回: 可能的错误
func (l *Logic) evaluateRules(ctx context.Context, values map[string]float64) error {
	rules := make([]model.AlertRule, 0, 32)
	if err := l.svcCtx.DB.WithContext(ctx).Where("enabled = 1").Find(&rules).Error; err != nil {
		return err
	}
	now := time.Now()
	for _, rule := range rules {
		val, ok := values[rule.Metric]
		if !ok {
			continue
		}
		triggered := compareValue(val, rule.Operator, rule.Threshold)
		source := fmt.Sprintf("%s/%s", rule.Source, rule.Metric)

		prevState := "normal"
		var firing model.AlertEvent
		err := l.svcCtx.DB.WithContext(ctx).
			Where("rule_id = ? AND source = ? AND status = ?", rule.ID, source, "firing").
			Order("id DESC").
			First(&firing).Error
		if err == nil {
			prevState = "firing"
		}
		if triggered && prevState != "firing" {
			event := model.AlertEvent{
				RuleID:      rule.ID,
				Title:       rule.Name,
				Message:     fmt.Sprintf("%s 当前值 %.2f，阈值 %.2f", rule.Metric, val, rule.Threshold),
				Metric:      rule.Metric,
				Value:       val,
				Threshold:   rule.Threshold,
				Severity:    normalizeSeverity(rule.Severity),
				Source:      source,
				Status:      "firing",
				TriggeredAt: now,
			}
			if err := l.svcCtx.DB.WithContext(ctx).Create(&event).Error; err != nil {
				return err
			}
			if err := l.deliverAlert(ctx, event); err != nil {
				return err
			}
			// 创建通知并推送
			integrator := notification.NewNotificationIntegrator(l.svcCtx.DB)
			go integrator.CreateAlertNotification(runtimectx.Detach(ctx), &event)
			continue
		}

		if !triggered && prevState == "firing" {
			if err := l.svcCtx.DB.WithContext(ctx).
				Model(&model.AlertEvent{}).
				Where("id = ?", firing.ID).
				Updates(map[string]any{"status": "resolved", "resolved_at": now}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// deliverAlert 发送告警通知。
//
// 向所有启用的通知渠道发送告警信息。
//
// 参数:
//   - ctx: 上下文
//   - alert: 告警事件
//
// 返回: 可能的错误
func (l *Logic) deliverAlert(ctx context.Context, alert model.AlertEvent) error {
	channels := make([]model.AlertNotificationChannel, 0, 8)
	if err := l.svcCtx.DB.WithContext(ctx).Where("enabled = 1").Find(&channels).Error; err != nil {
		return err
	}
	payload := NotificationPayload{
		AlertID:   alert.ID,
		RuleID:    alert.RuleID,
		Title:     alert.Title,
		Message:   alert.Message,
		Severity:  alert.Severity,
		Metric:    alert.Metric,
		Value:     alert.Value,
		Threshold: alert.Threshold,
	}
	for _, ch := range channels {
		notifier, err := buildNotifier(ch.Type)
		if err != nil {
			if err := l.recordDelivery(ctx, alert, ch, DeliveryResult{Status: "failed", Error: err.Error()}); err != nil {
				return err
			}
			continue
		}
		result := notifier.Send(ctx, ch, payload)
		if strings.TrimSpace(result.Status) == "" {
			result.Status = "sent"
		}
		if err := l.recordDelivery(ctx, alert, ch, result); err != nil {
			return err
		}
	}
	return nil
}

// recordDelivery 记录告警投递结果。
//
// 将投递信息写入数据库。
//
// 参数:
//   - ctx: 上下文
//   - alert: 告警事件
//   - channel: 通知渠道配置
//   - result: 投递结果
//
// 返回: 可能的错误
func (l *Logic) recordDelivery(ctx context.Context, alert model.AlertEvent, channel model.AlertNotificationChannel, result DeliveryResult) error {
	row := model.AlertNotificationDelivery{
		AlertID:      alert.ID,
		RuleID:       alert.RuleID,
		ChannelID:    channel.ID,
		ChannelType:  channel.Type,
		Target:       channel.Target,
		Status:       strings.TrimSpace(result.Status),
		ErrorMessage: strings.TrimSpace(result.Error),
		DeliveredAt:  time.Now(),
	}
	if row.Status == "" {
		row.Status = "sent"
	}
	return l.svcCtx.DB.WithContext(ctx).Create(&row).Error
}

// ensureDefaultRules 确保默认规则存在。
//
// 如果规则表为空，插入预设的默认告警规则。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 可能的错误
func (l *Logic) ensureDefaultRules(ctx context.Context) error {
	var count int64
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertRule{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	rules := []model.AlertRule{
		{Name: "主机 CPU 高使用", Metric: "cpu_usage", Operator: "gt", Threshold: 85, DurationSec: 300, WindowSec: 3600, GranularitySec: 60, Severity: "warning", Source: "host", Scope: "global", Enabled: true, State: "enabled"},
		{Name: "主机内存高使用", Metric: "memory_usage", Operator: "gt", Threshold: 90, DurationSec: 300, WindowSec: 3600, GranularitySec: 60, Severity: "critical", Source: "host", Scope: "global", Enabled: true, State: "enabled"},
		{Name: "主机磁盘高使用", Metric: "disk_usage", Operator: "gt", Threshold: 90, DurationSec: 600, WindowSec: 3600, GranularitySec: 60, Severity: "critical", Source: "host", Scope: "global", Enabled: true, State: "enabled"},
		{Name: "K8s 节点异常", Metric: "k8s_node_not_ready", Operator: "gt", Threshold: 0, DurationSec: 180, WindowSec: 3600, GranularitySec: 60, Severity: "critical", Source: "k8s", Scope: "global", Enabled: true, State: "enabled"},
		{Name: "Pod CrashLoopBackOff", Metric: "pod_crashloop_count", Operator: "gt", Threshold: 0, DurationSec: 180, WindowSec: 3600, GranularitySec: 60, Severity: "warning", Source: "k8s", Scope: "global", Enabled: true, State: "enabled"},
		{Name: "发布失败告警", Metric: "deploy_failed_count", Operator: "gt", Threshold: 0, DurationSec: 60, WindowSec: 3600, GranularitySec: 60, Severity: "critical", Source: "deploy", Scope: "global", Enabled: true, State: "enabled"},
	}
	for _, rule := range rules {
		item := rule
		if err := l.svcCtx.DB.WithContext(ctx).Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

// ensureDefaultChannels 确保默认通知渠道存在。
//
// 如果渠道表为空，插入默认的日志渠道。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 可能的错误
func (l *Logic) ensureDefaultChannels(ctx context.Context) error {
	var count int64
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.AlertNotificationChannel{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	row := model.AlertNotificationChannel{
		Name:     "default-log",
		Type:     "log",
		Provider: "builtin",
		Target:   "stdout",
		Enabled:  true,
	}
	return l.svcCtx.DB.WithContext(ctx).Create(&row).Error
}

// compareValue 比较值与阈值。
//
// 根据运算符执行比较操作。
//
// 参数:
//   - value: 当前值
//   - op: 比较运算符
//   - threshold: 阈值
//
// 返回: 比较结果
func compareValue(value float64, op string, threshold float64) bool {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "gt", ">":
		return value > threshold
	case "gte", ">=":
		return value >= threshold
	case "lt", "<":
		return value < threshold
	case "lte", "<=":
		return value <= threshold
	case "eq", "=":
		return value == threshold
	default:
		return value > threshold
	}
}

// normalizeSeverity 标准化严重级别。
//
// 将不同来源的严重级别映射为统一的三个级别。
//
// 参数:
//   - v: 原始严重级别字符串
//
// 返回: 标准化后的严重级别
func normalizeSeverity(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "critical", "warning", "info":
		return s
	default:
		return "warning"
	}
}

// boolToRuleState 将布尔值转换为规则状态字符串。
//
// 参数:
//   - enabled: 是否启用
//
// 返回: 状态字符串 (enabled/disabled)
func boolToRuleState(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
