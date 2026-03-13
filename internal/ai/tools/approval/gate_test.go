package approval

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

type stubTool struct {
	name      string
	result    string
	calls     int
	lastArgs  string
	returnErr error
}

func (s *stubTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: s.name, Desc: s.name}, nil
}

func (s *stubTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	s.calls++
	s.lastArgs = argumentsInJSON
	return s.result, s.returnErr
}

func TestGatePassesReadonlyToolThrough(t *testing.T) {
	inner := &stubTool{name: "service_status", result: `{"ok":true}`}
	gate := NewGate(inner, airuntime.ApprovalToolSpec{
		Name: "service_status",
		Mode: "readonly",
		Risk: "low",
	}, nil, NewSummaryRenderer())

	got, err := gate.InvokableRun(context.Background(), `{"service_id":1}`)
	if err != nil {
		t.Fatalf("InvokableRun() error = %v", err)
	}
	if got != `{"ok":true}` {
		t.Fatalf("result = %q", got)
	}
	if inner.calls != 1 {
		t.Fatalf("calls = %d, want 1", inner.calls)
	}
}

func TestGateInterruptsMutatingToolBeforeExecution(t *testing.T) {
	inner := &stubTool{name: "service_deploy_apply", result: `{"ok":true}`}
	gate := NewGate(inner, airuntime.ApprovalToolSpec{
		Name:        "service_deploy_apply",
		DisplayName: "Service Deploy Apply",
		Mode:        "mutating",
		Risk:        "medium",
	}, nil, NewSummaryRenderer())

	_, err := gate.InvokableRun(context.Background(), `{"service_id":1,"namespace":"prod"}`)
	if err == nil {
		t.Fatalf("InvokableRun() error = nil, want interrupt")
	}
	if !strings.Contains(err.Error(), "interrupt signal") {
		t.Fatalf("error is not interrupt: %v", err)
	}
	if inner.calls != 0 {
		t.Fatalf("calls = %d, want 0", inner.calls)
	}
	if !strings.Contains(err.Error(), "service_deploy_apply") {
		t.Fatalf("interrupt payload does not mention tool name: %v", err)
	}
}

func TestGateResumesApprovedExecution(t *testing.T) {
	inner := &stubTool{name: "service_deploy_apply", result: `{"ok":true}`}
	gate := NewGate(inner, airuntime.ApprovalToolSpec{
		Name:        "service_deploy_apply",
		DisplayName: "Service Deploy Apply",
		Mode:        "mutating",
		Risk:        "medium",
	}, nil, NewSummaryRenderer())

	_, err := gate.InvokableRun(context.Background(), `{"service_id":1}`)
	if err == nil {
		t.Fatalf("first InvokableRun() error = nil, want interrupt")
	}
	got, err := gate.executeResumed(context.Background(), interruptState{
		ArgumentsInJSON: `{"service_id":1}`,
		Approved:        true,
	})
	if err != nil {
		t.Fatalf("resume InvokableRun() error = %v", err)
	}
	if got != `{"ok":true}` {
		t.Fatalf("result = %q", got)
	}
	if inner.calls != 1 {
		t.Fatalf("calls = %d, want 1", inner.calls)
	}
	if inner.lastArgs != `{"service_id":1}` {
		t.Fatalf("lastArgs = %q", inner.lastArgs)
	}
}
