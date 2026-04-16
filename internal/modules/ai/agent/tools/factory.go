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

func BuildToolsForScene(ctx context.Context, scene string) []tool.BaseTool {
	return BuildToolsForSceneWithMode(ctx, scene, false)
}

// BuildToolsForSceneWithMode 根据场景和访问模式构建工具列表。
func BuildToolsForSceneWithMode(ctx context.Context, scene string, readOnly bool) []tool.BaseTool {
	var tools []tool.InvokableTool

	switch normalizeScene(scene) {
	case "kubernetes", "cluster":
		if readOnly {
			tools = kubernetes.NewKubernetesReadonlyTools(ctx)
		} else {
			tools = kubernetes.NewKubernetesTools(ctx)
		}
	case "cicd":
		if readOnly {
			tools = cicd.NewCICDReadonlyTools(ctx)
		} else {
			tools = cicd.NewCICDTools(ctx)
		}
	case "monitoring":
		tools = monitor.NewMonitorReadonlyTools(ctx)
	case "host":
		if readOnly {
			tools = host.NewHostReadonlyTools(ctx)
		} else {
			tools = host.NewHostTools(ctx)
		}
	case "service":
		if readOnly {
			tools = service.NewServiceReadonlyTools(ctx)
		} else {
			tools = service.NewServiceTools(ctx)
		}
	case "deployment":
		tools = deployment.NewDeploymentTools(ctx)
	case "infrastructure":
		tools = infrastructure.NewInfrastructureTools(ctx)
	case "governance":
		tools = governance.NewGovernanceTools(ctx)
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
	tools = append(tools, service.NewServiceReadonlyTools(ctx)...)
	tools = append(tools, monitor.NewMonitorReadonlyTools(ctx)...)
	tools = append(tools, kubernetes.NewKubernetesReadonlyTools(ctx)...)
	tools = append(tools, host.NewHostReadonlyTools(ctx)...)

	return tools
}

func normalizeScene(scene string) string {
	return strings.ToLower(strings.TrimSpace(scene))
}
