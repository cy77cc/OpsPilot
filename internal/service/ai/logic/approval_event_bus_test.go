package logic

import (
	"context"
	"errors"
	"fmt"
	"time"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
)

func TestApprovalEventBus_PreservesPerAggregateSequence(t *testing.T) {
	t.Parallel()

	bus := NewApprovalEventBus()
	var seen []string
	if err := bus.Subscribe("ai.run.*", func(ctx context.Context, event ApprovalEventEnvelope) error {
		seen = append(seen, fmt.Sprintf("%s:%d", event.AggregateID, event.Sequence))
		return nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	events := []ApprovalEventEnvelope{
		testApprovalEnvelope("evt-1", "ai.run.resuming", "run-1", 1),
		testApprovalEnvelope("evt-2", "ai.run.resuming", "run-2", 1),
		testApprovalEnvelope("evt-3", "ai.run.resumed", "run-1", 2),
	}
	for _, event := range events {
		if err := bus.Publish(context.Background(), event); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	if len(seen) != len(events) {
		t.Fatalf("expected %d deliveries, got %d", len(events), len(seen))
	}
	if seen[0] != "run-1:1" || seen[1] != "run-2:1" || seen[2] != "run-1:2" {
		t.Fatalf("unexpected delivery order: %#v", seen)
	}
}

func TestApprovalEventDispatcher_MarkDoneOnSuccess(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-dispatch",
		userID:             42,
		runID:              "run-dispatch",
		userMessageID:      "msg-dispatch-user",
		assistantMessageID: "msg-dispatch-assistant",
		runStatus:          "resuming",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-dispatch",
		CheckpointID:   "checkpoint-dispatch",
		SessionID:      "session-dispatch",
		RunID:          "run-dispatch",
		UserID:         42,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-dispatch",
		ArgumentsJSON:  `{"cmd":"whoami"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     42,
		TimeoutSeconds: 300,
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID: "approval-dispatch",
		EventType:  RunEventTypeResuming,
		RunID:      "run-dispatch",
		SessionID:  "session-dispatch",
		PayloadJSON: `{"status":"resuming"}`,
		Status:     "pending",
	})

	bus := NewApprovalEventBus()
	var delivered []string
	if err := bus.Subscribe("ai.run.*", func(ctx context.Context, event ApprovalEventEnvelope) error {
		delivered = append(delivered, event.EventType)
		return nil
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	dispatcher := NewApprovalEventDispatcher(db, bus)
	claimed, err := dispatcher.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run dispatcher: %v", err)
	}
	if !claimed {
		t.Fatal("expected dispatcher to claim an event")
	}
	if len(delivered) != 1 || delivered[0] != RunEventTypeResuming {
		t.Fatalf("expected run.resuming delivery, got %#v", delivered)
	}

	var outbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-dispatch", RunEventTypeResuming).First(&outbox).Error; err != nil {
		t.Fatalf("reload outbox: %v", err)
	}
	if outbox.Status != "done" {
		t.Fatalf("expected done status, got %q", outbox.Status)
	}
}

func TestApprovalEventDispatcher_MarkRetryOnFailure(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-retry",
		userID:             42,
		runID:              "run-retry",
		userMessageID:      "msg-retry-user",
		assistantMessageID: "msg-retry-assistant",
		runStatus:          "resuming",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-retry",
		CheckpointID:   "checkpoint-retry",
		SessionID:      "session-retry",
		RunID:          "run-retry",
		UserID:         42,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-retry",
		ArgumentsJSON:  `{"cmd":"false"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     42,
		TimeoutSeconds: 300,
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID: "approval-retry",
		EventType:  RunEventTypeResuming,
		RunID:      "run-retry",
		SessionID:  "session-retry",
		PayloadJSON: `{"status":"resuming"}`,
		Status:     "pending",
	})

	bus := NewApprovalEventBus()
	if err := bus.Subscribe("ai.run.*", func(ctx context.Context, event ApprovalEventEnvelope) error {
		return errors.New("boom")
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	dispatcher := NewApprovalEventDispatcher(db, bus)
	claimed, err := dispatcher.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected dispatch failure to return error")
	}
	if !claimed {
		t.Fatal("expected dispatcher to claim an event")
	}

	var outbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-retry", RunEventTypeResuming).First(&outbox).Error; err != nil {
		t.Fatalf("reload outbox: %v", err)
	}
	if outbox.Status != "pending" {
		t.Fatalf("expected retry to keep row pending, got %q", outbox.Status)
	}
	if outbox.RetryCount != 1 {
		t.Fatalf("expected retry_count=1, got %d", outbox.RetryCount)
	}
	if outbox.NextRetryAt == nil {
		t.Fatal("expected next_retry_at to be set")
	}
}

func testApprovalEnvelope(eventID, eventType, aggregateID string, sequence int64) ApprovalEventEnvelope {
	return ApprovalEventEnvelope{
		EventID:     eventID,
		EventType:   eventType,
		OccurredAt:  nowUTC(),
		Sequence:    sequence,
		Version:     1,
		RunID:       aggregateID,
		SessionID:   "session",
		ApprovalID:  "approval",
		ToolCallID:  "tool",
		AggregateID: aggregateID,
		PayloadJSON: `{"sequence":` + fmt.Sprintf("%d", sequence) + `}`,
	}
}

func nowUTC() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}
