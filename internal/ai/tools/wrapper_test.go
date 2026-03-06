package tools

import (
	"context"
	"encoding/gob"
	"errors"
	"bytes"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type fakeInvokableTool struct {
	name   string
	result string
	err    error
	calls  int
	last   string
}

func (f *fakeInvokableTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: f.name}, nil
}

func (f *fakeInvokableTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	f.calls++
	f.last = args
	if f.err != nil {
		return "", f.err
	}
	return f.result, nil
}

func TestApprovableTool_FirstRunInterrupts(t *testing.T) {
	base := &fakeInvokableTool{name: "service_deploy_apply", result: "ok"}
	wrapped := NewApprovableTool(base, ToolRiskHigh,
		func(_ context.Context, args string) (map[string]any, error) {
			return map[string]any{"args": args}, nil
		})

	out, err := wrapped.InvokableRun(context.Background(), `{"service_id":1}`)
	if err == nil {
		t.Fatalf("expected interrupt error on first run")
	}
	if out != "" {
		t.Fatalf("unexpected output: %q", out)
	}
	if base.calls != 0 {
		t.Fatalf("wrapped tool should not execute on first run")
	}
}

func TestApprovableTool_PreviewError(t *testing.T) {
	want := errors.New("preview failed")
	wrapped := NewApprovableTool(&fakeInvokableTool{name: "service_deploy_apply"}, ToolRiskHigh,
		func(_ context.Context, _ string) (map[string]any, error) {
			return nil, want
		})

	_, err := wrapped.InvokableRun(context.Background(), `{"service_id":1}`)
	if err == nil {
		t.Fatalf("expected preview error")
	}
	if !errors.Is(err, want) {
		t.Fatalf("expected wrapped preview error, got %v", err)
	}
}

func TestReviewableTool_FirstRunInterrupts(t *testing.T) {
	base := &fakeInvokableTool{name: "host_ssh_exec_readonly", result: "ok"}
	wrapped := NewReviewableTool(base)

	out, err := wrapped.InvokableRun(context.Background(), `{"host_id":1,"command":"uptime"}`)
	if err == nil {
		t.Fatalf("expected interrupt error on first run")
	}
	if out != "" {
		t.Fatalf("unexpected output: %q", out)
	}
	if base.calls != 0 {
		t.Fatalf("wrapped tool should not execute on first run")
	}
}

func TestApprovalInfoPreviewSupportsGobSerialization(t *testing.T) {
	var buf bytes.Buffer
	payload := struct {
		Value any
	}{
		Value: &ApprovalInfo{
			ToolName:        "host_batch_exec_apply",
			ArgumentsInJSON: `{"host_ids":[2]}`,
			Risk:            ToolRiskHigh,
			Preview: map[string]any{
				"tool": "host_batch_exec_apply",
				"arguments": map[string]any{
					"host_ids": []any{2, 3},
					"options": map[string]any{
						"mode": "script",
					},
				},
			},
		},
	}

	if err := gob.NewEncoder(&buf).Encode(payload); err != nil {
		t.Fatalf("expected approval payload to be gob encodable, got %v", err)
	}
}
