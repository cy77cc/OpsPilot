package bootstrap

import (
	"testing"

	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEnsureAIAlertHealJobEventIndex_NoTableIsNoop(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bootstrap-ai-index-no-table?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := ensureAIAlertHealJobEventIndex(db); err != nil {
		t.Fatalf("ensure index without table should be noop: %v", err)
	}
}

func TestEnsureAIAlertHealJobEventIndex_CreatesMissingIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bootstrap-ai-index-create?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE ai_alert_heal_jobs (
			id TEXT PRIMARY KEY,
			event_id TEXT NOT NULL,
			scene TEXT NOT NULL,
			status TEXT NOT NULL
		)
	`).Error; err != nil {
		t.Fatalf("create table without unique index: %v", err)
	}

	if db.Migrator().HasIndex(&aimodel.AIAlertHealJob{}, "uk_ai_alert_heal_jobs_event_id") {
		t.Fatal("expected index to be absent before safeguard")
	}
	if err := ensureAIAlertHealJobEventIndex(db); err != nil {
		t.Fatalf("ensure index: %v", err)
	}
	if !db.Migrator().HasIndex(&aimodel.AIAlertHealJob{}, "uk_ai_alert_heal_jobs_event_id") {
		t.Fatal("expected safeguard to create uk_ai_alert_heal_jobs_event_id")
	}
}
