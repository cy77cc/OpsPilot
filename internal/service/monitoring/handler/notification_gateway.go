// Package handler 提供监控告警服务的 HTTP 处理器。
//
// 本文件实现通知网关，负责接收 Alertmanager Webhook 请求，
// 处理告警事件并分发到各个通知渠道。
package handler

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	notifsvc "github.com/cy77cc/OpsPilot/internal/service/notification"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// AlertmanagerWebhook 是 Alertmanager Webhook 请求的载荷格式。
//
// 包含完整的告警信息，由 Prometheus Alertmanager 发送。
type AlertmanagerWebhook struct {
	Receiver          string              `json:"receiver"`           // 接收器名称
	Status            string              `json:"status"`             // 整体状态 (firing/resolved)
	Alerts            []AlertmanagerAlert `json:"alerts"`             // 告警列表
	GroupLabels       map[string]string   `json:"groupLabels"`        // 分组标签
	CommonLabels      map[string]string   `json:"commonLabels"`       // 公共标签
	CommonAnnotations map[string]string   `json:"commonAnnotations"`  // 公共注解
	ExternalURL       string              `json:"externalURL"`        // Alertmanager 外部 URL
}

// AlertmanagerAlert 是单个 Alertmanager 告警的结构。
//
// 包含告警的状态、标签、注解和时间信息。
type AlertmanagerAlert struct {
	Status       string            `json:"status"`       // 状态 (firing/resolved)
	Labels       map[string]string `json:"labels"`       // 标签
	Annotations  map[string]string `json:"annotations"`  // 注解
	StartsAt     time.Time         `json:"startsAt"`     // 触发时间
	EndsAt       time.Time         `json:"endsAt"`       // 结束时间
	GeneratorURL string            `json:"generatorURL"` // 生成器 URL
	Fingerprint  string            `json:"fingerprint"`  // 指纹
}

// NotificationGateway 是通知网关处理器。
//
// 接收 Alertmanager Webhook 请求，处理告警事件的生命周期，
// 并异步分发给所有启用的通知渠道。
type NotificationGateway struct {
	svcCtx    *svc.ServiceContext        // 服务上下文
	providers *notifsvc.ProviderRegistry // 通知提供者注册表
}

// NewNotificationGateway 创建通知网关实例。
//
// 初始化默认的通知提供者注册表。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: 初始化完成的 NotificationGateway 实例
func NewNotificationGateway(svcCtx *svc.ServiceContext) *NotificationGateway {
	return &NotificationGateway{
		svcCtx:    svcCtx,
		providers: notifsvc.NewDefaultProviderRegistry(),
	}
}

// HandleWebhook 处理 Alertmanager Webhook 请求。
//
// 遍历请求中的所有告警，创建或更新告警事件，
// 并异步分发通知。
//
// 参数:
//   - ctx: 上下文
//   - payload: Webhook 载荷
//
// 返回: 成功处理的告警数量和可能的错误
func (g *NotificationGateway) HandleWebhook(ctx context.Context, payload AlertmanagerWebhook) (int, error) {
	processed := 0
	for _, a := range payload.Alerts {
		event, err := g.upsertAlertEvent(ctx, a)
		if err != nil {
			return processed, err
		}
		processed++
		g.dispatchAsync(runtimectx.Detach(ctx), *event)
	}
	return processed, nil
}

// upsertAlertEvent 创建或更新告警事件。
//
// 根据告警指纹查找现有事件，存在则更新状态，不存在则创建新事件。
// 对于 resolved 状态的事件，会更新恢复时间。
//
// 参数:
//   - ctx: 上下文
//   - alert: Alertmanager 告警数据
//
// 返回: 创建或更新后的告警事件和可能的错误
func (g *NotificationGateway) upsertAlertEvent(ctx context.Context, alert AlertmanagerAlert) (*model.AlertEvent, error) {
	source := "alertmanager/" + strings.TrimSpace(alert.Fingerprint)
	if source == "alertmanager/" {
		source = "alertmanager/unknown"
	}

	status := strings.ToLower(strings.TrimSpace(alert.Status))
	if status == "" {
		status = "firing"
	}

	ruleID := uint(0)
	if v := strings.TrimSpace(alert.Labels["rule_id"]); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			ruleID = uint(n)
		}
	}

	title := strings.TrimSpace(alert.Labels["alertname"])
	if title == "" {
		title = "Prometheus Alert"
	}
	metric := strings.TrimSpace(alert.Labels["metric"])
	severity := normalizeSeverity(alert.Labels["severity"])
	message := strings.TrimSpace(alert.Annotations["summary"])
	if message == "" {
		message = strings.TrimSpace(alert.Annotations["description"])
	}
	if message == "" {
		message = title
	}

	var existed model.AlertEvent
	err := g.svcCtx.DB.WithContext(ctx).Where("source = ?", source).Order("id DESC").First(&existed).Error
	if err == nil {
		updates := map[string]any{
			"status":     status,
			"title":      title,
			"message":    message,
			"severity":   severity,
			"metric":     metric,
			"updated_at": time.Now(),
		}
		if status == "resolved" {
			resolvedAt := alert.EndsAt
			if resolvedAt.IsZero() {
				resolvedAt = time.Now()
			}
			updates["resolved_at"] = resolvedAt
		}
		if err := g.svcCtx.DB.WithContext(ctx).Model(&model.AlertEvent{}).Where("id = ?", existed.ID).Updates(updates).Error; err != nil {
			return nil, err
		}
		existed.Status = status
		existed.Title = title
		existed.Message = message
		existed.Severity = severity
		existed.Metric = metric
		return &existed, nil
	}

	event := model.AlertEvent{
		RuleID:      ruleID,
		Title:       title,
		Message:     message,
		Metric:      metric,
		Severity:    severity,
		Source:      source,
		Status:      status,
		TriggeredAt: alert.StartsAt,
	}
	if event.TriggeredAt.IsZero() {
		event.TriggeredAt = time.Now()
	}
	if status == "resolved" {
		resolvedAt := alert.EndsAt
		if resolvedAt.IsZero() {
			resolvedAt = time.Now()
		}
		event.ResolvedAt = &resolvedAt
	}
	if err := g.svcCtx.DB.WithContext(ctx).Create(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

// dispatchAsync 异步分发告警通知。
//
// 查询所有启用的通知渠道，并发发送通知。
// 使用 WaitGroup 确保所有发送任务都已启动。
//
// 参数:
//   - ctx: 上下文
//   - alert: 告警事件
func (g *NotificationGateway) dispatchAsync(ctx context.Context, alert model.AlertEvent) {
	channels := make([]model.AlertNotificationChannel, 0, 16)
	if err := g.svcCtx.DB.WithContext(ctx).Where("enabled = 1").Find(&channels).Error; err != nil {
		logger.L().Warn("load alert channels failed", logger.Error(err))
		return
	}
	if len(channels) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, ch := range channels {
		channel := ch
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.sendWithRetry(runtimectx.Detach(ctx), alert, channel)
		}()
	}
	go func() {
		wg.Wait()
	}()
}

// sendWithRetry 发送通知并支持重试。
//
// 根据渠道类型选择对应的通知提供者，失败时进行指数退避重试。
// 最多重试 3 次，重试间隔为 1s、2s、4s。
//
// 参数:
//   - ctx: 上下文
//   - alert: 告警事件
//   - channel: 通知渠道配置
func (g *NotificationGateway) sendWithRetry(ctx context.Context, alert model.AlertEvent, channel model.AlertNotificationChannel) {
	providerName := strings.TrimSpace(channel.Provider)
	if providerName == "" {
		providerName = strings.TrimSpace(channel.Type)
	}
	if providerName == "" {
		providerName = "log"
	}
	provider, ok := g.providers.Get(providerName)
	if !ok {
		provider, _ = g.providers.Get("log")
	}

	result := DeliveryResult{Status: "sent"}
	var lastErr error
	for i := 0; i < 3; i++ {
		err := provider.Send(ctx, &alert, channel)
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = err
		time.Sleep(time.Duration(1<<i) * time.Second)
	}
	if lastErr != nil {
		result.Status = "failed"
		result.Error = lastErr.Error()
	}
	if err := g.recordDelivery(ctx, alert, channel, result); err != nil {
		logger.L().Warn("record delivery failed", logger.Error(err))
	}
}

// recordDelivery 记录通知投递结果。
//
// 将投递状态、错误信息等写入数据库，用于后续查询和审计。
//
// 参数:
//   - ctx: 上下文
//   - alert: 告警事件
//   - channel: 通知渠道配置
//   - result: 投递结果
//
// 返回: 可能的错误
func (g *NotificationGateway) recordDelivery(ctx context.Context, alert model.AlertEvent, channel model.AlertNotificationChannel, result DeliveryResult) error {
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
	return g.svcCtx.DB.WithContext(ctx).Create(&row).Error
}

// normalizeSeverity 标准化严重级别。
//
// 将不同来源的严重级别映射为统一的三个级别：
// critical (critical/fatal/error)、warning (warning/warn)、info (info/notice)。
//
// 参数:
//   - v: 原始严重级别字符串
//
// 返回: 标准化后的严重级别
func normalizeSeverity(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "critical", "fatal", "error":
		return "critical"
	case "warning", "warn":
		return "warning"
	case "info", "notice":
		return "info"
	default:
		return "warning"
	}
}
