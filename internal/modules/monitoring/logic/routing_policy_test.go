package logic

import (
	"context"
	"errors"
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

func TestUpdateSeverityRoute_ScopedSuccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:update-route-scoped-success?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertSeverityRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	projectID := uint(42)
	if err := db.Create(&model.AlertSeverityRoute{
		ID: 101, Scope: "project", ProjectID: &projectID, Severity: "warning", ChannelIDsJSON: `[1001]`, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed project route: %v", err)
	}
	if err := db.Create(&model.AlertSeverityRoute{
		ID: 102, Scope: "global", Severity: "critical", ChannelIDsJSON: `[2002]`, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed global route: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	updated, err := l.UpdateSeverityRoute(context.Background(), 101, 42, SeverityRouteInput{
		Scope:      "project",
		Severity:   "critical",
		ChannelIDs: []uint{3003},
		Enabled:    false,
	})
	if err != nil {
		t.Fatalf("update severity route: %v", err)
	}
	if updated.ID != 101 {
		t.Fatalf("expected updated id=101, got %d", updated.ID)
	}
	if updated.ProjectID == nil || *updated.ProjectID != 42 {
		t.Fatalf("expected updated route in project 42, got %#v", updated.ProjectID)
	}
	if updated.Severity != "critical" {
		t.Fatalf("expected severity critical, got %q", updated.Severity)
	}
	if updated.ChannelIDsJSON != `[3003]` {
		t.Fatalf("expected channels [3003], got %s", updated.ChannelIDsJSON)
	}
	if updated.Enabled {
		t.Fatalf("expected enabled=false after update")
	}
}

func TestUpdateSeverityRoute_ReturnsNotFoundOnScopeMismatch(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:update-route-scope-mismatch?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertSeverityRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	projectID := uint(42)
	if err := db.Create(&model.AlertSeverityRoute{
		ID: 201, Scope: "project", ProjectID: &projectID, Severity: "warning", ChannelIDsJSON: `[1001]`, Enabled: true,
	}).Error; err != nil {
		t.Fatalf("seed project route: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	updated, err := l.UpdateSeverityRoute(context.Background(), 201, 7, SeverityRouteInput{
		Scope:      "project",
		Severity:   "critical",
		ChannelIDs: []uint{9009},
		Enabled:    false,
	})
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got row=%#v err=%v", updated, err)
	}

	var row model.AlertSeverityRoute
	if err := db.Where("id = ?", 201).Take(&row).Error; err != nil {
		t.Fatalf("refetch seeded route: %v", err)
	}
	if row.Severity != "warning" || row.ChannelIDsJSON != `[1001]` || !row.Enabled {
		t.Fatalf("expected route unchanged on scope mismatch, got %#v", row)
	}
}
