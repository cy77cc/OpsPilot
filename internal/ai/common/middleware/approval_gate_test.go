package middleware

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/common/approval"
)

type noopApprovalEvaluator struct{}

func (noopApprovalEvaluator) Evaluate(context.Context, string, string, approval.ApprovalEvalMeta) (*approval.ApprovalDecision, error) {
	return &approval.ApprovalDecision{RequiresApproval: false}, nil
}

func TestRequiresApprovalGateBypassesTaskTool(t *testing.T) {
	mw := &approvalMiddleware{
		config: &ApprovalMiddlewareConfig{
			Orchestrator:  noopApprovalEvaluator{},
			NeedsApproval: DefaultNeedsApproval,
		},
	}
	if mw.requiresApprovalGate("task") {
		t.Fatal("expected task tool to bypass approval gate")
	}
	if !mw.requiresApprovalGate("host_exec") {
		t.Fatal("expected host_exec to remain guarded when orchestrator is enabled")
	}
}

func TestDefaultNeedsApprovalTaskIsFalse(t *testing.T) {
	if DefaultNeedsApproval("task") {
		t.Fatal("expected task tool not to require approval by default")
	}
	if !DefaultNeedsApproval("host_exec") {
		t.Fatal("expected host_exec to require approval by default")
	}
}
