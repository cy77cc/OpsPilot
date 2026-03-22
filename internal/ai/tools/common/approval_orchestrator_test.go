package common

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
)

func TestFallbackRequiresApproval_CoversHostExecChange(t *testing.T) {
	if !fallbackRequiresApproval("host_exec_change", "service_control") {
		t.Fatal("expected host_exec_change to require approval in fallback policy")
	}
}

type noopPolicyStore struct{}

func (noopPolicyStore) ListEnabledByToolName(ctx context.Context, toolName string) ([]model.AIToolRiskPolicy, error) {
	return nil, nil
}

type captureApprovalTaskWriter struct{}

func (captureApprovalTaskWriter) Create(ctx context.Context, task *model.AIApprovalTask) error {
	return nil
}

type captureApprovalOutboxWriter struct {
	event *model.AIApprovalOutboxEvent
}

func (w *captureApprovalOutboxWriter) EnqueueOrTouch(ctx context.Context, event *model.AIApprovalOutboxEvent) error {
	w.event = event
	return nil
}

func TestApprovalOrchestrator_AuditPayloadContainsApprovalAuditKeys(t *testing.T) {
	outbox := &captureApprovalOutboxWriter{}
	o := NewApprovalOrchestratorWithStores(noopPolicyStore{}, captureApprovalTaskWriter{}, outbox)
	now := time.Now().UTC()
	o.now = func() time.Time { return now }
	o.newID = func() string { return "approval-1" }

	meta := ApprovalEvalMeta{
		SessionID:    "session-1",
		RunID:        "run-1",
		CheckpointID: "checkpoint-1",
		CallID:       "call-1",
	}
	if _, err := o.createApprovalDecision(context.Background(), "host_exec_change", `{"command":"systemctl restart nginx"}`, meta, now, 300, nil, "fallback_safe"); err != nil {
		t.Fatalf("create approval decision: %v", err)
	}
	if outbox.event == nil {
		t.Fatal("expected outbox event")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(outbox.event.PayloadJSON), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if _, ok := payload["approver_id"]; !ok {
		t.Fatal("expected approver_id key in payload")
	}
	if _, ok := payload["approval_timestamp"]; !ok {
		t.Fatal("expected approval_timestamp key in payload")
	}
	if _, ok := payload["reject_reason"]; !ok {
		t.Fatal("expected reject_reason key in payload")
	}
}
