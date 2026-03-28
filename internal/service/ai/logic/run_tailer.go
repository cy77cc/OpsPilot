package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
)

const (
	defaultTailPollInterval = 250 * time.Millisecond
	defaultTailIdleTimeout  = 30 * time.Second
	defaultTailMaxDuration  = 5 * time.Minute
)

type TailOptions struct {
	PollInterval    time.Duration
	IdleTimeout     time.Duration
	MaxTailDuration time.Duration
}

func (o TailOptions) withDefaults() TailOptions {
	if o.PollInterval <= 0 {
		o.PollInterval = defaultTailPollInterval
	}
	if o.IdleTimeout <= 0 {
		o.IdleTimeout = defaultTailIdleTimeout
	}
	if o.MaxTailDuration <= 0 {
		o.MaxTailDuration = defaultTailMaxDuration
	}
	return o
}

type RunTailer struct {
	RunDAO      *aidao.AIRunDAO
	RunEventDAO *aidao.AIRunEventDAO
}

func (t *RunTailer) ReplayThenTail(ctx context.Context, runID, lastEventID string, emit EventEmitter, options TailOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil
	}
	if t == nil || t.RunEventDAO == nil || strings.TrimSpace(runID) == "" {
		return nil
	}

	options = options.withDefaults()
	startedAt := time.Now()
	lastActivityAt := startedAt
	cursor := strings.TrimSpace(lastEventID)
	emitted := false

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		events, err := t.RunEventDAO.ListAfterEventID(ctx, runID, cursor)
		if err != nil {
			if isTailShutdown(err) {
				return nil
			}
			return err
		}
		if len(events) > 0 {
			for _, event := range events {
				payload, err := decodeRunEventPayload(event.PayloadJSON)
				if err != nil {
					return err
				}
				emit(event.EventType, withEventID(payload, event.ID))
				cursor = event.ID
				emitted = true
			}
			lastActivityAt = time.Now()
		}

		run, err := t.loadRun(ctx, runID)
		if err != nil {
			if isTailShutdown(err) {
				return nil
			}
			return err
		}
		if run == nil {
			return nil
		}
		if isTailTerminalStatus(run.Status) {
			if !emitted {
				emit("run_state", map[string]any{
					"run_id": runID,
					"status": strings.TrimSpace(run.Status),
					"agent":  "executor",
				})
			}
			return nil
		}
		if !isTailOpenStatus(run.Status) {
			return nil
		}

		now := time.Now()
		if options.MaxTailDuration > 0 && now.Sub(startedAt) >= options.MaxTailDuration {
			return nil
		}
		if options.IdleTimeout > 0 && now.Sub(lastActivityAt) >= options.IdleTimeout {
			return nil
		}

		waitFor := options.PollInterval
		if remaining := time.Until(startedAt.Add(options.MaxTailDuration)); options.MaxTailDuration > 0 && remaining < waitFor {
			waitFor = remaining
		}
		if remaining := time.Until(lastActivityAt.Add(options.IdleTimeout)); options.IdleTimeout > 0 && remaining < waitFor {
			waitFor = remaining
		}
		if waitFor <= 0 {
			waitFor = options.PollInterval
		}

		timer := time.NewTimer(waitFor)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (t *RunTailer) loadRun(ctx context.Context, runID string) (*aidaoRunLike, error) {
	if t == nil || t.RunDAO == nil {
		return nil, nil
	}
	run, err := t.RunDAO.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, nil
	}
	return &aidaoRunLike{Status: run.Status}, nil
}

type aidaoRunLike struct {
	Status string
}

func isTailOpenStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "waiting_approval", "resuming", "running", "resume_failed_retryable":
		return true
	default:
		return false
	}
}

func isTailTerminalStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "completed", "completed_with_tool_errors", "failed", "failed_runtime", "cancelled", "expired":
		return true
	default:
		return false
	}
}

func isTailShutdown(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
