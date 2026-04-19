package logic

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResolveChannels_BindingWinsSeverityFallback(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:routing-precedence?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertRuleChannelBinding{}, &model.AlertSeverityRoute{}, &model.AlertNotificationChannel{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertNotificationChannel{ID: 1001, Name: "bound", Type: "log", Provider: "log", Enabled: true}).Error; err != nil {
		t.Fatalf("seed bound channel: %v", err)
	}
	if err := db.Create(&model.AlertNotificationChannel{ID: 2001, Name: "fallback", Type: "log", Provider: "log", Enabled: true}).Error; err != nil {
		t.Fatalf("seed fallback channel: %v", err)
	}
	if err := db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Priority: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}
	if err := db.Create(&model.AlertSeverityRoute{Scope: "global", Severity: "critical", ChannelIDsJSON: `[2001]`, Enabled: true}).Error; err != nil {
		t.Fatalf("seed route: %v", err)
	}

	logic := NewLogic(&svc.ServiceContext{DB: db})
	channels, err := logic.ResolveChannelsForAlert(context.Background(), 0, 7, "critical")
	if err != nil {
		t.Fatalf("resolve channels: %v", err)
	}
	if len(channels) != 1 || channels[0].ID != 1001 {
		t.Fatalf("expected bound channel 1001, got %#v", channels)
	}
}
