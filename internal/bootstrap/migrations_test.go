package bootstrap

import (
	"testing"
	"time"

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
			status TEXT NOT NULL,
			created_at DATETIME
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

func TestEnsureAIAlertHealJobEventIndex_DedupesExistingRowsBeforeIndexCreate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:bootstrap-ai-index-dedupe?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE ai_alert_heal_jobs (
			id TEXT PRIMARY KEY,
			event_id TEXT NOT NULL,
			scene TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create table without unique index: %v", err)
	}

	earliest := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC)
	later := earliest.Add(time.Hour)

	// For event-1, keep the earliest created_at row; tie-break by smallest id.
	if err := db.Exec(`
		INSERT INTO ai_alert_heal_jobs (id, event_id, scene, status, created_at) VALUES
			('b-id', 'event-1', 'scene-a', 'pending', ?),
			('a-id', 'event-1', 'scene-a', 'pending', ?),
			('c-id', 'event-1', 'scene-a', 'pending', ?),
			('z-id', 'event-2', 'scene-b', 'pending', ?)
	`, earliest, earliest, later, earliest).Error; err != nil {
		t.Fatalf("seed duplicate rows: %v", err)
	}

	if err := ensureAIAlertHealJobEventIndex(db); err != nil {
		t.Fatalf("ensure index with dedupe: %v", err)
	}

	type eventCount struct {
		EventID string `gorm:"column:event_id"`
		Count   int    `gorm:"column:cnt"`
	}
	var counts []eventCount
	if err := db.Table("ai_alert_heal_jobs").
		Select("event_id, COUNT(*) AS cnt").
		Group("event_id").
		Order("event_id ASC").
		Find(&counts).Error; err != nil {
		t.Fatalf("query grouped counts: %v", err)
	}
	if len(counts) != 2 {
		t.Fatalf("expected 2 event groups after dedupe, got %d", len(counts))
	}
	for _, c := range counts {
		if c.Count != 1 {
			t.Fatalf("expected one row per event_id, event=%s count=%d", c.EventID, c.Count)
		}
	}

	var keptID string
	if err := db.Table("ai_alert_heal_jobs").
		Select("id").
		Where("event_id = ?", "event-1").
		Scan(&keptID).Error; err != nil {
		t.Fatalf("query kept id: %v", err)
	}
	if keptID != "a-id" {
		t.Fatalf("expected deterministic keeper id a-id, got %s", keptID)
	}

	if !db.Migrator().HasIndex(&aimodel.AIAlertHealJob{}, "uk_ai_alert_heal_jobs_event_id") {
		t.Fatal("expected safeguard to create uk_ai_alert_heal_jobs_event_id")
	}
}
