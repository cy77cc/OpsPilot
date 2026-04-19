package logic

import (
	"context"
	"testing"
	"time"

	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	monitoringmodel "github.com/cy77cc/OpsPilot/internal/modules/monitoring/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListAlerts_EnrichesLatestAlertHealSummary(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:list-alerts-heal-summary?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&monitoringmodel.AlertEvent{},
		&aimodel.AIAlertIngestEvent{},
		&aimodel.AIAlertHealJob{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	now := time.Date(2026, 4, 18, 15, 0, 0, 0, time.UTC)
	alert := monitoringmodel.AlertEvent{
		ID:          42,
		Title:       "CPU high",
		Message:     "CPU high",
		Severity:    "critical",
		Source:      "alertmanager/fp-1",
		Status:      "firing",
		TriggeredAt: now,
	}
	if err := db.Create(&alert).Error; err != nil {
		t.Fatalf("seed alert: %v", err)
	}
	other := monitoringmodel.AlertEvent{
		ID:          43,
		Title:       "manual",
		Message:     "manual",
		Severity:    "warning",
		Source:      "manual/ops",
		Status:      "firing",
		TriggeredAt: now,
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("seed other alert: %v", err)
	}

	event1 := aimodel.AIAlertIngestEvent{
		ID:          "evt-1",
		Source:      "receiver",
		Protocol:    "alertmanager",
		Fingerprint: "fp-1",
		Status:      "firing",
		DedupeKey:   "receiver:fp-1:firing",
		Title:       "CPU high",
		ReceivedAt:  now,
	}
	event2 := aimodel.AIAlertIngestEvent{
		ID:          "evt-2",
		Source:      "receiver",
		Protocol:    "alertmanager",
		Fingerprint: "fp-1",
		Status:      "firing",
		DedupeKey:   "receiver:fp-1:firing:retry",
		Title:       "CPU high retry",
		ReceivedAt:  now.Add(time.Minute),
	}
	if err := db.Create(&event1).Error; err != nil {
		t.Fatalf("seed event1: %v", err)
	}
	if err := db.Create(&event2).Error; err != nil {
		t.Fatalf("seed event2: %v", err)
	}
	job1Updated := now.Add(2 * time.Minute)
	job2Updated := now.Add(5 * time.Minute)
	if err := db.Create(&aimodel.AIAlertHealJob{
		ID:          "job-1",
		EventID:     event1.ID,
		Scene:       "alert_self_heal",
		Status:      "failed_manual",
		UpdatedAt:   job1Updated,
		LatestRunID: "run-1",
	}).Error; err != nil {
		t.Fatalf("seed job1: %v", err)
	}
	if err := db.Create(&aimodel.AIAlertHealJob{
		ID:          "job-2",
		EventID:     event2.ID,
		Scene:       "alert_self_heal",
		Status:      "waiting_approval",
		UpdatedAt:   job2Updated,
		LatestRunID: "run-2",
	}).Error; err != nil {
		t.Fatalf("seed job2: %v", err)
	}

	logic := NewLogic(&svc.ServiceContext{DB: db})
	rows, total, err := logic.ListAlerts(context.Background(), "", "", 0, 1, 20)
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}

	var matched *monitoringmodel.AlertEvent
	for i := range rows {
		if rows[i].ID == 42 {
			matched = &rows[i]
			break
		}
	}
	if matched == nil {
		t.Fatal("expected alert 42 in result set")
	}
	if matched.LatestHealJobID != "job-2" {
		t.Fatalf("expected latest heal job id job-2, got %q", matched.LatestHealJobID)
	}
	if matched.LatestHealStatus != "waiting_approval" {
		t.Fatalf("expected latest heal status waiting_approval, got %q", matched.LatestHealStatus)
	}
	if matched.LatestHealRunID != "run-2" {
		t.Fatalf("expected latest heal run id run-2, got %q", matched.LatestHealRunID)
	}
	if matched.LatestHealUpdatedAt == nil || !matched.LatestHealUpdatedAt.Equal(job2Updated) {
		t.Fatalf("expected latest heal updated_at %s, got %+v", job2Updated, matched.LatestHealUpdatedAt)
	}

	for i := range rows {
		if rows[i].ID == 43 && rows[i].LatestHealJobID != "" {
			t.Fatalf("expected unrelated alert to have no heal summary, got %q", rows[i].LatestHealJobID)
		}
	}
}

func TestListAlerts_HealSummaryDoesNotCrossProtocol(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:summary-protocol?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&monitoringmodel.AlertEvent{},
		&aimodel.AIAlertIngestEvent{},
		&aimodel.AIAlertHealJob{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	now := time.Now().UTC()
	if err := db.Create(&monitoringmodel.AlertEvent{ID: 42, Source: "alertmanager/fp-1", Severity: "critical", Status: "firing", TriggeredAt: now}).Error; err != nil {
		t.Fatalf("seed alert event: %v", err)
	}
	if err := db.Create(&aimodel.AIAlertIngestEvent{
		ID:          "evt-am",
		Protocol:    "alertmanager",
		Source:      "alertmanager",
		Fingerprint: "fp-1",
		Status:      "firing",
		DedupeKey:   "alertmanager:fp-1:firing",
		Title:       "am",
		ReceivedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed alertmanager ingest event: %v", err)
	}
	if err := db.Create(&aimodel.AIAlertIngestEvent{
		ID:          "evt-u",
		Protocol:    "opspilot.alert.v1",
		Source:      "opspilot.alert.v1",
		Fingerprint: "fp-1",
		Status:      "firing",
		DedupeKey:   "opspilot.alert.v1:fp-1:firing",
		Title:       "u",
		ReceivedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed unified ingest event: %v", err)
	}
	if err := db.Create(&aimodel.AIAlertHealJob{
		ID:        "job-am",
		EventID:   "evt-am",
		Scene:     "alert_self_heal",
		Status:    "succeeded",
		UpdatedAt: now.Add(time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed alertmanager job: %v", err)
	}
	if err := db.Create(&aimodel.AIAlertHealJob{
		ID:        "job-u",
		EventID:   "evt-u",
		Scene:     "alert_self_heal",
		Status:    "failed_manual",
		UpdatedAt: now.Add(2 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed unified job: %v", err)
	}

	rows, _, err := NewLogic(&svc.ServiceContext{DB: db}).ListAlerts(context.Background(), "", "", 42, 1, 20)
	if err != nil {
		t.Fatalf("list alerts: %v", err)
	}
	if len(rows) != 1 || rows[0].LatestHealJobID != "job-am" {
		t.Fatalf("expected only alertmanager heal summary, got %#v", rows)
	}
}
