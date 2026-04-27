// Package stream 实现运行事件尾部轮询逻辑。
package stream

import (
	"context"
	"strings"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
)

const (
	defaultTailPollInterval = 250 * time.Millisecond
	defaultTailIdleTimeout  = 30 * time.Second
	defaultTailMaxDuration  = 5 * time.Minute
)

// TailOptions 定义尾部轮询选项。
type TailOptions struct {
	PollInterval    time.Duration
	IdleTimeout     time.Duration
	MaxTailDuration time.Duration
}

func (o TailOptions) WithDefaults() TailOptions {
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

// RunTailer 实现运行事件尾部轮询。
type RunTailer struct {
	RunDAO      *aidao.AIRunDAO
	RunEventDAO *aidao.AIRunEventDAO
}

// ReplayThenTail 从重放最后事件开始，然后尾部轮询新事件。
func (t *RunTailer) ReplayThenTail(ctx context.Context, runID, lastEventID string, emit func(event string, data any), options TailOptions) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if t == nil || t.RunEventDAO == nil || strings.TrimSpace(runID) == "" {
		return nil
	}
	options = options.WithDefaults()
	startedAt := time.Now()
	lastActivityAt := startedAt
	cursor := strings.TrimSpace(lastEventID)
	emitted := false

	for {
		if err := ctx.Err(); err != nil {
			return err
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
				payload, err := DecodeRunEventPayload(event.PayloadJSON)
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
		if IsTailTerminalStatus(run.Status) {
			if !emitted {
				emit("run_state", map[string]any{"run_id": runID, "status": strings.TrimSpace(run.Status), "agent": "executor"})
			}
			return nil
		}
		if !IsTailOpenStatus(run.Status) {
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
			return ctx.Err()
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

type aidaoRunLike struct{ Status string }

// IsTailOpenStatus 判断运行状态是否处于开放状态。
func IsTailOpenStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "waiting_approval", "resuming", "running", "resume_failed_retryable":
		return true
	default:
		return false
	}
}

// IsTailTerminalStatus 判断运行状态是否为终态。
func IsTailTerminalStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "completed", "completed_with_tool_errors", "failed", "failed_runtime", "cancelled", "expired":
		return true
	default:
		return false
	}
}

func isTailShutdown(err error) bool {
	return err == context.Canceled || err == context.DeadlineExceeded
}

func withEventID(payload any, eventID string) any {
	if strings.TrimSpace(eventID) == "" {
		return payload
	}
	data, ok := payload.(map[string]any)
	if !ok {
		return payload
	}
	copyPayload := make(map[string]any, len(data)+1)
	for key, value := range data {
		copyPayload[key] = value
	}
	copyPayload["event_id"] = eventID
	return copyPayload
}
