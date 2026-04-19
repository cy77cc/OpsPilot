package handler

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
)

func TestEmailProvider_Send_RequiresGlobalSMTPConfig(t *testing.T) {
	original := config.CFG
	t.Cleanup(func() {
		config.CFG = original
	})

	config.CFG.Notification.SMTP = config.SMTP{}

	provider := &EmailProvider{}
	channel := monitoringmodel.AlertNotificationChannel{Name: "mail", Provider: "email"}
	err := provider.Send(context.Background(), &monitoringmodel.AlertEvent{Title: "CPU high"}, channel)
	if err == nil {
		t.Fatal("expected missing smtp config error")
	}
}
