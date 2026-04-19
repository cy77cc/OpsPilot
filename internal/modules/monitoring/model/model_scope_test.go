package model_test

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMonitoringModels_AutoMigrateScopedTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:monitoring-model-scope?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	err = db.AutoMigrate(
		&model.AlertRule{},
		&model.AlertNotificationChannel{},
		&model.AlertRuleChannelBinding{},
		&model.AlertSeverityRoute{},
	)
	if err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
}
