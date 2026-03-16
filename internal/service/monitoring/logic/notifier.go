package logic

import (
	"context"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
)

type NotificationPayload struct {
	AlertID   uint
	RuleID    uint
	Title     string
	Message   string
	Severity  string
	Metric    string
	Value     float64
	Threshold float64
}

type DeliveryResult struct {
	Status string
	Error  string
}

type Notifier interface {
	Send(ctx context.Context, channel model.AlertNotificationChannel, payload NotificationPayload) DeliveryResult
}

type logNotifier struct{}

func (n *logNotifier) Send(_ context.Context, _ model.AlertNotificationChannel, _ NotificationPayload) DeliveryResult {
	return DeliveryResult{Status: "sent"}
}

type webhookNotifier struct{}

func (n *webhookNotifier) Send(_ context.Context, channel model.AlertNotificationChannel, payload NotificationPayload) DeliveryResult {
	target := strings.TrimSpace(channel.Target)
	if target == "" {
		return DeliveryResult{Status: "failed", Error: "webhook target is empty"}
	}
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return DeliveryResult{Status: "failed", Error: "invalid webhook url"}
	}
	_ = payload
	return DeliveryResult{Status: "sent"}
}

func buildNotifier(channelType string) (Notifier, error) {
	switch strings.ToLower(strings.TrimSpace(channelType)) {
	case "", "log":
		return &logNotifier{}, nil
	case "webhook":
		return &webhookNotifier{}, nil
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}
