package logic

import (
	"context"
	"encoding/json"
)

type auditContextKey struct{}

type CommandAuditContext struct {
	CommandID       string         `json:"command_id"`
	Intent          string         `json:"intent"`
	PlanHash        string         `json:"plan_hash"`
	TraceID         string         `json:"trace_id"`
	ApprovalContext map[string]any `json:"approval_context,omitempty"`
	Summary         string         `json:"summary,omitempty"`
}

func WithCommandAuditContext(ctx context.Context, meta CommandAuditContext) context.Context {
	return context.WithValue(ctx, auditContextKey{}, meta)
}

func commandAuditContextFromContext(ctx context.Context) (CommandAuditContext, bool) {
	if ctx == nil {
		return CommandAuditContext{}, false
	}
	meta, ok := ctx.Value(auditContextKey{}).(CommandAuditContext)
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
