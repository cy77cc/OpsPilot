package logic

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListEffectiveRules_ProjectOverrideWins(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:effective-rules?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AlertRule{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&model.AlertRule{
		Name: "CPU High", Metric: "cpu_usage", Operator: "gt", Threshold: 85, Severity: "warning",
		Enabled: true, State: "enabled", Scope: "global", InheritKey: "cpu-high",
	}).Error; err != nil {
		t.Fatalf("seed global rule: %v", err)
	}
	projectID := uint(42)
	if err := db.Create(&model.AlertRule{
		Name: "CPU High Project", Metric: "cpu_usage", Operator: "gt", Threshold: 92, Severity: "warning",
		Enabled: true, State: "enabled", Scope: "project", ProjectID: &projectID, InheritKey: "cpu-high", IsOverride: true,
	}).Error; err != nil {
		t.Fatalf("seed project rule: %v", err)
	}

	l := NewLogic(&svc.ServiceContext{DB: db})
	rules, _, err := l.ListEffectiveRules(context.Background(), 42, 1, 50)
	if err != nil {
		t.Fatalf("ListEffectiveRules: %v", err)
	}
	var got model.AlertRule
	for _, row := range rules {
		if row.InheritKey == "cpu-high" {
			got = row
			break
		}
	}
	if got.Threshold != 92 {
		t.Fatalf("expected override threshold 92, got %.2f", got.Threshold)
	}
}
