package workers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao"
	aidaoalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/alertheal"
	ailogicalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/logic/alertheal"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

func TestAlertHealWorker_FailureRetriesThenEscalatesToApproval(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{}, &model.AIAlertHealJob{})
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	if err := db.Create(&model.AIAlertIngestEvent{
		ID:          "evt-1",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: "fp-1",
		Status:      "firing",
		DedupeKey:   "alertmanager:fp-1:firing",
		Title:       "CPU high",
		ReceivedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed ingest event: %v", err)
	}
	if err := db.Create(&model.AIAlertHealJob{
		ID:         "job-1",
		EventID:    "evt-1",
		Scene:      "alert_self_heal",
		Status:     "pending",
		RetryCount: 0,
		MaxRetry:   3,
	}).Error; err != nil {
		t.Fatalf("seed heal job: %v", err)
	}

	svc := ailogicalertheal.NewService(aidaoalertheal.NewDAO(db))
	executor := &stubAlertHealExecutor{
		errs: []error{
			errors.New("boom-1"),
			errors.New("boom-2"),
			errors.New("boom-3"),
		},
	}
	worker := NewAlertHealWorker(
		svc,
		executor,
		WithAlertHealWorkerClock(func() time.Time { return now }),
		WithAlertHealWorkerBaseBackoff(time.Second),
		WithAlertHealWorkerMaxBackoff(time.Second),
	)

	for i := 0; i < 3; i++ {
		claimed, err := worker.RunOnce(context.Background())
		if err != nil {
			t.Fatalf("run once #%d: %v", i+1, err)
		}
		if !claimed {
			t.Fatalf("expected claim on run #%d", i+1)
		}
		now = now.Add(2 * time.Second)
	}

	var saved model.AIAlertHealJob
	if err := db.Where("id = ?", "job-1").Take(&saved).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if saved.Status != "waiting_approval" {
		t.Fatalf("expected status waiting_approval, got %q", saved.Status)
	}
	if saved.RetryCount != 3 {
		t.Fatalf("expected retry_count=3, got %d", saved.RetryCount)
	}
	if !strings.Contains(saved.LastError, "boom-3") {
		t.Fatalf("expected last_error to contain boom-3, got %q", saved.LastError)
	}
	if executor.calls != 3 {
		t.Fatalf("expected executor to run 3 times, got %d", executor.calls)
	}
}

func TestAlertHealWorker_ResolvedCancelsActiveJobs(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{}, &model.AIAlertHealJob{})
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	if err := db.Create(&model.AIAlertIngestEvent{
		ID:          "evt-firing",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: "fp-resolved",
		Status:      "firing",
		DedupeKey:   "alertmanager:fp-resolved:firing",
		Title:       "CPU high",
		ReceivedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed firing event: %v", err)
	}
	if err := db.Create(&model.AIAlertIngestEvent{
		ID:          "evt-resolved",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: "fp-resolved",
		Status:      "resolved",
		DedupeKey:   "alertmanager:fp-resolved:resolved",
		Title:       "CPU recovered",
		ReceivedAt:  now.Add(2 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed resolved event: %v", err)
	}
	if err := db.Create(&model.AIAlertHealJob{
		ID:         "job-resolved",
		EventID:    "evt-firing",
		Scene:      "alert_self_heal",
		Status:     "pending",
		RetryCount: 0,
		MaxRetry:   3,
	}).Error; err != nil {
		t.Fatalf("seed heal job: %v", err)
	}

	svc := ailogicalertheal.NewService(aidaoalertheal.NewDAO(db))
	executor := &stubAlertHealExecutor{resultRunID: "run-should-not-happen"}
	worker := NewAlertHealWorker(
		svc,
		executor,
		WithAlertHealWorkerClock(func() time.Time { return now }),
	)

	claimed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !claimed {
		t.Fatal("expected worker to claim resolved job")
	}
	if executor.calls != 0 {
		t.Fatalf("expected executor not to run for resolved job, got %d calls", executor.calls)
	}

	var saved model.AIAlertHealJob
	if err := db.Where("id = ?", "job-resolved").Take(&saved).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if saved.Status != "canceled_resolved" {
		t.Fatalf("expected status canceled_resolved, got %q", saved.Status)
	}
}

func TestAlertHealWorker_ReclaimsStaleAutoFixingJob(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{}, &model.AIAlertHealJob{})
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	if err := db.Create(&model.AIAlertIngestEvent{
		ID:          "evt-stale",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: "fp-stale",
		Status:      "firing",
		DedupeKey:   "alertmanager:fp-stale:firing",
		Title:       "Disk high",
		ReceivedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed ingest event: %v", err)
	}
	if err := db.Create(&model.AIAlertHealJob{
		ID:         "job-stale",
		EventID:    "evt-stale",
		Scene:      "alert_self_heal",
		Status:     "auto_fixing",
		RetryCount: 1,
		MaxRetry:   3,
	}).Error; err != nil {
		t.Fatalf("seed stale job: %v", err)
	}
	staleAt := now.Add(-10 * time.Minute)
	if err := db.Model(&model.AIAlertHealJob{}).Where("id = ?", "job-stale").UpdateColumn("updated_at", staleAt).Error; err != nil {
		t.Fatalf("mark stale updated_at: %v", err)
	}

	svc := ailogicalertheal.NewService(aidaoalertheal.NewDAO(db))
	executor := &stubAlertHealExecutor{resultRunID: "run-stale-1"}
	worker := NewAlertHealWorker(
		svc,
		executor,
		WithAlertHealWorkerClock(func() time.Time { return now }),
	)

	claimed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !claimed {
		t.Fatal("expected stale auto_fixing job to be reclaimed")
	}
	if executor.calls != 1 {
		t.Fatalf("expected executor to run once, got %d", executor.calls)
	}

	var saved model.AIAlertHealJob
	if err := db.Where("id = ?", "job-stale").Take(&saved).Error; err != nil {
		t.Fatalf("reload stale job: %v", err)
	}
	if saved.Status != "succeeded" {
		t.Fatalf("expected status succeeded, got %q", saved.Status)
	}
	if saved.LatestRunID != "run-stale-1" {
		t.Fatalf("expected latest_run_id run-stale-1, got %q", saved.LatestRunID)
	}
}

type stubAlertHealExecutor struct {
	errs        []error
	resultRunID string
	calls       int
}

func (s *stubAlertHealExecutor) Execute(_ context.Context, _ *model.AIAlertHealJob) (*ailogicalertheal.ExecutionResult, error) {
	s.calls++
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		return nil, err
	}
	return &ailogicalertheal.ExecutionResult{RunID: s.resultRunID}, nil
}
