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

// AlertmanagerWebhook is the payload format sent by Alertmanager webhook receiver.
type AlertmanagerWebhook struct {
	Receiver          string              `json:"receiver"`
	Status            string              `json:"status"`
	Alerts            []AlertmanagerAlert `json:"alerts"`
	GroupLabels       map[string]string   `json:"groupLabels"`
	CommonLabels      map[string]string   `json:"commonLabels"`
	CommonAnnotations map[string]string   `json:"commonAnnotations"`
	ExternalURL       string              `json:"externalURL"`
}

type AlertmanagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

type NotificationGateway struct {
	svcCtx    *svc.ServiceContext
	providers *notifsvc.ProviderRegistry
}

func NewNotificationGateway(svcCtx *svc.ServiceContext) *NotificationGateway {
	return &NotificationGateway{
		svcCtx:    svcCtx,
		providers: notifsvc.NewDefaultProviderRegistry(),
	}
}

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
