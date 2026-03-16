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
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected migration to contain %q", fragment)
		}
	}
}
