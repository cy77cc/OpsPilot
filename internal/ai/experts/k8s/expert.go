// Package k8s 提供 Kubernetes 专家实现。
//
// 本文件实现 Kubernetes 集群专家，负责集群工作负载查询、事件检查和 Pod 日志查看等操作。
// 该专家通过 Kubernetes API 与集群交互，支持只读的集群状态查询。
package k8s

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	expertspec "github.com/cy77cc/OpsPilot/internal/ai/experts/spec"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	k8stools "github.com/cy77cc/OpsPilot/internal/ai/tools/kubernetes"
)

// Expert Kubernetes 集群专家，处理 K8s 相关的运维任务。
//
// 提供的能力包括：
//   - 集群资源查询（Pod、Deployment、Service 等）
//   - 事件检查和过滤
//   - Pod/容器日志获取
type Expert struct {
	deps common.PlatformDeps // 平台依赖
}

// New 创建新的 Kubernetes 专家实例。
func New(deps common.PlatformDeps) *Expert {
	return &Expert{deps: deps}
}

// Name 返回专家名称标识。
func (e *Expert) Name() string { return "k8s" }

// Description 返回专家功能描述。
func (e *Expert) Description() string {
	return "Kubernetes expert for cluster workload queries, events, and pod log inspection."
}

// Tools 返回专家提供的可调用工具列表。
func (e *Expert) Tools(ctx context.Context) []tool.InvokableTool {
	return k8stools.NewKubernetesTools(ctx, e.deps)
}

// Capabilities 返回专家的工具能力清单。
//
// 所有能力均为只读模式、低风险等级：
//   - k8s_query: 查询 Kubernetes 资源
//   - k8s_list_resources: 按类型列出资源
//   - k8s_events: 检查集群事件
//   - k8s_get_events: 检查过滤后的事件
//   - k8s_logs: 获取 Pod/容器日志
//   - k8s_get_pod_logs: 直接获取 Pod 日志
func (e *Expert) Capabilities() []expertspec.ToolCapability {
	return []expertspec.ToolCapability{
		{Name: "k8s_query", Mode: "readonly", Risk: "low", Description: "Query Kubernetes resources."},
		{Name: "k8s_list_resources", Mode: "readonly", Risk: "low", Description: "List Kubernetes resources by type."},
		{Name: "k8s_events", Mode: "readonly", Risk: "low", Description: "Inspect Kubernetes events."},
		{Name: "k8s_get_events", Mode: "readonly", Risk: "low", Description: "Inspect filtered Kubernetes events."},
		{Name: "k8s_logs", Mode: "readonly", Risk: "low", Description: "Fetch pod/container logs."},
		{Name: "k8s_get_pod_logs", Mode: "readonly", Risk: "low", Description: "Fetch pod logs directly."},
	}
}

// AsTool 将专家导出为工具目录条目，供规划器使用。
func (e *Expert) AsTool() expertspec.ToolExport {
	return expertspec.ToolExport{
		Name:         "k8s_expert",
		Description:  e.Description(),
		Capabilities: e.Capabilities(),
	}
}
