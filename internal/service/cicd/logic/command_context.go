package logic

import (
	"context"
	"encoding/json"

	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

type CommandAuditContext struct {
	CommandID       string         `json:"command_id"`
	Intent          string         `json:"intent"`
	PlanHash        string         `json:"plan_hash"`
	TraceID         string         `json:"trace_id"`
	ApprovalContext map[string]any `json:"approval_context,omitempty"`
	Summary         string         `json:"summary,omitempty"`
}

type commandAuditContextKey struct{}

func WithCommandAuditContext(ctx context.Context, meta CommandAuditContext) context.Context {
	return runtimectx.WithValue(ctx, commandAuditContextKey{}, meta)
}

func commandAuditContextFromContext(ctx context.Context) (CommandAuditContext, bool) {
	if ctx == nil {
		return CommandAuditContext{}, false
	}
	meta, ok := runtimectx.Value(ctx, commandAuditContextKey{}).(CommandAuditContext)
	if !ok {
		return CommandAuditContext{}, false
	}
	return meta, true
}

func mustJSONOrEmpty(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
