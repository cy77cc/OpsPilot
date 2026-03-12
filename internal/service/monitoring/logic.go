package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/notification"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

type Logic struct {
	svcCtx *svc.ServiceContext
}

type MetricQuery struct {
	Metric         string
	Start          time.Time
	End            time.Time
	GranularitySec int
	Source         string
}

type AggregationQuery struct {
	Metric    string
	Func      string
	Start     time.Time
	End       time.Time
	Source    string
	WindowMin int
}

type MetricQueryResult struct {
	Window struct {
		Start          time.Time `json:"start"`
		End            time.Time `json:"end"`
		GranularitySec int       `json:"granularity_sec"`
	} `json:"window"`
	Dimensions map[string]any   `json:"dimensions"`
	Series     []map[string]any `json:"series"`
}

type AggregationResult struct {
	Metric    string    `json:"metric"`
	Func      string    `json:"func"`
	Source    string    `json:"source"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{svcCtx: svcCtx}
}

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

func (l *Logic) SetRuleEnabled(ctx context.Context, id uint, enabled bool) (*model.AlertRule, error) {
	payload := map[string]any{
		"enabled": enabled,
		"state":   boolToRuleState(enabled),
	}
	return l.UpdateRule(ctx, id, payload)
}

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

func (l *Logic) ListChannels(ctx context.Context) ([]model.AlertNotificationChannel, error) {
	if err := l.ensureDefaultChannels(ctx); err != nil {
		return nil, err
	}
	rows := make([]model.AlertNotificationChannel, 0, 16)
	err := l.svcCtx.DB.WithContext(ctx).Order("id ASC").Find(&rows).Error
	return rows, err
}

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
			go integrator.CreateAlertNotification(context.Background(), &event)
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

func normalizeSeverity(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "critical", "warning", "info":
		return s
	default:
		return "warning"
	}
}

func boolToRuleState(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
