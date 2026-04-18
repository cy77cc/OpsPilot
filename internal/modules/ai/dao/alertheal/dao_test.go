package alertheal

import (
	"context"
	"errors"
	"testing"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"gorm.io/gorm"
)

func TestDAO_UpsertIngestEvent_DeduplicatesByDedupeKey(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{})
	dao := NewDAO(db)
	ctx := context.Background()

	first := &model.AIAlertIngestEvent{
		ID:             "evt-1",
		Source:         "alertmanager",
		Protocol:       "alertmanager",
		Fingerprint:    "fp-1",
		Status:         "firing",
		DedupeKey:      "alertmanager:fp-1:firing",
		Severity:       "warning",
		Title:          "CPU",
		RawPayloadJSON: `{"alerts":[]}`,
		ReceivedAt:     time.Now().UTC(),
	}
	saved1, err := dao.UpsertIngestEvent(ctx, first)
	if err != nil {
		t.Fatalf("upsert first: %v", err)
	}

	second := &model.AIAlertIngestEvent{
		ID:             "evt-2",
		Source:         "alertmanager",
		Protocol:       "alertmanager",
		Fingerprint:    "fp-1",
		Status:         "firing",
		DedupeKey:      "alertmanager:fp-1:firing",
		Severity:       "warning",
		Title:          "CPU changed",
		RawPayloadJSON: `{"alerts":[{"fingerprint":"fp-1"}]}`,
		ReceivedAt:     time.Now().UTC(),
	}
	saved2, err := dao.UpsertIngestEvent(ctx, second)
	if err != nil {
		t.Fatalf("upsert duplicate: %v", err)
	}

	var count int64
	if err := db.Model(&model.AIAlertIngestEvent{}).Where("dedupe_key = ?", first.DedupeKey).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected single deduped row, got %d", count)
	}
	if saved1.ID == "" || saved2.ID == "" {
		t.Fatalf("expected persisted ids, got saved1=%q saved2=%q", saved1.ID, saved2.ID)
	}
	if saved1.ID != saved2.ID {
		t.Fatalf("expected duplicate upsert to return original row id, got %q and %q", saved1.ID, saved2.ID)
	}
}

func TestDAO_UpsertIngestEvents_RollsBackOnDBError(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{})
	dao := NewDAO(db)
	ctx := context.Background()

	rows := []*model.AIAlertIngestEvent{
		{
			ID:             "dup-id",
			Source:         "alertmanager",
			Protocol:       "alertmanager",
			Fingerprint:    "fp-1",
			Status:         "firing",
			DedupeKey:      "alertmanager:fp-1:firing",
			Severity:       "warning",
			Title:          "CPU",
			RawPayloadJSON: `{"alerts":[]}`,
			ReceivedAt:     time.Now().UTC(),
		},
		{
			ID:             "dup-id",
			Source:         "alertmanager",
			Protocol:       "alertmanager",
			Fingerprint:    "fp-2",
			Status:         "firing",
			DedupeKey:      "alertmanager:fp-2:firing",
			Severity:       "warning",
			Title:          "MEM",
			RawPayloadJSON: `{"alerts":[]}`,
			ReceivedAt:     time.Now().UTC(),
		},
	}

	if _, err := dao.UpsertIngestEvents(ctx, rows); err == nil {
		t.Fatal("expected db error, got nil")
	}

	var count int64
	if err := db.Model(&model.AIAlertIngestEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected zero rows after transactional rollback, got %d", count)
	}
}

func TestDAO_RenewAutoFixingLease_TouchesUpdatedAt(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertHealJob{})
	dao := NewDAO(db)
	ctx := context.Background()

	if err := db.Create(&model.AIAlertHealJob{
		ID:      "job-lease",
		EventID: "evt-lease",
		Scene:   "alert_self_heal",
		Status:  "auto_fixing",
	}).Error; err != nil {
		t.Fatalf("seed auto_fixing job: %v", err)
	}
	before := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	if err := db.Model(&model.AIAlertHealJob{}).Where("id = ?", "job-lease").UpdateColumn("updated_at", before).Error; err != nil {
		t.Fatalf("set initial updated_at: %v", err)
	}

	renewAt := before.Add(90 * time.Second)
	if err := dao.RenewAutoFixingLease(ctx, "job-lease", renewAt); err != nil {
		t.Fatalf("renew lease: %v", err)
	}

	var saved model.AIAlertHealJob
	if err := db.Where("id = ?", "job-lease").Take(&saved).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if !saved.UpdatedAt.Equal(renewAt) {
		t.Fatalf("expected updated_at=%s, got %s", renewAt.UTC(), saved.UpdatedAt.UTC())
	}
}

func TestDAO_RenewAutoFixingLease_RequiresAutoFixingState(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertHealJob{})
	dao := NewDAO(db)
	ctx := context.Background()

	if err := db.Create(&model.AIAlertHealJob{
		ID:      "job-pending",
		EventID: "evt-pending",
		Scene:   "alert_self_heal",
		Status:  "pending",
	}).Error; err != nil {
		t.Fatalf("seed pending job: %v", err)
	}

	err := dao.RenewAutoFixingLease(ctx, "job-pending", time.Date(2026, 4, 18, 10, 5, 0, 0, time.UTC))
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected record not found for non-auto-fixing state, got %v", err)
	}
}
