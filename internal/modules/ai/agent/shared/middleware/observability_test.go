package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/middleware/metricsnoop"
)

func TestObservabilityMiddleware_RecordsToolCall(t *testing.T) {
	noop := metricsnoop.New()
	mw := NewObservabilityMiddleware(noop)

	called := false
	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		called = true
		return "ok", nil
	}

	wrapped, err := mw.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{
		Name: "test_tool",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, _ = wrapped(context.Background(), `{}`)

	if !called {
		t.Error("expected endpoint to be called")
	}

	metrics := noop.Snapshot()
	if metrics.ToolCallCount != 1 {
		t.Errorf("expected 1 tool call, got %d", metrics.ToolCallCount)
	}
	if metrics.ToolCallErrors != 0 {
		t.Errorf("expected 0 errors, got %d", metrics.ToolCallErrors)
	}
}

func TestObservabilityMiddleware_RecordsError(t *testing.T) {
	noop := metricsnoop.New()
	mw := NewObservabilityMiddleware(noop)

	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		return "", fmt.Errorf("tool failed")
	}

	wrapped, _ := mw.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{
		Name: "failing_tool",
	})
	_, _ = wrapped(context.Background(), `{}`)

	metrics := noop.Snapshot()
	if metrics.ToolCallErrors != 1 {
		t.Errorf("expected 1 error, got %d", metrics.ToolCallErrors)
	}
}

func TestObservabilityMiddleware_NilMetrics(t *testing.T) {
	mw := NewObservabilityMiddleware(nil)

	endpoint := func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		return "ok", nil
	}

	wrapped, err := mw.WrapInvokableToolCall(context.Background(), endpoint, &adk.ToolContext{
		Name: "test_tool",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not panic with nil metrics
	result, err := wrapped(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Errorf("expected 'ok', got %q", result)
	}
}
