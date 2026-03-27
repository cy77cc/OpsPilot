package middleware

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/ai/common/approval"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// BuildAgentHandlers builds the default handlers chain:
// approval middleware + arg normalization middleware.
func BuildAgentHandlers(ctx context.Context, tools []tool.BaseTool) ([]adk.ChatModelAgentMiddleware, error) {
	argMw, err := NewArgNormalizationHandler(ctx, tools, &ArgNormalizeConfig{
		Enabled:    true,
		ShadowMode: false,
	})
	if err != nil {
		return nil, err
	}
	cfg := &ApprovalMiddlewareConfig{}
	if svcCtx, ok := runtimectx.ServicesAs[*svc.ServiceContext](ctx); ok && svcCtx != nil && svcCtx.DB != nil {
		cfg.Orchestrator = approval.NewApprovalOrchestrator(svcCtx.DB)
	}
	return []adk.ChatModelAgentMiddleware{
		ApprovalMiddleware(cfg),
		argMw,
	}, nil
}
