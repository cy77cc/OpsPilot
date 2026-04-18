package api

import (
	"context"
	"testing"

	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

func TestAlertHealJobEnqueuer_ReusesExistingJobByEventID(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertHealJob{})
	enqueuer := &alertHealJobEnqueuer{svcCtx: &svc.ServiceContext{DB: db}}

	firstID, err := enqueuer.EnqueueBatch(context.Background(), []model.AIAlertIngestEvent{{ID: "evt-1"}})
	if err != nil {
		t.Fatalf("enqueue first batch: %v", err)
	}
	secondID, err := enqueuer.EnqueueBatch(context.Background(), []model.AIAlertIngestEvent{{ID: "evt-1"}})
	if err != nil {
		t.Fatalf("enqueue second batch: %v", err)
	}
	if firstID == "" || secondID == "" {
		t.Fatalf("expected non-empty job ids, first=%q second=%q", firstID, secondID)
	}
	if firstID != secondID {
		t.Fatalf("expected idempotent enqueue to reuse job id, first=%q second=%q", firstID, secondID)
	}

	var count int64
	if err := db.Model(&model.AIAlertHealJob{}).Where("event_id = ?", "evt-1").Count(&count).Error; err != nil {
		t.Fatalf("count jobs by event_id: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one job row for event_id, got %d", count)
	}
}

func TestAlertHealJobEnqueuer_BatchAtomicOnInvalidEventInput(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertHealJob{})
	enqueuer := &alertHealJobEnqueuer{svcCtx: &svc.ServiceContext{DB: db}}

	_, err := enqueuer.EnqueueBatch(context.Background(), []model.AIAlertIngestEvent{
		{ID: "evt-1"},
		{ID: "   "},
	})
	if err == nil {
		t.Fatal("expected batch enqueue error for empty event id, got nil")
	}

	var count int64
	if err := db.Model(&model.AIAlertHealJob{}).Count(&count).Error; err != nil {
		t.Fatalf("count jobs after failed batch: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected atomic rollback with zero rows, got %d", count)
	}
}
