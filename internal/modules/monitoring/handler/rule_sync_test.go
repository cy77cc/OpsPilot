package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRuleSyncService_SyncRules_BidirectionalUnion(t *testing.T) {
	db := newRuleSyncTestDB(t)
	if err := db.Create(&monitoringmodel.AlertRule{
		Name:           "DBOnlyRule",
		Metric:         "cpu_usage",
		Operator:       "gt",
		Threshold:      90,
		DurationSec:    120,
		Severity:       "warning",
		Source:         "host",
		Scope:          "global",
		WindowSec:      3600,
		GranularitySec: 60,
		Enabled:        true,
		State:          "enabled",
	}).Error; err != nil {
		t.Fatalf("seed db-only rule: %v", err)
	}

	var reloadCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/rules":
			payload := map[string]any{
				"status": "success",
				"data": map[string]any{
					"groups": []map[string]any{
						{
							"name": "external",
							"rules": []map[string]any{
								{
									"type":     "alerting",
									"name":     "PromOnlyRule",
									"query":    "node_filesystem_avail_bytes < 1024",
									"duration": 120,
									"labels": map[string]string{
										"severity": "critical",
										"team":     "ops",
									},
									"annotations": map[string]string{
										"summary": "external rule",
									},
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "/-/reload":
			reloadCalls++
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	rulesFile := filepath.Join(t.TempDir(), "alerting_rules.yml")
	restore := setRuleSyncTestConfig(t, srv.URL, rulesFile)
	defer restore()

	svc := NewRuleSyncService(db)
	n, err := svc.SyncRules(context.Background())
	if err != nil {
		t.Fatalf("sync rules: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 managed rule exported, got %d", n)
	}
	if reloadCalls != 1 {
		t.Fatalf("expected reload call once, got %d", reloadCalls)
	}

	var imported monitoringmodel.AlertRule
	if err := db.Where("name = ?", "PromOnlyRule").First(&imported).Error; err != nil {
		t.Fatalf("query imported rule: %v", err)
	}
	if imported.Source != importedPrometheusSource {
		t.Fatalf("expected imported source %q, got %q", importedPrometheusSource, imported.Source)
	}
	if imported.PromQLExpr != "node_filesystem_avail_bytes < 1024" {
		t.Fatalf("unexpected imported promql expr: %q", imported.PromQLExpr)
	}

	content, err := os.ReadFile(rulesFile)
	if err != nil {
		t.Fatalf("read rules file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "DBOnlyRule") {
		t.Fatalf("managed file should contain db-only rule, got %s", text)
	}
	if strings.Contains(text, "PromOnlyRule") {
		t.Fatalf("managed file should not duplicate imported prometheus rule, got %s", text)
	}
}

func TestRuleSyncService_SyncRules_PrometheusWinsOnSameName(t *testing.T) {
	db := newRuleSyncTestDB(t)
	if err := db.Create(&monitoringmodel.AlertRule{
		Name:           "SharedRule",
		Metric:         "cpu_usage",
		Operator:       "gt",
		Threshold:      95,
		DurationSec:    60,
		Severity:       "warning",
		Source:         "custom",
		Scope:          "global",
		WindowSec:      3600,
		GranularitySec: 60,
		Enabled:        true,
		State:          "enabled",
	}).Error; err != nil {
		t.Fatalf("seed shared rule: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/rules":
			payload := map[string]any{
				"status": "success",
				"data": map[string]any{
					"groups": []map[string]any{
						{
							"name": "external",
							"rules": []map[string]any{
								{
									"type":     "alerting",
									"name":     "SharedRule",
									"query":    "cpu_usage > 80",
									"duration": 90,
									"labels": map[string]string{
										"severity": "critical",
									},
									"annotations": map[string]string{
										"summary": "prometheus wins",
									},
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "/-/reload":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	restore := setRuleSyncTestConfig(t, srv.URL, filepath.Join(t.TempDir(), "alerting_rules.yml"))
	defer restore()

	svc := NewRuleSyncService(db)
	if _, err := svc.SyncRules(context.Background()); err != nil {
		t.Fatalf("sync rules: %v", err)
	}

	var row monitoringmodel.AlertRule
	if err := db.Where("name = ?", "SharedRule").First(&row).Error; err != nil {
		t.Fatalf("query shared rule: %v", err)
	}
	if row.Source != importedPrometheusSource {
		t.Fatalf("expected source overwritten by prometheus, got %q", row.Source)
	}
	if row.PromQLExpr != "cpu_usage > 80" {
		t.Fatalf("expected promql expr to follow prometheus, got %q", row.PromQLExpr)
	}
	if row.Metric != "cpu_usage" || row.Operator != "gt" || row.Threshold != 80 {
		t.Fatalf("expected parsed threshold rule from prometheus, got metric=%q operator=%q threshold=%v", row.Metric, row.Operator, row.Threshold)
	}
}

func TestRuleSyncService_SyncRules_ComplexExprKeepsLocalThresholdFields(t *testing.T) {
	db := newRuleSyncTestDB(t)
	if err := db.Create(&monitoringmodel.AlertRule{
		Name:           "ComplexSharedRule",
		Metric:         "http_error_rate",
		Operator:       "gt",
		Threshold:      0.1,
		DurationSec:    120,
		Severity:       "warning",
		Source:         "custom",
		Scope:          "global",
		WindowSec:      3600,
		GranularitySec: 60,
		Enabled:        true,
		State:          "enabled",
	}).Error; err != nil {
		t.Fatalf("seed complex shared rule: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/rules":
			payload := map[string]any{
				"status": "success",
				"data": map[string]any{
					"groups": []map[string]any{
						{
							"name": "external",
							"rules": []map[string]any{
								{
									"type":     "alerting",
									"name":     "ComplexSharedRule",
									"query":    "sum(rate(http_requests_total[5m])) > 0.2",
									"duration": 300,
									"labels": map[string]string{
										"severity": "warning",
									},
									"annotations": map[string]string{
										"summary": "complex query",
									},
								},
							},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "/-/reload":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	restore := setRuleSyncTestConfig(t, srv.URL, filepath.Join(t.TempDir(), "alerting_rules.yml"))
	defer restore()

	svc := NewRuleSyncService(db)
	if _, err := svc.SyncRules(context.Background()); err != nil {
		t.Fatalf("sync rules: %v", err)
	}

	var row monitoringmodel.AlertRule
	if err := db.Where("name = ?", "ComplexSharedRule").First(&row).Error; err != nil {
		t.Fatalf("query complex shared rule: %v", err)
	}
	if row.PromQLExpr != "sum(rate(http_requests_total[5m])) > 0.2" {
		t.Fatalf("expected promql expr updated from prometheus, got %q", row.PromQLExpr)
	}
	if row.Metric != "http_error_rate" || row.Operator != "gt" || row.Threshold != 0.1 {
		t.Fatalf("expected local threshold fields preserved for complex expr, got metric=%q operator=%q threshold=%v", row.Metric, row.Operator, row.Threshold)
	}
}

func newRuleSyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:monitoring-rule-sync-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&monitoringmodel.AlertRule{}); err != nil {
		t.Fatalf("migrate alert_rules: %v", err)
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&monitoringmodel.AlertRule{}).Error; err != nil {
		t.Fatalf("clear alert_rules: %v", err)
	}
	return db
}

func setRuleSyncTestConfig(t *testing.T, address, rulesFile string) func() {
	t.Helper()
	oldAddress := config.CFG.Prometheus.Address
	oldRulesFile := os.Getenv("PROMETHEUS_ALERTING_RULES_FILE")
	config.CFG.Prometheus.Address = address
	if err := os.Setenv("PROMETHEUS_ALERTING_RULES_FILE", rulesFile); err != nil {
		t.Fatalf("set rules file env: %v", err)
	}
	return func() {
		config.CFG.Prometheus.Address = oldAddress
		if oldRulesFile == "" {
			_ = os.Unsetenv("PROMETHEUS_ALERTING_RULES_FILE")
			return
		}
		_ = os.Setenv("PROMETHEUS_ALERTING_RULES_FILE", oldRulesFile)
	}
}
