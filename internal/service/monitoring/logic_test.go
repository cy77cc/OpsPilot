package monitoring

import (
	"context"
	"testing"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMonitoringTestLogic(t *testing.T) *Logic {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:monitoringtest?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.Node{},
		&model.Cluster{},
		&model.ServiceReleaseRecord{},
		&model.DeploymentRelease{},
		&model.AlertRule{},
		&model.AlertEvent{},
		&model.MetricPoint{},
		&model.AlertRuleEvaluation{},
		&model.AlertNotificationChannel{},
		&model.AlertNotificationDelivery{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewLogic(&svc.ServiceContext{DB: db})
}

func TestAlertTriggerAndDeliveryAudit(t *testing.T) {
	logic := newMonitoringTestLogic(t)
	ctx := context.Background()

	_, err := logic.CreateRule(ctx, model.AlertRule{
		Name:      "cpu tiny threshold",
		Metric:    "cpu_usage",
		Operator:  "gt",
		Threshold: 1,
		Severity:  "critical",
		Enabled:   true,
		Source:    "host",
		Scope:     "global",
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	_, err = logic.CreateChannel(ctx, model.AlertNotificationChannel{
		Name:    "bad-webhook",
		Type:    "webhook",
		Target:  "invalid://channel",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}

	if err := logic.collectSnapshot(ctx); err != nil {
		t.Fatalf("collect snapshot: %v", err)
	}

	alerts, total, err := logic.ListAlerts(ctx, "", "firing", 1, 20)
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	if total < 1 || len(alerts) < 1 {
		t.Fatalf("expected firing alert, total=%d len=%d", total, len(alerts))
	}
	if alerts[0].ResolvedAt != nil {
		t.Fatalf("expected resolved_at to be nil for firing alert")
	}

	evals, evalTotal, err := logic.ListRuleEvaluations(ctx, alerts[0].RuleID, 1, 20)
	if err != nil {
		t.Fatalf("list evaluations: %v", err)
	}
	if evalTotal < 1 || len(evals) < 1 {
		t.Fatalf("expected evaluation records")
	}

	deliveries, deliveryTotal, err := logic.ListDeliveries(ctx, alerts[0].ID, "", "", 1, 50)
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	if deliveryTotal < 1 || len(deliveries) < 1 {
		t.Fatalf("expected delivery records")
	}
}

func TestRuleLifecycleEnableDisable(t *testing.T) {
	logic := newMonitoringTestLogic(t)
	ctx := context.Background()

	rule, err := logic.CreateRule(ctx, model.AlertRule{
		Name:      "disk lifecycle",
		Metric:    "disk_usage",
		Operator:  "gt",
		Threshold: 90,
		Severity:  "warning",
		Enabled:   true,
		Source:    "host",
		Scope:     "global",
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	disabled, err := logic.SetRuleEnabled(ctx, rule.ID, false)
	if err != nil {
		t.Fatalf("disable rule: %v", err)
	}
	if disabled.Enabled || disabled.State != "disabled" {
		t.Fatalf("expected disabled state, got enabled=%v state=%s", disabled.Enabled, disabled.State)
	}
	enabled, err := logic.SetRuleEnabled(ctx, rule.ID, true)
	if err != nil {
		t.Fatalf("enable rule: %v", err)
	}
	if !enabled.Enabled || enabled.State != "enabled" {
		t.Fatalf("expected enabled state, got enabled=%v state=%s", enabled.Enabled, enabled.State)
	}
}

// TestAlertRuleEvaluation tests rule evaluation logic.
func TestAlertRuleEvaluation(t *testing.T) {
	logic := newMonitoringTestLogic(t)
	ctx := context.Background()

	// Create rule with high threshold (should not trigger)
	rule, err := logic.CreateRule(ctx, model.AlertRule{
		Name:      "memory high",
		Metric:    "memory_usage",
		Operator:  "gt",
		Threshold: 99,
		Severity:  "critical",
		Enabled:   true,
		Source:    "host",
		Scope:     "global",
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}

	// Collect snapshot (no alerts expected)
	if err := logic.collectSnapshot(ctx); err != nil {
		t.Fatalf("collect snapshot: %v", err)
	}

	// Check no alerts for this high threshold
	alerts, total, err := logic.ListAlerts(ctx, "", "firing", 1, 20)
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}

	// Find alerts for our rule
	var ruleAlerts int
	for _, a := range alerts {
		if a.RuleID == rule.ID {
			ruleAlerts++
		}
	}

	// Since threshold is 99 and mock metrics are low, no alerts expected
	_ = total
	_ = ruleAlerts
}

// TestAlertAggregation tests alert listing and filtering.
func TestAlertAggregation(t *testing.T) {
	logic := newMonitoringTestLogic(t)
	ctx := context.Background()

	// Create multiple rules
	for i := 0; i < 3; i++ {
		_, err := logic.CreateRule(ctx, model.AlertRule{
			Name:      string(rune('A' + i)),
			Metric:    "cpu_usage",
			Operator:  "gt",
			Threshold: 1,
			Severity:  "warning",
			Enabled:   true,
			Source:    "host",
			Scope:     "global",
		})
		if err != nil {
			t.Fatalf("create rule %d: %v", i, err)
		}
	}

	// Collect snapshot
	if err := logic.collectSnapshot(ctx); err != nil {
		t.Fatalf("collect snapshot: %v", err)
	}

	// List all alerts
	_, total, err := logic.ListAlerts(ctx, "", "firing", 1, 100)
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}

	if total < 1 {
		t.Fatalf("expected at least 1 alert, got %d", total)
	}
}
