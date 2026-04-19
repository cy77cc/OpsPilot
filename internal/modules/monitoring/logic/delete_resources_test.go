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

func TestDeleteRule_ReturnsConflictWhenBindingExists(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:delete-rule-conflict?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertRule{}, &model.AlertRuleChannelBinding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertRule{
		ID: 7, Name: "cpu", Metric: "cpu_usage", Operator: "gt", Threshold: 80, Severity: "warning", Enabled: true, State: "enabled",
	}).Error; err != nil {
		t.Fatalf("seed alert rule: %v", err)
	}
	if err := db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Priority: 1, Enabled: true}).Error; err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	err = l.DeleteRule(context.Background(), 7)
	var conflict *DeleteConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected DeleteConflictError, got %v", err)
	}
	if len(conflict.Blockers) == 0 {
		t.Fatalf("expected blockers, got %#v", conflict)
	}

	var ruleRemain int64
	if err := db.Model(&model.AlertRule{}).Where("id = ?", 7).Count(&ruleRemain).Error; err != nil {
		t.Fatalf("count rule after conflict delete: %v", err)
	}
	if ruleRemain != 1 {
		t.Fatalf("expected rule to remain after conflict, got %d", ruleRemain)
	}
}

func TestDeleteChannel_ReturnsConflictWhenReferencedBySeverityRoute(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:delete-channel-conflict?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertNotificationChannel{}, &model.AlertSeverityRoute{}, &model.AlertRuleChannelBinding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertNotificationChannel{ID: 1001, Name: "webhook", Type: "webhook", Provider: "webhook", Enabled: true}).Error; err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	if err := db.Create(&model.AlertSeverityRoute{ID: 11, Scope: "global", Severity: "critical", ChannelIDsJSON: `[1001]`, Enabled: true}).Error; err != nil {
		t.Fatalf("seed route: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	err = l.DeleteChannel(context.Background(), 1001)
	var conflict *DeleteConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected DeleteConflictError, got %v", err)
	}
	if len(conflict.Blockers) == 0 {
		t.Fatalf("expected blockers, got %#v", conflict)
	}

	var channelRemain int64
	if err := db.Model(&model.AlertNotificationChannel{}).Where("id = ?", 1001).Count(&channelRemain).Error; err != nil {
		t.Fatalf("count channel after conflict delete: %v", err)
	}
	if channelRemain != 1 {
		t.Fatalf("expected channel to remain after conflict, got %d", channelRemain)
	}
}

func TestDeleteSeverityRoute_DeletesExactRow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:delete-route-success?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertSeverityRoute{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertSeverityRoute{ID: 31, Scope: "global", Severity: "warning", ChannelIDsJSON: `[1001]`, Enabled: true}).Error; err != nil {
		t.Fatalf("seed route: %v", err)
	}
	if err := db.Create(&model.AlertSeverityRoute{ID: 32, Scope: "global", Severity: "warning", ChannelIDsJSON: `[2002]`, Enabled: true}).Error; err != nil {
		t.Fatalf("seed sibling route: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	if err := l.DeleteSeverityRoute(context.Background(), 31, 0); err != nil {
		t.Fatalf("delete severity route: %v", err)
	}

	var deleted int64
	if err := db.Model(&model.AlertSeverityRoute{}).Where("id = ?", 31).Count(&deleted).Error; err != nil {
		t.Fatalf("count deleted route: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected route 31 to be deleted, got count=%d", deleted)
	}

	var sibling int64
	if err := db.Model(&model.AlertSeverityRoute{}).Where("id = ?", 32).Count(&sibling).Error; err != nil {
		t.Fatalf("count sibling route: %v", err)
	}
	if sibling != 1 {
		t.Fatalf("expected sibling route 32 to remain, got count=%d", sibling)
	}
}

func TestDeleteRuleChannelBinding_RespectsProjectScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:delete-binding-scope?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertRuleChannelBinding{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	projectID := uint(42)
	if err := db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, Enabled: true, Priority: 1}).Error; err != nil {
		t.Fatalf("seed global binding: %v", err)
	}
	if err := db.Create(&model.AlertRuleChannelBinding{RuleID: 7, ChannelID: 1001, ProjectID: &projectID, Enabled: true, Priority: 1}).Error; err != nil {
		t.Fatalf("seed project binding: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	if err := l.DeleteRuleChannelBinding(context.Background(), 42, 7, 1001); err != nil {
		t.Fatalf("delete project binding: %v", err)
	}

	var remain int64
	if err := db.Model(&model.AlertRuleChannelBinding{}).
		Where("rule_id = ? AND channel_id = ? AND project_id IS NULL", 7, 1001).
		Count(&remain).Error; err != nil {
		t.Fatalf("count global binding: %v", err)
	}
	if remain != 1 {
		t.Fatalf("expected global binding to remain, got %d", remain)
	}

	var projectRemain int64
	if err := db.Model(&model.AlertRuleChannelBinding{}).
		Where("rule_id = ? AND channel_id = ? AND project_id = ?", 7, 1001, 42).
		Count(&projectRemain).Error; err != nil {
		t.Fatalf("count project binding: %v", err)
	}
	if projectRemain != 0 {
		t.Fatalf("expected project binding to be deleted, got %d", projectRemain)
	}
}
