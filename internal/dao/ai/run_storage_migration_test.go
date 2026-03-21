package ai

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/storage/migration"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAIRunStorageTablesExist(t *testing.T) {
	db := newRunStorageMigrationTestDB(t)

	for _, table := range []string{"ai_run_events", "ai_run_projections", "ai_run_contents"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected %s table", table)
		}
	}
}

func TestRunMigration_AddsClientRequestIDAndExpiryFields(t *testing.T) {
	db := newRunStorageMigrationTestDB(t)

	for _, column := range []string{"client_request_id", "last_event_at"} {
		if !db.Migrator().HasColumn(&model.AIRun{}, column) {
			t.Fatalf("expected ai_runs.%s column", column)
		}
	}
	if !db.Migrator().HasIndex(&model.AIRun{}, "uk_ai_runs_session_request") {
		t.Fatal("expected ai_runs unique index uk_ai_runs_session_request")
	}

	scriptBytes, err := os.ReadFile("../../../storage/migrations/20260320_0003_add_ai_failed_session_persistence.sql")
	if err != nil {
		t.Fatalf("read migration script: %v", err)
	}
	script := string(scriptBytes)
	for _, fragment := range []string{
		"ADD COLUMN client_request_id",
		"ADD COLUMN last_event_at",
		"ADD UNIQUE KEY uk_ai_runs_session_request",
		"UPDATE ai_runs\nSET client_request_id = id",
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("expected migration script to contain %q", fragment)
		}
	}
}

func newRunStorageMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := migration.RunDevAutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
