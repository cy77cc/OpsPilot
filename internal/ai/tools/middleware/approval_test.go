package middleware

import (
	"context"
	"testing"

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
