// Package service 提供服务管理专家实现。
//
// 本文件实现服务管理专家，负责服务状态查询、部署目标管理、配置检查和凭证验证等操作。
// 该专家整合了服务工具、部署工具和基础设施工具。
package service

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	expertspec "github.com/cy77cc/OpsPilot/internal/ai/experts/spec"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	deploymenttools "github.com/cy77cc/OpsPilot/internal/ai/tools/deployment"
	infratools "github.com/cy77cc/OpsPilot/internal/ai/tools/infrastructure"
	servicetools "github.com/cy77cc/OpsPilot/internal/ai/tools/service"
)

// Expert 服务管理专家，处理服务相关的运维任务。
//
// 提供的能力包括：
//   - 服务详情和状态查询
//   - 部署预览和应用（高风险需审批）
//   - 配置项读取和对比
//   - 基础设施凭证管理
type Expert struct {
	deps common.PlatformDeps // 平台依赖
}

// New 创建新的服务管理专家实例。
func New(deps common.PlatformDeps) *Expert {
	return &Expert{deps: deps}
}

// Name 返回专家名称标识。
func (e *Expert) Name() string { return "service" }

// Description 返回专家功能描述。
func (e *Expert) Description() string {
	return "Service expert for service status, deployment targets, config inspection, and credential checks."
}

// Tools 返回专家提供的可调用工具列表。
//
// 整合了三类工具：
//   - 服务工具（排除 service_deploy）
//   - 部署工具（保留 cluster/service inventory 以便先解析目标）
//   - 基础设施工具（全部包含）
func (e *Expert) Tools(ctx context.Context) []tool.InvokableTool {
	out := make([]tool.InvokableTool, 0, 24)
	out = append(out, expertspec.FilterToolsByName(ctx, servicetools.NewServiceTools(ctx, e.deps),
		"service_deploy",
	)...)
	out = append(out, deploymenttools.NewDeploymentTools(ctx, e.deps)...)
	out = append(out, infratools.NewInfrastructureTools(ctx, e.deps)...)
	return out
}

// Capabilities 返回专家的工具能力清单。
//
// 能力覆盖服务管理的各个方面：
//   - 只读操作：服务详情、状态、部署目标、配置项、凭证列表
//   - 中风险操作：部署预览、凭证测试
//   - 高风险操作：部署应用（需审批）
func (e *Expert) Capabilities() []expertspec.ToolCapability {
	return []expertspec.ToolCapability{
		{Name: "cluster_list_inventory", Mode: "readonly", Risk: "low", Description: "List and resolve cluster inventory before service actions."},
		{Name: "service_list_inventory", Mode: "readonly", Risk: "low", Description: "List and resolve service inventory before service actions."},
		{Name: "service_get_detail", Mode: "readonly", Risk: "low", Description: "Fetch full service detail."},
		{Name: "service_status", Mode: "readonly", Risk: "low", Description: "Fetch current service runtime status."},
		{Name: "service_status_by_target", Mode: "readonly", Risk: "low", Description: "Resolve a service target and fetch current service runtime status."},
		{Name: "service_deploy_preview", Mode: "readonly", Risk: "medium", Description: "Preview a service deployment."},
		{Name: "service_deploy_apply", Mode: "mutating", Risk: "high", Description: "Apply a service deployment."},
		{Name: "deployment_target_list", Mode: "readonly", Risk: "low", Description: "List deployment targets."},
		{Name: "config_item_get", Mode: "readonly", Risk: "low", Description: "Read service config items."},
		{Name: "config_diff", Mode: "readonly", Risk: "low", Description: "Compare service config across environments."},
		{Name: "credential_list", Mode: "readonly", Risk: "low", Description: "List infrastructure credentials."},
		{Name: "credential_test", Mode: "readonly", Risk: "medium", Description: "Check credential test status."},
	}
}

// AsTool 将专家导出为工具目录条目，供规划器使用。
func (e *Expert) AsTool() expertspec.ToolExport {
	return expertspec.ToolExport{
		Name:         "service_expert",
		Description:  e.Description(),
		Capabilities: e.Capabilities(),
	}
}
