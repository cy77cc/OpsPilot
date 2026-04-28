package approval

import (
	"testing"
	"time"
)

func TestNewApprovalEventEnvelope_Unified(t *testing.T) {
	input := ApprovalEventInput{
		EventID:     "evt-1",
		OccurredAt:  time.Now(),
		Sequence:    1,
		Version:     1,
		RunID:       "run-1",
		SessionID:   "sess-1",
		ApprovalID:  "appr-1",
		ToolCallID:  "call-1",
		AggregateID: "run-1",
		Payload:     map[string]string{"key": "value"},
	}

	env, err := NewApprovalRequestedEnvelope(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.EventType != ApprovalEventTypeRequested {
		t.Errorf("expected %q, got %q", ApprovalEventTypeRequested, env.EventType)
	}
	if env.ApprovalID != "appr-1" {
		t.Errorf("expected approval_id appr-1, got %q", env.ApprovalID)
	}
}

func TestNewApprovalEventEnvelope_AllTypes(t *testing.T) {
	input := ApprovalEventInput{
		EventID:     "evt-1",
		Sequence:    1,
		RunID:       "run-1",
		SessionID:   "sess-1",
		ApprovalID:  "appr-1",
		ToolCallID:  "call-1",
		AggregateID: "run-1",
		Payload:     map[string]string{},
	}

	tests := []struct {
		name        string
		constructor func(ApprovalEventInput) (*ApprovalEventEnvelope, error)
		wantType    string
	}{
		{"Requested", NewApprovalRequestedEnvelope, ApprovalEventTypeRequested},
		{"Decided", NewApprovalDecidedEnvelope, ApprovalEventTypeDecided},
		{"Expired", NewApprovalExpiredEnvelope, ApprovalEventTypeExpired},
		{"Resuming", NewRunResumingEnvelope, RunEventTypeResuming},
		{"Resumed", NewRunResumedEnvelope, RunEventTypeResumed},
		{"ResumeFailed", NewRunResumeFailedEnvelope, RunEventTypeResumeFailed},
		{"Completed", NewRunCompletedEnvelope, RunEventTypeCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := tt.constructor(input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if env.EventType != tt.wantType {
				t.Errorf("expected %q, got %q", tt.wantType, env.EventType)
			}
		})
	}
}

func TestNewApprovalEventEnvelope_Validation(t *testing.T) {
	input := ApprovalEventInput{}
	_, err := NewApprovalRequestedEnvelope(input)
	if err == nil {
		t.Error("expected error for missing required fields")
	}
}
