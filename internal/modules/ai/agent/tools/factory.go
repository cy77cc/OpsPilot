package tools

import (
	"context"

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

// BuildToolsForScene 根据场景构建工具列表。
//
// 该函数根据场景名称返回对应模块的工具实例。
// 如果场景不匹配任何已知模块，返回通用工具集。
func BuildToolsForScene(ctx context.Context, scene string) []tool.BaseTool {
	var tools []tool.InvokableTool

	switch scene {
	case "kubernetes", "cluster":
		tools = kubernetes.NewKubernetesTools(ctx)
	case "cicd":
		tools = cicd.NewCICDTools(ctx)
	case "monitoring":
		tools = monitor.NewMonitorTools(ctx)
	case "host":
		tools = host.NewHostTools(ctx)
	case "service":
		tools = service.NewServiceTools(ctx)
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

	// 添加各模块的工具（优先使用只读版本，如果不存在则使用完整版本）
	tools = append(tools, service.NewServiceReadonlyTools(ctx)...)
	tools = append(tools, monitor.NewMonitorReadonlyTools(ctx)...)
	tools = append(tools, kubernetes.NewKubernetesReadonlyTools(ctx)...)
	// host 包没有只读版本，使用完整版本
	tools = append(tools, host.NewHostTools(ctx)...)

	return tools
}
