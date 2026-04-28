package middleware

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ObservabilityMetrics defines the metrics collection interface.
// Implementations can export to Prometheus, OpenTelemetry, or in-memory for testing.
type ObservabilityMetrics interface {
	RecordToolCall(toolName string, duration time.Duration, err error)
}

// ObservabilityMiddleware provides tracing and metrics for agent tool calls.
type ObservabilityMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	metrics ObservabilityMetrics
}

// NewObservabilityMiddleware creates a new observability middleware.
func NewObservabilityMiddleware(metrics ObservabilityMetrics) *ObservabilityMiddleware {
	return &ObservabilityMiddleware{metrics: metrics}
}

func (m *ObservabilityMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	toolName := ""
	if tCtx != nil {
		toolName = tCtx.Name
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		start := time.Now()
		result, err := endpoint(ctx, args, opts...)
		if m.metrics != nil {
			m.metrics.RecordToolCall(toolName, time.Since(start), err)
		}
		return result, err
	}, nil
}

func (m *ObservabilityMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	toolName := ""
	if tCtx != nil {
		toolName = tCtx.Name
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		start := time.Now()
		result, err := endpoint(ctx, args, opts...)
		if m.metrics != nil {
			m.metrics.RecordToolCall(toolName, time.Since(start), err)
		}
		return result, err
	}, nil
}
