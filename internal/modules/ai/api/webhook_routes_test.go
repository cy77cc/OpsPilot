package api

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestAlertHealJobEnqueuer_ConcurrentDuplicateEnqueueKeepsSingleRow(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertHealJob{})
	enqueuer := &alertHealJobEnqueuer{svcCtx: &svc.ServiceContext{DB: db}}

	const workers = 8
	results := make(chan string, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			var lastErr error
			for attempt := 0; attempt < 5; attempt++ {
				jobID, err := enqueuer.EnqueueBatch(context.Background(), []model.AIAlertIngestEvent{{ID: "evt-concurrent"}})
				if err == nil {
					results <- jobID
					return
				}
				lastErr = err
				lowerErr := strings.ToLower(err.Error())
				if strings.Contains(lowerErr, "locked") || strings.Contains(lowerErr, "busy") {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				errs <- err
				return
			}
			errs <- lastErr
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent enqueue failed: %v", err)
		}
	}

	firstID := ""
	for id := range results {
		if id == "" {
			t.Fatal("expected non-empty job id")
		}
		if firstID == "" {
			firstID = id
			continue
		}
		if id != firstID {
			t.Fatalf("expected same job id across concurrent enqueues, got first=%q another=%q", firstID, id)
		}
	}
	if firstID == "" {
		t.Fatal("expected at least one successful enqueue")
	}

	var count int64
	if err := db.Model(&model.AIAlertHealJob{}).Where("event_id = ?", "evt-concurrent").Count(&count).Error; err != nil {
		t.Fatalf("count jobs by event_id: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly one job row for concurrent duplicate event, got %d", count)
	}
}
