package workers

import (
	"context"
	"errors"
	"strings"
	"sync"
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
	executor := newBlockingAlertHealExecutor("run-should-not-happen")
	worker := NewAlertHealWorker(
		svc,
		executor,
		WithAlertHealWorkerClock(func() time.Time { return now }),
	)

	runDone := make(chan struct{})
	var claimed bool
	var err error
	go func() {
		defer close(runDone)
		claimed, err = worker.RunOnce(context.Background())
	}()

	select {
	case <-executor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for execution to start")
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

	close(executor.release)
	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for execution to finish")
	}

	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !claimed {
		t.Fatal("expected worker to claim resolved job")
	}
	if executor.calls != 1 {
		t.Fatalf("expected executor to run once before post-run cancel, got %d calls", executor.calls)
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

func TestAlertHealWorker_ExecutorWaitingApprovalDoesNotConsumeRetryBudget(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{}, &model.AIAlertHealJob{})
	now := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	if err := db.Create(&model.AIAlertIngestEvent{
		ID:          "evt-approval",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: "fp-approval",
		Status:      "firing",
		DedupeKey:   "alertmanager:fp-approval:firing",
		Title:       "Needs approval",
		ReceivedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed ingest event: %v", err)
	}
	if err := db.Create(&model.AIAlertHealJob{
		ID:         "job-approval",
		EventID:    "evt-approval",
		Scene:      "alert_self_heal",
		Status:     "pending",
		RetryCount: 1,
		MaxRetry:   3,
	}).Error; err != nil {
		t.Fatalf("seed heal job: %v", err)
	}

	svc := ailogicalertheal.NewService(aidaoalertheal.NewDAO(db))
	executor := &stubAlertHealExecutor{
		results: []*ailogicalertheal.ExecutionResult{{
			RunID:           "run-approval-1",
			RunStatus:       "waiting_approval",
			WaitingApproval: true,
		}},
	}
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
		t.Fatal("expected worker to claim pending job")
	}

	var saved model.AIAlertHealJob
	if err := db.Where("id = ?", "job-approval").Take(&saved).Error; err != nil {
		t.Fatalf("reload job: %v", err)
	}
	if saved.Status != "waiting_approval" {
		t.Fatalf("expected status waiting_approval, got %q", saved.Status)
	}
	if saved.RetryCount != 1 {
		t.Fatalf("expected retry_count unchanged at 1, got %d", saved.RetryCount)
	}
	if saved.LatestRunID != "run-approval-1" {
		t.Fatalf("expected latest_run_id run-approval-1, got %q", saved.LatestRunID)
	}
}

func TestAlertHealWorker_HeartbeatPreventsStaleReclaimDuringLongExecution(t *testing.T) {
	db := aidao.NewTestDB(t, &model.AIAlertIngestEvent{}, &model.AIAlertHealJob{})
	if err := db.Exec("PRAGMA busy_timeout = 2000").Error; err != nil {
		t.Fatalf("set busy timeout: %v", err)
	}
	base := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	clock := newTestAlertHealClock(base)

	if err := db.Create(&model.AIAlertIngestEvent{
		ID:          "evt-heartbeat",
		Source:      "alertmanager",
		Protocol:    "alertmanager",
		Fingerprint: "fp-heartbeat",
		Status:      "firing",
		DedupeKey:   "alertmanager:fp-heartbeat:firing",
		Title:       "Long running",
		ReceivedAt:  base,
	}).Error; err != nil {
		t.Fatalf("seed ingest event: %v", err)
	}
	if err := db.Create(&model.AIAlertHealJob{
		ID:         "job-heartbeat",
		EventID:    "evt-heartbeat",
		Scene:      "alert_self_heal",
		Status:     "pending",
		RetryCount: 0,
		MaxRetry:   3,
	}).Error; err != nil {
		t.Fatalf("seed heal job: %v", err)
	}

	svc := ailogicalertheal.NewService(aidaoalertheal.NewDAO(db))
	blockingExecutor := newBlockingAlertHealExecutor("run-heartbeat-1")
	worker := NewAlertHealWorker(
		svc,
		blockingExecutor,
		WithAlertHealWorkerClock(clock.Now),
		WithAlertHealWorkerLeaseHeartbeat(25*time.Millisecond),
	)

	runDone := make(chan struct{})
	var runClaimed bool
	var runErr error
	go func() {
		defer close(runDone)
		runClaimed, runErr = worker.RunOnce(context.Background())
	}()

	select {
	case <-blockingExecutor.started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for long-running execution to start")
	}

	clock.Set(base.Add(2*time.Minute + 10*time.Second))
	time.Sleep(100 * time.Millisecond)

	reclaimNow := base.Add(3 * time.Minute)
	reclaimExecutor := &stubAlertHealExecutor{resultRunID: "run-should-not-claim"}
	reclaimWorker := NewAlertHealWorker(
		svc,
		reclaimExecutor,
		WithAlertHealWorkerClock(func() time.Time { return reclaimNow }),
	)
	reclaimed, err := reclaimWorker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("reclaim worker run once: %v", err)
	}
	if reclaimed {
		t.Fatal("expected heartbeat to prevent stale reclaim while execution is in progress")
	}
	if reclaimExecutor.calls != 0 {
		t.Fatalf("expected reclaim executor to stay idle, got %d calls", reclaimExecutor.calls)
	}

	close(blockingExecutor.release)
	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for long-running execution to finish")
	}
	if runErr != nil {
		t.Fatalf("run once with heartbeat: %v", runErr)
	}
	if !runClaimed {
		t.Fatal("expected first worker to claim and process job")
	}

	var saved model.AIAlertHealJob
	if err := db.Where("id = ?", "job-heartbeat").Take(&saved).Error; err != nil {
		t.Fatalf("reload heartbeat job: %v", err)
	}
	if saved.Status != "succeeded" {
		t.Fatalf("expected status succeeded, got %q", saved.Status)
	}
}

type stubAlertHealExecutor struct {
	errs        []error
	resultRunID string
	results     []*ailogicalertheal.ExecutionResult
	calls       int
}

func (s *stubAlertHealExecutor) Execute(_ context.Context, _ *model.AIAlertHealJob) (*ailogicalertheal.ExecutionResult, error) {
	s.calls++
	if len(s.results) > 0 {
		result := s.results[0]
		s.results = s.results[1:]
		return result, nil
	}
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		return nil, err
	}
	return &ailogicalertheal.ExecutionResult{RunID: s.resultRunID}, nil
}

type blockingAlertHealExecutor struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
	runID   string
	calls   int
}

func newBlockingAlertHealExecutor(runID string) *blockingAlertHealExecutor {
	return &blockingAlertHealExecutor{
		started: make(chan struct{}),
		release: make(chan struct{}),
		runID:   runID,
	}
}

func (e *blockingAlertHealExecutor) Execute(ctx context.Context, _ *model.AIAlertHealJob) (*ailogicalertheal.ExecutionResult, error) {
	e.calls++
	e.once.Do(func() { close(e.started) })
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-e.release:
		return &ailogicalertheal.ExecutionResult{RunID: e.runID}, nil
	}
}

type testAlertHealClock struct {
	mu  sync.RWMutex
	now time.Time
}

func newTestAlertHealClock(initial time.Time) *testAlertHealClock {
	return &testAlertHealClock{now: initial.UTC()}
}

func (c *testAlertHealClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *testAlertHealClock) Set(next time.Time) {
	c.mu.Lock()
	c.now = next.UTC()
	c.mu.Unlock()
}
