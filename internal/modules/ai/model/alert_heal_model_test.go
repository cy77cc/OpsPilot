package model

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAIAlertIngestEvent_DedupeKeyUnique(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:ai-alert-heal-model?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AIAlertIngestEvent{}, &AIAlertHealJob{}, &AIAlertHealAttempt{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	first := AIAlertIngestEvent{
		ID:          "evt-1",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: "fp-1",
		Status:      "firing",
		DedupeKey:   "alertmanager:fp-1:firing",
		Title:       "CPU high",
		ReceivedAt:  time.Now().UTC(),
	}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first: %v", err)
	}

	duplicate := first
	duplicate.ID = "evt-2"
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("expected unique dedupe_key violation, got nil")
	}
}
