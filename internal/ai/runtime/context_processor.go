package runtime

import (
	"context"
)

const (
	SessionKeyRuntimeContext = "ai.runtime_context"
	SessionKeySessionID      = "ai.session_id"
	SessionKeyPlanID         = "ai.plan_id"
	SessionKeyTurnID         = "ai.turn_id"
)

type runtimeContextKey struct{}

func ContextWithRuntimeContext(ctx context.Context, runtimeCtx RuntimeContext) context.Context {
	return context.WithValue(ctx, runtimeContextKey{}, runtimeCtx)
}

func RuntimeContextFromContext(ctx context.Context) RuntimeContext {
	runtimeCtx, _ := ctx.Value(runtimeContextKey{}).(RuntimeContext)
	return runtimeCtx
}
