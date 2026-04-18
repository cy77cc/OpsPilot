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

func TestAlertHealJobEnqueuer_ResolvedEventCancelsWaitingApprovalRun(t *testing.T) {
	db := aidao.NewTestDB(t,
		&model.AIAlertIngestEvent{},
		&model.AIAlertHealJob{},
		&model.AIRun{},
		&model.AIApprovalTask{},
	)
	enqueuer := &alertHealJobEnqueuer{svcCtx: &svc.ServiceContext{DB: db}}
	now := time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC)

	firing := model.AIAlertIngestEvent{
		ID:          "evt-firing",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: "fp-approval",
		Status:      "firing",
		DedupeKey:   "alertmanager:fp-approval:firing",
		Title:       "CPU high",
		ReceivedAt:  now,
	}
	resolved := model.AIAlertIngestEvent{
		ID:          "evt-resolved",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: "fp-approval",
		Status:      "resolved",
		DedupeKey:   "alertmanager:fp-approval:resolved",
		Title:       "CPU recovered",
		ReceivedAt:  now.Add(time.Minute),
	}
	if err := db.Create(&firing).Error; err != nil {
		t.Fatalf("seed firing event: %v", err)
	}
	if err := db.Create(&resolved).Error; err != nil {
		t.Fatalf("seed resolved event: %v", err)
	}
	if err := db.Create(&model.AIRun{
		ID:                 "run-1",
		SessionID:          "sess-1",
		ClientRequestID:    "req-1",
		UserMessageID:      "msg-user-1",
		AssistantMessageID: "msg-assistant-1",
		Status:             "waiting_approval",
		TraceJSON:          `{}`,
	}).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
	if err := db.Create(&model.AIApprovalTask{
		ApprovalID:     "approval-1",
		CheckpointID:   "checkpoint-1",
		SessionID:      "sess-1",
		RunID:          "run-1",
		UserID:         0,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-1",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "pending",
		TimeoutSeconds: 300,
	}).Error; err != nil {
		t.Fatalf("seed approval task: %v", err)
	}
	if err := db.Create(&model.AIAlertHealJob{
		ID:          "job-1",
		EventID:     firing.ID,
		Scene:       "alert_self_heal",
		Status:      "waiting_approval",
		LatestRunID: "run-1",
	}).Error; err != nil {
		t.Fatalf("seed alert-heal job: %v", err)
	}

	jobID, err := enqueuer.EnqueueBatch(context.Background(), []model.AIAlertIngestEvent{resolved})
	if err != nil {
		t.Fatalf("enqueue resolved event: %v", err)
	}
	if jobID == "" {
		t.Fatal("expected resolved event to return a job id")
	}

	var savedJob model.AIAlertHealJob
	if err := db.Where("id = ?", "job-1").Take(&savedJob).Error; err != nil {
		t.Fatalf("reload alert-heal job: %v", err)
	}
	if savedJob.Status != "canceled_resolved" {
		t.Fatalf("expected alert-heal job canceled_resolved, got %q", savedJob.Status)
	}

	var savedRun model.AIRun
	if err := db.Where("id = ?", "run-1").Take(&savedRun).Error; err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if savedRun.Status != "cancelled" {
		t.Fatalf("expected run cancelled, got %q", savedRun.Status)
	}

	var savedApproval model.AIApprovalTask
	if err := db.Where("approval_id = ?", "approval-1").Take(&savedApproval).Error; err != nil {
		t.Fatalf("reload approval task: %v", err)
	}
	if savedApproval.Status != "expired" {
		t.Fatalf("expected approval task expired, got %q", savedApproval.Status)
	}
}
