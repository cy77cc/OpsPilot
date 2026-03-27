package logic

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
)

func TestRunTailer_WaitsForNewEventsWhenRunStillOpen(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-tailer-open",
		userID:             77,
		runID:              "run-tailer-open",
		userMessageID:      "msg-tailer-user",
		assistantMessageID: "msg-tailer-assistant",
		runStatus:          "resuming",
	})
	if err := db.Create(&model.AIRunEvent{
		ID:          "evt-1",
		RunID:       "run-tailer-open",
		SessionID:   "session-tailer-open",
		Seq:         1,
		EventType:   "run_state",
		PayloadJSON: `{"status":"resuming"}`,
	}).Error; err != nil {
		t.Fatalf("seed cursor event: %v", err)
	}

	l := newApprovalWorkerTestLogic(db)
	tailer := &RunTailer{
		RunDAO:      l.RunDAO,
		RunEventDAO: l.RunEventDAO,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		mu      sync.Mutex
		emitted []string
	)
	streamDone := make(chan error, 1)
	go func() {
		streamDone <- tailer.ReplayThenTail(ctx, "run-tailer-open", "evt-1", func(event string, data any) {
			mu.Lock()
			defer mu.Unlock()
			emitted = append(emitted, event)
		}, TailOptions{
			PollInterval:    5 * time.Millisecond,
			IdleTimeout:     250 * time.Millisecond,
			MaxTailDuration: time.Second,
		})
	}()

	select {
	case err := <-streamDone:
		if err != nil {
			t.Fatalf("tail returned error: %v", err)
		}
		t.Fatal("expected tailer to stay attached while the run remains open")
	case <-time.After(50 * time.Millisecond):
	}

	if err := db.Create(&model.AIRunEvent{
		ID:          "evt-2",
		RunID:       "run-tailer-open",
		SessionID:   "session-tailer-open",
		Seq:         2,
		EventType:   "run_state",
		PayloadJSON: `{"status":"running"}`,
	}).Error; err != nil {
		t.Fatalf("seed follow-up event: %v", err)
	}

	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		mu.Lock()
		count := len(emitted)
		mu.Unlock()
		if count > 0 {
			cancel()
			if err := <-streamDone; err != nil {
				t.Fatalf("tail shutdown: %v", err)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-streamDone:
		if err != nil {
			t.Fatalf("tail shutdown: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected tailer to stop after context cancellation")
	}
	t.Fatal("expected tailer to emit the follow-up event once it was appended")
}

func TestRunTailer_IdleTimeoutAllowsSafeReattachFromSameCursor(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-tailer-idle",
		userID:             78,
		runID:              "run-tailer-idle",
		userMessageID:      "msg-tailer-idle-user",
		assistantMessageID: "msg-tailer-idle-assistant",
		runStatus:          "waiting_approval",
	})
	if err := db.Create(&model.AIRunEvent{
		ID:          "evt-1",
		RunID:       "run-tailer-idle",
		SessionID:   "session-tailer-idle",
		Seq:         1,
		EventType:   "run_state",
		PayloadJSON: `{"status":"waiting_approval"}`,
	}).Error; err != nil {
		t.Fatalf("seed cursor event: %v", err)
	}

	l := newApprovalWorkerTestLogic(db)
	tailer := &RunTailer{
		RunDAO:      l.RunDAO,
		RunEventDAO: l.RunEventDAO,
	}

	start := time.Now()
	if err := tailer.ReplayThenTail(context.Background(), "run-tailer-idle", "evt-1", func(string, any) {}, TailOptions{
		PollInterval:    5 * time.Millisecond,
		IdleTimeout:     40 * time.Millisecond,
		MaxTailDuration: time.Second,
	}); err != nil {
		t.Fatalf("first attach returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("expected first attach to wait for idle timeout, returned too quickly in %s", elapsed)
	}

	if err := db.Create(&model.AIRunEvent{
		ID:          "evt-2",
		RunID:       "run-tailer-idle",
		SessionID:   "session-tailer-idle",
		Seq:         2,
		EventType:   "run_state",
		PayloadJSON: `{"status":"waiting_approval","checkpoint":"second-attach"}`,
	}).Error; err != nil {
		t.Fatalf("seed replay event: %v", err)
	}

	var emitted []string
	if err := tailer.ReplayThenTail(context.Background(), "run-tailer-idle", "evt-1", func(event string, data any) {
		emitted = append(emitted, event)
	}, TailOptions{
		PollInterval:    5 * time.Millisecond,
		IdleTimeout:     40 * time.Millisecond,
		MaxTailDuration: time.Second,
	}); err != nil {
		t.Fatalf("second attach returned error: %v", err)
	}

	if len(emitted) != 1 || emitted[0] != "run_state" {
		t.Fatalf("expected second attach from same cursor to replay only the new event, got %#v", emitted)
	}
}

func TestRunTailer_StopsImmediatelyWhenClientContextCancels(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-tailer-cancel",
		userID:             79,
		runID:              "run-tailer-cancel",
		userMessageID:      "msg-tailer-cancel-user",
		assistantMessageID: "msg-tailer-cancel-assistant",
		runStatus:          "running",
	})

	l := newApprovalWorkerTestLogic(db)
	tailer := &RunTailer{
		RunDAO:      l.RunDAO,
		RunEventDAO: l.RunEventDAO,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	if err := tailer.ReplayThenTail(ctx, "run-tailer-cancel", "", func(string, any) {
		t.Fatal("expected canceled client context to prevent event emission")
	}, TailOptions{
		PollInterval:    5 * time.Millisecond,
		IdleTimeout:     time.Second,
		MaxTailDuration: time.Second,
	}); err != nil {
		t.Fatalf("tail returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("expected client cancellation to stop tailing immediately, took %s", elapsed)
	}
}

func TestRunTailer_AbsoluteTailDeadlineForcesGracefulDisconnect(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-tailer-deadline",
		userID:             80,
		runID:              "run-tailer-deadline",
		userMessageID:      "msg-tailer-deadline-user",
		assistantMessageID: "msg-tailer-deadline-assistant",
		runStatus:          "waiting_approval",
	})
	if err := db.Create(&model.AIRunEvent{
		ID:          "evt-1",
		RunID:       "run-tailer-deadline",
		SessionID:   "session-tailer-deadline",
		Seq:         1,
		EventType:   "run_state",
		PayloadJSON: `{"status":"waiting_approval"}`,
	}).Error; err != nil {
		t.Fatalf("seed cursor event: %v", err)
	}

	l := newApprovalWorkerTestLogic(db)
	tailer := &RunTailer{
		RunDAO:      l.RunDAO,
		RunEventDAO: l.RunEventDAO,
	}

	start := time.Now()
	if err := tailer.ReplayThenTail(context.Background(), "run-tailer-deadline", "evt-1", func(string, any) {}, TailOptions{
		PollInterval:    5 * time.Millisecond,
		IdleTimeout:     time.Second,
		MaxTailDuration: 40 * time.Millisecond,
	}); err != nil {
		t.Fatalf("tail returned error: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < 30*time.Millisecond || elapsed > 250*time.Millisecond {
		t.Fatalf("expected max tail duration to stop the tailer near its deadline, took %s", elapsed)
	}
}

func TestRunTailer_EmitsTerminalRunStateWhenCursorAlreadyAtEnd(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-tailer-terminal",
		userID:             81,
		runID:              "run-tailer-terminal",
		userMessageID:      "msg-tailer-terminal-user",
		assistantMessageID: "msg-tailer-terminal-assistant",
		runStatus:          "completed",
	})
	if err := db.Create(&model.AIRunEvent{
		ID:          "evt-1",
		RunID:       "run-tailer-terminal",
		SessionID:   "session-tailer-terminal",
		Seq:         1,
		EventType:   "done",
		PayloadJSON: `{"run_id":"run-tailer-terminal","status":"completed"}`,
	}).Error; err != nil {
		t.Fatalf("seed terminal event: %v", err)
	}

	l := newApprovalWorkerTestLogic(db)
	tailer := &RunTailer{
		RunDAO:      l.RunDAO,
		RunEventDAO: l.RunEventDAO,
	}

	var (
		emittedEvent string
		emittedData  map[string]any
	)
	if err := tailer.ReplayThenTail(context.Background(), "run-tailer-terminal", "evt-1", func(event string, data any) {
		emittedEvent = event
		payload, _ := data.(map[string]any)
		emittedData = payload
	}, TailOptions{
		PollInterval:    5 * time.Millisecond,
		IdleTimeout:     50 * time.Millisecond,
		MaxTailDuration: 100 * time.Millisecond,
	}); err != nil {
		t.Fatalf("tail returned error: %v", err)
	}

	if emittedEvent != "run_state" {
		t.Fatalf("expected synthetic terminal run_state event, got %q payload=%#v", emittedEvent, emittedData)
	}
	if emittedData["run_id"] != "run-tailer-terminal" || emittedData["status"] != "completed" {
		t.Fatalf("expected terminal payload with run_id/status, got %#v", emittedData)
	}
}
