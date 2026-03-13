package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAIApprovalMigrationContainsRequiredTables(t *testing.T) {
	path := filepath.Join("..", "migrations", "20260305_000029_ai_confirmation_and_approval.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration file failed: %v", err)
	}

	sql := strings.ToLower(string(content))
	required := []string{
		"create table if not exists ai_confirmations",
		"create table if not exists ai_approval_tickets",
	}
	for _, item := range required {
		if !strings.Contains(sql, item) {
			t.Fatalf("migration missing required sql %q", item)
		}
	}
}

func TestAIModuleRedesignMigrationContainsRequiredTables(t *testing.T) {
	path := filepath.Join("..", "migrations", "20260313_000036_ai_module_redesign_backend.sql")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration file failed: %v", err)
	}

	sql := strings.ToLower(string(content))
	required := []string{
		"create table if not exists ai_scene_configs",
		"create table if not exists ai_approvals",
		"create table if not exists ai_executions",
		"insert into ai_scene_configs",
	}
	for _, item := range required {
		if !strings.Contains(sql, item) {
			t.Fatalf("migration missing required sql %q", item)
		}
	}
}
