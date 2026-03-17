package migration

import (
	"os"
	"strings"
	"testing"
)

func TestAIPhase1Migration_DefinesRunAndDiagnosisReportTables(t *testing.T) {
	content, err := os.ReadFile("../migrations/20260316_000039_ai_phase1_runs_and_reports.sql")
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sql := string(content)
	for _, fragment := range []string{
		"CREATE TABLE IF NOT EXISTS ai_runs",
		"CREATE TABLE IF NOT EXISTS ai_diagnosis_reports",
		"session_id VARCHAR(64) NOT NULL",
		"status VARCHAR(32) NOT NULL",
		"run_id VARCHAR(64) NOT NULL",
		"INDEX idx_ai_runs_session_id (session_id)",
		"INDEX idx_ai_runs_status (status)",
		"INDEX idx_ai_diagnosis_reports_run_id (run_id)",
		"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected migration to contain %q", fragment)
		}
	}
}

func TestAIPhase1UTF8MB4Migration_ConvertsExistingTables(t *testing.T) {
	content, err := os.ReadFile("../migrations/20260317_000040_ai_phase1_utf8mb4.sql")
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}

	sql := string(content)
	for _, fragment := range []string{
		"ALTER TABLE ai_runs",
		"ALTER TABLE ai_diagnosis_reports",
		"CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected utf8mb4 migration to contain %q", fragment)
		}
	}
}
