package tools

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/cicd"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/deployment"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/governance"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/host"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/infrastructure"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/monitor"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/tools/service"
)

var (
	buildKubernetesTools         = kubernetes.NewKubernetesTools
	buildKubernetesReadonlyTools = kubernetes.NewKubernetesReadonlyTools
	buildCICDTools               = cicd.NewCICDTools
	buildCICDReadonlyTools       = cicd.NewCICDReadonlyTools
	buildMonitorReadonlyTools    = monitor.NewMonitorReadonlyTools
	buildHostTools               = host.NewHostTools
	buildHostReadonlyTools       = host.NewHostReadonlyTools
	buildServiceTools            = service.NewServiceTools
	buildServiceReadonlyTools    = service.NewServiceReadonlyTools
	buildDeploymentTools         = deployment.NewDeploymentTools
	buildInfrastructureTools     = infrastructure.NewInfrastructureTools
	buildGovernanceTools         = governance.NewGovernanceTools
)

func BuildToolsForScene(ctx context.Context, scene string) []tool.BaseTool {
	return BuildToolsForSceneWithMode(ctx, scene, false)
}

// BuildToolsForSceneWithMode 根据场景和访问模式构建工具列表。
func BuildToolsForSceneWithMode(ctx context.Context, scene string, readOnly bool) []tool.BaseTool {
	var tools []tool.InvokableTool

	switch normalizeScene(scene) {
	case "kubernetes", "cluster":
		if readOnly {
			tools = safeInvokableTools(ctx, buildKubernetesReadonlyTools)
		} else {
			tools = safeInvokableTools(ctx, buildKubernetesTools)
		}
	case "cicd":
		if readOnly {
			tools = safeInvokableTools(ctx, buildCICDReadonlyTools)
		} else {
			tools = safeInvokableTools(ctx, buildCICDTools)
		}
	case "monitoring":
		tools = safeInvokableTools(ctx, buildMonitorReadonlyTools)
	case "host":
		if readOnly {
			tools = safeInvokableTools(ctx, buildHostReadonlyTools)
		} else {
			tools = safeInvokableTools(ctx, buildHostTools)
		}
	case "service":
		if readOnly {
			tools = safeInvokableTools(ctx, buildServiceReadonlyTools)
		} else {
			tools = safeInvokableTools(ctx, buildServiceTools)
		}
	case "deployment":
		tools = safeInvokableTools(ctx, buildDeploymentTools)
	case "infrastructure":
		tools = safeInvokableTools(ctx, buildInfrastructureTools)
	case "governance":
		tools = safeInvokableTools(ctx, buildGovernanceTools)
	default:
		// 默认场景：返回通用只读工具集
		tools = buildDefaultTools(ctx)
	}

	// 转换为 BaseTool 接口
	result := make([]tool.BaseTool, 0, len(tools))
	for _, t := range tools {
		result = append(result, t)
	}
	return result
}

// buildDefaultTools 构建默认工具集（通用只读工具）。
func buildDefaultTools(ctx context.Context) []tool.InvokableTool {
	var tools []tool.InvokableTool

	// 添加各模块的只读工具，避免默认路由暴露变更型能力。
	tools = append(tools, safeInvokableTools(ctx, buildServiceReadonlyTools)...)
	tools = append(tools, safeInvokableTools(ctx, buildMonitorReadonlyTools)...)
	tools = append(tools, safeInvokableTools(ctx, buildKubernetesReadonlyTools)...)
	tools = append(tools, safeInvokableTools(ctx, buildHostReadonlyTools)...)

	return tools
}

func safeInvokableTools(ctx context.Context, builder func(context.Context) []tool.InvokableTool) (tools []tool.InvokableTool) {
	if builder == nil {
		return nil
	}
	defer func() {
		if recover() != nil {
			tools = nil
		}
	}()
	return builder(ctx)
}

func normalizeScene(scene string) string {
	return strings.ToLower(strings.TrimSpace(scene))
}
