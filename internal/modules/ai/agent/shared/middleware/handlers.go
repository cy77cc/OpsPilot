package middleware

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// BuildAgentHandlers builds the default handlers chain:
// scene router + audit logger + approval middleware + output summarizer + arg normalization middleware.
//
// 中间件执行顺序（洋葱模型）：
//  1. Scene Router: 设置场景边界，过滤工具
//  2. Audit Logger: 记录工具调用审计
//  3. Approval: 高风险工具审批拦截
//  4. Output Summarizer: 工具输出摘要
//  5. Arg Normalizer: 参数标准化
func BuildAgentHandlers(ctx context.Context, scene string, tools []tool.BaseTool) ([]adk.ChatModelAgentMiddleware, error) {
	var middlewares []adk.ChatModelAgentMiddleware

	// 1. 场景路由器（最先执行，设置边界）
	sceneRouter, err := NewSceneRouter(ctx, &SceneRouterConfig{
		SceneToolMap: DefaultSceneToolMap(),
	})
	if err != nil {
		return nil, err
	}
	middlewares = append(middlewares, sceneRouter)

	// 2. 审批中间件（高风险拦截）
	cfg := &ApprovalMiddlewareConfig{}
	if svcCtx, ok := runtimectx.ServicesAs[*svc.ServiceContext](ctx); ok && svcCtx != nil && svcCtx.DB != nil {
		cfg.Orchestrator = approval.NewApprovalOrchestrator(svcCtx.DB)
	}
	middlewares = append(middlewares, ApprovalMiddleware(cfg))

	// 3. 参数标准化器（最后执行）
	argMw, err := NewArgNormalizationHandler(ctx, tools, &ArgNormalizeConfig{
		Enabled:    true,
		ShadowMode: false,
	})
	if err != nil {
		return nil, err
	}
	middlewares = append(middlewares, argMw)

	return middlewares, nil
}
