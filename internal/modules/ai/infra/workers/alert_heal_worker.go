package workers

import (
	"context"
	"strings"
	"sync"
	"time"

	ailogicalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/logic/alertheal"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

const (
	defaultAlertHealBaseBackoff = 5 * time.Second
	defaultAlertHealMaxBackoff  = time.Minute
	defaultAlertHealLeaseBeat   = 30 * time.Second
)

type alertHealService interface {
	ClaimRunnableJob(ctx context.Context, now time.Time) (*model.AIAlertHealJob, error)
	CancelIfResolved(ctx context.Context, job *model.AIAlertHealJob) (bool, error)
	MarkSucceeded(ctx context.Context, jobID, runID string) error
	MarkWaitingApproval(ctx context.Context, jobID, runID, lastError string, consumeRetry bool) error
	MarkRetryWait(ctx context.Context, jobID, lastError string, nextRetryAt time.Time) error
	RenewAutoFixingLease(ctx context.Context, jobID string, now time.Time) error
}

type AlertHealWorker struct {
	svc         alertHealService
	executor    ailogicalertheal.Executor
	now         func() time.Time
	baseBackoff time.Duration
	maxBackoff  time.Duration
	leaseBeat   time.Duration
}

type AlertHealWorkerOption func(*AlertHealWorker)

func WithAlertHealWorkerClock(now func() time.Time) AlertHealWorkerOption {
	return func(w *AlertHealWorker) {
		if now != nil {
			w.now = now
		}
	}
}

func WithAlertHealWorkerBaseBackoff(d time.Duration) AlertHealWorkerOption {
	return func(w *AlertHealWorker) {
		if d > 0 {
			w.baseBackoff = d
		}
	}
}

func WithAlertHealWorkerMaxBackoff(d time.Duration) AlertHealWorkerOption {
	return func(w *AlertHealWorker) {
		if d > 0 {
			w.maxBackoff = d
		}
	}
}

func WithAlertHealWorkerLeaseHeartbeat(d time.Duration) AlertHealWorkerOption {
	return func(w *AlertHealWorker) {
		if d > 0 {
			w.leaseBeat = d
		}
	}
}

func NewAlertHealWorker(svc alertHealService, executor ailogicalertheal.Executor, opts ...AlertHealWorkerOption) *AlertHealWorker {
	worker := &AlertHealWorker{
		svc:         svc,
		executor:    executor,
		now:         time.Now,
		baseBackoff: defaultAlertHealBaseBackoff,
		maxBackoff:  defaultAlertHealMaxBackoff,
		leaseBeat:   defaultAlertHealLeaseBeat,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(worker)
		}
	}
	return worker
}

func (w *AlertHealWorker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil || w.svc == nil || w.executor == nil {
		return false, nil
	}

	now := w.now().UTC()
	job, err := w.svc.ClaimRunnableJob(ctx, now)
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}

	canceled, err := w.svc.CancelIfResolved(ctx, job)
	if err != nil {
		return true, err
	}
	if canceled {
		return true, nil
	}

	result, execErr := w.executeWithLeaseHeartbeat(ctx, job)
	if execErr == nil {
		runID := ""
		if result != nil {
			runID = strings.TrimSpace(result.RunID)
		}
		if result != nil && result.WaitingApproval {
			return true, w.svc.MarkWaitingApproval(ctx, job.ID, runID, "", false)
		}
		return true, w.svc.MarkSucceeded(ctx, job.ID, runID)
	}

	lastError := strings.TrimSpace(execErr.Error())
	maxRetry := job.MaxRetry
	if maxRetry <= 0 {
		maxRetry = 1
	}
	if job.RetryCount+1 >= maxRetry {
		return true, w.svc.MarkWaitingApproval(ctx, job.ID, "", lastError, true)
	}

	nextRetryAt := now.Add(w.retryBackoff(job.RetryCount))
	return true, w.svc.MarkRetryWait(ctx, job.ID, lastError, nextRetryAt)
}

func (w *AlertHealWorker) executeWithLeaseHeartbeat(ctx context.Context, job *model.AIAlertHealJob) (*ailogicalertheal.ExecutionResult, error) {
	if w.leaseBeat <= 0 || w.svc == nil || job == nil || strings.TrimSpace(job.ID) == "" {
		return w.executor.Execute(ctx, job)
	}

	heartbeatCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(w.leaseBeat)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				_ = w.svc.RenewAutoFixingLease(heartbeatCtx, job.ID, w.now().UTC())
			}
		}
	}()

	result, execErr := w.executor.Execute(ctx, job)
	cancel()
	wg.Wait()
	return result, execErr
}

func (w *AlertHealWorker) retryBackoff(retryCount int) time.Duration {
	if retryCount < 0 {
		retryCount = 0
	}
	delay := w.baseBackoff
	if delay <= 0 {
		delay = defaultAlertHealBaseBackoff
	}
	for i := 0; i < retryCount; i++ {
		delay *= 2
		if delay >= w.maxBackoff {
			return w.maxBackoff
		}
	}
	if w.maxBackoff > 0 && delay > w.maxBackoff {
		return w.maxBackoff
	}
	return delay
}
