package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

type captureApprovalEvaluator struct {
	meta common.ApprovalEvalMeta
}

func (c *captureApprovalEvaluator) Evaluate(ctx context.Context, toolName string, args string, meta common.ApprovalEvalMeta) (*common.ApprovalDecision, error) {
	c.meta = meta
	return &common.ApprovalDecision{RequiresApproval: false}, nil
}

func TestApprovalMiddlewarePropagatesCheckpointAndBatchCommandClass(t *testing.T) {
	cases := []struct {
		name           string
		toolName       string
		args           string
		wantClass      string
		wantCheckpoint string
	}{
		{
			name:           "readonly batch command",
			toolName:       "host_batch_exec_preview",
			args:           `{"host_ids":[1,2],"command":"uptime"}`,
			wantClass:      "readonly",
			wantCheckpoint: "checkpoint-1",
		},
		{
			name:           "service control batch command",
			toolName:       "host_batch_exec_apply",
			args:           `{"host_ids":[1,2],"command":"systemctl restart nginx"}`,
			wantClass:      "service_control",
			wantCheckpoint: "checkpoint-2",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capture := &captureApprovalEvaluator{}
			mw := &approvalMiddleware{
				config: &ApprovalMiddlewareConfig{
					Orchestrator:     capture,
					NeedsApproval:    DefaultNeedsApproval,
					PreviewGenerator: DefaultPreviewGenerator,
					DefaultTimeout:   common.DefaultApprovalTimeout,
					ToolConfigs:      DefaultToolConfigs(),
				},
			}
			ctx := runtimectx.WithAIMetadata(context.Background(), runtimectx.AIMetadata{
				SessionID:    "session-1",
				RunID:        "run-1",
				CheckpointID: tc.wantCheckpoint,
				Scene:        "cluster",
				UserID:       1001,
			})

			decision, _, wasInterrupted, err := mw.evaluateApproval(ctx, tc.toolName, tc.args, "call-1")
			if err != nil {
				t.Fatalf("evaluate approval: %v", err)
			}
			if wasInterrupted {
				t.Fatal("did not expect interrupted flow for evaluator capture test")
			}
			if decision == nil {
				t.Fatal("expected a decision")
			}
			if capture.meta.CheckpointID != tc.wantCheckpoint {
				t.Fatalf("expected checkpoint id %q, got %q", tc.wantCheckpoint, capture.meta.CheckpointID)
			}
			if capture.meta.CommandClass != tc.wantClass {
				t.Fatalf("expected command class %q, got %q", tc.wantClass, capture.meta.CommandClass)
			}
		})
	}
}

func TestApprovalBridge_ReturnsSuspendedPayload(t *testing.T) {
	now := time.Now().UTC()
	decision := &common.ApprovalDecision{
		ApprovalID:     "approval-1",
		TimeoutSeconds: 300,
		DecisionSource: "fallback_static",
		ExpiresAt:      now,
	}
	info := buildApprovalInterruptInfo("host_exec_change", "call-1", decision)
	if got, _ := info["status"].(string); got != "suspended" {
		t.Fatalf("expected suspended status, got %v", info["status"])
	}
	if got, _ := info["approval_id"].(string); got != "approval-1" {
		t.Fatalf("expected approval_id approval-1, got %v", info["approval_id"])
	}
}

func TestApprovalResume_RejectsMismatchedSessionOrRole(t *testing.T) {
	mw := &approvalMiddleware{}
	decision := &common.ApprovalDecision{
		BoundSessionID: "session-a",
		BoundAgentRole: "diagnosis",
	}

	ctxSessionMismatch := runtimectx.WithAIMetadata(context.Background(), runtimectx.AIMetadata{SessionID: "session-b"})
	ctxSessionMismatch = runtimectx.WithContext(ctxSessionMismatch, &runtimectx.Context{Role: "diagnosis"})
	if mw.resumeBindingMatches(ctxSessionMismatch, decision) {
		t.Fatal("expected session mismatch to fail binding check")
	}

	ctxRoleMismatch := runtimectx.WithAIMetadata(context.Background(), runtimectx.AIMetadata{SessionID: "session-a"})
	ctxRoleMismatch = runtimectx.WithContext(ctxRoleMismatch, &runtimectx.Context{Role: "change"})
	if mw.resumeBindingMatches(ctxRoleMismatch, decision) {
		t.Fatal("expected role mismatch to fail binding check")
	}
}

func TestDefaultNeedsApproval_CoversHostExecChange(t *testing.T) {
	if !DefaultNeedsApproval("host_exec_change") {
		t.Fatal("expected host_exec_change to require approval")
	}
}

func TestCommandClassForTool_HostExecChange(t *testing.T) {
	if got := commandClassForTool("host_exec_change", `{"command":"systemctl restart nginx"}`); got != "service_control" {
		t.Fatalf("expected service_control, got %q", got)
	}
}

func TestApprovalAudit_RecordsApproverAndTimestamp(t *testing.T) {
	now := time.Now().UTC()
	decision := &common.ApprovalDecision{
		ApprovalID:        "approval-2",
		TimeoutSeconds:    300,
		DecisionSource:    "fallback_static",
		ExpiresAt:         now,
		ApproverID:        "user-42",
		ApprovalTimestamp: &now,
	}
	info := buildApprovalInterruptInfo("host_exec_change", "call-2", decision)
	if got, _ := info["approver_id"].(string); got != "user-42" {
		t.Fatalf("expected approver_id user-42, got %v", info["approver_id"])
	}
	if _, ok := info["approval_timestamp"].(string); !ok {
		t.Fatalf("expected approval_timestamp string, got %T", info["approval_timestamp"])
	}
}

func TestApprovalAudit_RecordsRejectReason(t *testing.T) {
	now := time.Now().UTC()
	decision := &common.ApprovalDecision{
		ApprovalID:     "approval-3",
		TimeoutSeconds: 300,
		DecisionSource: "fallback_static",
		ExpiresAt:      now,
		RejectReason:   "too risky",
	}
	info := buildApprovalInterruptInfo("host_exec_change", "call-3", decision)
	if got, _ := info["reject_reason"].(string); got != "too risky" {
		t.Fatalf("expected reject_reason too risky, got %v", info["reject_reason"])
	}
}

func TestApprovalAudit_RecordsParseFailuresAndViolations(t *testing.T) {
	now := time.Now().UTC()
	decision := &common.ApprovalDecision{
		ApprovalID:       "approval-4",
		TimeoutSeconds:   300,
		DecisionSource:   "fallback_static",
		ExpiresAt:        now,
		PolicyViolations: []string{"parse_error", "policy_violation"},
	}
	info := buildApprovalInterruptInfo("host_exec_readonly", "call-4", decision)
	raw, ok := info["policy_violations"].([]string)
	if !ok {
		t.Fatalf("expected policy_violations []string, got %T", info["policy_violations"])
	}
	if len(raw) != 2 {
		t.Fatalf("expected 2 policy violations, got %v", raw)
	}
}
