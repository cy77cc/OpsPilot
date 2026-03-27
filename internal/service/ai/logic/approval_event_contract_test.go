package logic

import (
	"encoding/json"
	"testing"
	"time"
)

func TestApprovalEnvelope_RequiresCoreFields(t *testing.T) {
	t.Parallel()

	envelope, err := NewApprovalRequestedEnvelope(ApprovalRequestedInput{
		EventID:     "evt-req-1",
		OccurredAt:   time.Unix(100, 0).UTC(),
		Sequence:    1,
		Version:     1,
		RunID:       "run-1",
		SessionID:   "session-1",
		ApprovalID:  "approval-1",
		ToolCallID:  "tool-call-1",
		AggregateID: "run-1",
		Payload: map[string]any{
			"approval_id": "approval-1",
			"tool_call_id": "tool-call-1",
		},
	})
	if err != nil {
		t.Fatalf("build requested envelope: %v", err)
	}

	if envelope.EventID == "" {
		t.Fatal("expected event id to be populated")
	}
	if envelope.EventType == "" {
		t.Fatal("expected event type to be populated")
	}
	if envelope.OccurredAt.IsZero() {
		t.Fatal("expected occurred_at to be populated")
	}
	if envelope.Sequence <= 0 {
		t.Fatal("expected sequence to be positive")
	}
	if envelope.Version <= 0 {
		t.Fatal("expected version to be positive")
	}
	if envelope.RunID == "" || envelope.SessionID == "" || envelope.ApprovalID == "" || envelope.ToolCallID == "" || envelope.AggregateID == "" {
		t.Fatalf("expected all identifiers to be populated, got %#v", envelope)
	}
	if len(envelope.PayloadJSON) == 0 {
		t.Fatal("expected payload json to be populated")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(envelope.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode payload json: %v", err)
	}
	if payload["approval_id"] != "approval-1" {
		t.Fatalf("expected payload to round-trip, got %#v", payload)
	}
}

func TestApprovalEnvelope_SequenceMonotonicPerRunID(t *testing.T) {
	t.Parallel()

	requested, err := NewApprovalRequestedEnvelope(ApprovalRequestedInput{
		EventID:     "evt-req-1",
		OccurredAt:   time.Unix(100, 0).UTC(),
		Sequence:    1,
		Version:     1,
		RunID:       "run-1",
		SessionID:   "session-1",
		ApprovalID:  "approval-1",
		ToolCallID:  "tool-call-1",
		AggregateID: "run-1",
		Payload:     map[string]any{"approval_id": "approval-1"},
	})
	if err != nil {
		t.Fatalf("build requested envelope: %v", err)
	}

	decided, err := NewApprovalDecidedEnvelope(ApprovalDecidedInput{
		EventID:     "evt-dec-1",
		OccurredAt:   time.Unix(101, 0).UTC(),
		Sequence:    2,
		Version:     1,
		RunID:       "run-1",
		SessionID:   "session-1",
		ApprovalID:  "approval-1",
		ToolCallID:  "tool-call-1",
		AggregateID: "run-1",
		Payload: map[string]any{
			"approval_id": "approval-1",
			"approved":    true,
		},
	})
	if err != nil {
		t.Fatalf("build decided envelope: %v", err)
	}

	if decided.RunID != requested.RunID {
		t.Fatalf("expected same run id, got %q and %q", requested.RunID, decided.RunID)
	}
	if decided.Sequence <= requested.Sequence {
		t.Fatalf("expected sequence to increase per run, got %d then %d", requested.Sequence, decided.Sequence)
	}
}

func TestApprovalPayloadBuilders_EmitStableEventTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		build   func() (*ApprovalEventEnvelope, error)
		wantType string
	}{
		{
			name: "requested",
			build: func() (*ApprovalEventEnvelope, error) {
				return NewApprovalRequestedEnvelope(ApprovalRequestedInput{
					EventID:     "evt-req-1",
					OccurredAt:   time.Unix(100, 0).UTC(),
					Sequence:    1,
					Version:     1,
					RunID:       "run-1",
					SessionID:   "session-1",
					ApprovalID:  "approval-1",
					ToolCallID:  "tool-call-1",
					AggregateID: "run-1",
					Payload:     map[string]any{"approval_id": "approval-1"},
				})
			},
			wantType: "ai.approval.requested",
		},
		{
			name: "decided",
			build: func() (*ApprovalEventEnvelope, error) {
				return NewApprovalDecidedEnvelope(ApprovalDecidedInput{
					EventID:     "evt-dec-1",
					OccurredAt:   time.Unix(101, 0).UTC(),
					Sequence:    2,
					Version:     1,
					RunID:       "run-1",
					SessionID:   "session-1",
					ApprovalID:  "approval-1",
					ToolCallID:  "tool-call-1",
					AggregateID: "run-1",
					Payload:     map[string]any{"approval_id": "approval-1", "approved": true},
				})
			},
			wantType: "ai.approval.decided",
		},
		{
			name: "expired",
			build: func() (*ApprovalEventEnvelope, error) {
				return NewApprovalExpiredEnvelope(ApprovalExpiredInput{
					EventID:     "evt-exp-1",
					OccurredAt:   time.Unix(102, 0).UTC(),
					Sequence:    3,
					Version:     1,
					RunID:       "run-1",
					SessionID:   "session-1",
					ApprovalID:  "approval-1",
					ToolCallID:  "tool-call-1",
					AggregateID: "run-1",
					Payload:     map[string]any{"approval_id": "approval-1"},
				})
			},
			wantType: "ai.approval.expired",
		},
		{
			name: "run-resuming",
			build: func() (*ApprovalEventEnvelope, error) {
				return NewRunResumingEnvelope(RunResumingInput{
					EventID:     "evt-rs-1",
					OccurredAt:   time.Unix(103, 0).UTC(),
					Sequence:    4,
					Version:     1,
					RunID:       "run-1",
					SessionID:   "session-1",
					ApprovalID:  "approval-1",
					ToolCallID:  "tool-call-1",
					AggregateID: "run-1",
					Payload:     map[string]any{"run_id": "run-1"},
				})
			},
			wantType: "ai.run.resuming",
		},
		{
			name: "run-resumed",
			build: func() (*ApprovalEventEnvelope, error) {
				return NewRunResumedEnvelope(RunResumedInput{
					EventID:     "evt-rsd-1",
					OccurredAt:   time.Unix(104, 0).UTC(),
					Sequence:    5,
					Version:     1,
					RunID:       "run-1",
					SessionID:   "session-1",
					ApprovalID:  "approval-1",
					ToolCallID:  "tool-call-1",
					AggregateID: "run-1",
					Payload:     map[string]any{"run_id": "run-1"},
				})
			},
			wantType: "ai.run.resumed",
		},
		{
			name: "run-resume-failed",
			build: func() (*ApprovalEventEnvelope, error) {
				return NewRunResumeFailedEnvelope(RunResumeFailedInput{
					EventID:     "evt-rsf-1",
					OccurredAt:   time.Unix(105, 0).UTC(),
					Sequence:    6,
					Version:     1,
					RunID:       "run-1",
					SessionID:   "session-1",
					ApprovalID:  "approval-1",
					ToolCallID:  "tool-call-1",
					AggregateID: "run-1",
					Payload:     map[string]any{"run_id": "run-1", "retryable": true},
				})
			},
			wantType: "ai.run.resume_failed",
		},
		{
			name: "run-completed",
			build: func() (*ApprovalEventEnvelope, error) {
				return NewRunCompletedEnvelope(RunCompletedInput{
					EventID:     "evt-done-1",
					OccurredAt:   time.Unix(106, 0).UTC(),
					Sequence:    7,
					Version:     1,
					RunID:       "run-1",
					SessionID:   "session-1",
					ApprovalID:  "approval-1",
					ToolCallID:  "tool-call-1",
					AggregateID: "run-1",
					Payload:     map[string]any{"run_id": "run-1", "status": "completed"},
				})
			},
			wantType: "ai.run.completed",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			envelope, err := tc.build()
			if err != nil {
				t.Fatalf("build envelope: %v", err)
			}
			if envelope.EventType != tc.wantType {
				t.Fatalf("expected event type %q, got %q", tc.wantType, envelope.EventType)
			}
		})
	}
}
