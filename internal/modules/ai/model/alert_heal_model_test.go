package model

import (
	"strings"
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
	err = db.Create(&duplicate).Error
	if err == nil {
		t.Fatal("expected unique dedupe_key violation, got nil")
	}
	lowerErr := strings.ToLower(err.Error())
	if !strings.Contains(lowerErr, "unique") {
		t.Fatalf("expected duplicate-key semantics (unique), got: %v", err)
	}
	if !(strings.Contains(lowerErr, "dedupe_key") || strings.Contains(lowerErr, "duplicate")) {
		t.Fatalf("expected duplicate-key semantics for dedupe_key, got: %v", err)
	}
}

func TestAIAlertHealJob_EventIDUnique(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:ai-alert-heal-job-unique?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&AIAlertHealJob{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	first := AIAlertHealJob{ID: "job-1", EventID: "evt-1", Scene: "alert_self_heal", Status: "pending"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first: %v", err)
	}

	dup := AIAlertHealJob{ID: "job-2", EventID: "evt-1", Scene: "alert_self_heal", Status: "pending"}
	err = db.Create(&dup).Error
	if err == nil {
		t.Fatal("expected unique event_id violation, got nil")
	}
	lowerErr := strings.ToLower(err.Error())
	if !strings.Contains(lowerErr, "unique") {
		t.Fatalf("expected duplicate-key semantics (unique), got: %v", err)
	}
}
