package handler

import (
	"context"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNotificationGateway_DispatchesOnlyResolvedChannels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:gateway-routing?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AlertEvent{},
		&model.AlertNotificationChannel{},
		&model.AlertRuleChannelBinding{},
		&model.AlertSeverityRoute{},
		&model.AlertNotificationDelivery{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertNotificationChannel{ID: 1001, Name: "bound", Provider: "log", Type: "log", Enabled: true}).Error; err != nil {
		t.Fatalf("seed bound channel: %v", err)
	}
	if err := db.Create(&model.AlertNotificationChannel{ID: 2001, Name: "fallback", Provider: "log", Type: "log", Enabled: true}).Error; err != nil {
		t.Fatalf("seed fallback channel: %v", err)
	}
	if err := db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Priority: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}
	alert := model.AlertEvent{ID: 5001, RuleID: 7, Severity: "critical", Status: "firing", Source: "alertmanager/fp-1", TriggeredAt: time.Now().UTC()}
	if err := db.Create(&alert).Error; err != nil {
		t.Fatalf("seed alert: %v", err)
	}

	gw := NewNotificationGateway(&svc.ServiceContext{DB: db})
	gw.dispatchAsync(context.Background(), alert)
	time.Sleep(50 * time.Millisecond)

	var rows []model.AlertNotificationDelivery
	if err := db.Where("alert_id = ?", alert.ID).Order("channel_id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("query deliveries: %v", err)
	}
	if len(rows) != 1 || rows[0].ChannelID != 1001 {
		t.Fatalf("expected only bound channel delivery, got %#v", rows)
	}
}
