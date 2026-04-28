package toolutil

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// ServiceContextFromRuntime extracts the ServiceContext from the runtime context.
// This is the single shared implementation used by all tool packages.
func ServiceContextFromRuntime(ctx context.Context) *svc.ServiceContext {
	svcCtx, _ := runtimectx.ServicesAs[*svc.ServiceContext](ctx)
	return svcCtx
}
