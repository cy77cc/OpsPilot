// Package tools 提供 AI 编排的工具集合入口。
//
// 本文件汇总所有领域工具，为规划器和执行器提供统一的工具访问入口。
// 工具按领域划分，包括 CI/CD、部署、治理、主机、基础设施、K8s、监控和服务等。
//
// 工具子集说明（按 Agent 用途划分）：
//   - NewDiagnosisTools：只读 K8s 工具 + 监控工具，供 Diagnosis Agent 使用
//   - NewChangeTools：只读工具 + 写操作工具（Phase 2 接入），供 Change Agent 使用
//   - NewInspectionTools：只读 K8s + 监控 + 服务目录，供 Inspection Agent 使用
package tools

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/cicd"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/deployment"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/governance"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/host"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/infrastructure"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/monitor"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/service"
)

// NewCommonTools 创建通用工具集合。
//
// 返回规划阶段常用的基础工具列表，用于：
//   - 检查权限
//   - 搜索审计日志
//
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - 基础工具列表
func NewCommonTools(ctx context.Context) []tool.BaseTool {
	return []tool.BaseTool{
		cicd.CICDPipelineList(ctx),
		deployment.ClusterListInventory(ctx),
		governance.AuditLogSearch(ctx),
		governance.PermissionCheck(ctx),
		host.HostListInventory(ctx),
		infrastructure.CredentialList(ctx),
		kubernetes.K8sListResources(ctx),
		monitor.MonitorAlertRuleList(ctx),
		service.ServiceCatalogList(ctx),
	}
}

// GetAllTools 创建完整工具集合。
//
// 返回执行阶段可用的全部领域工具，包含：
//   - CI/CD
//   - 部署
//   - 治理
//   - 主机
//   - 基础设施
//   - Kubernetes
//   - 监控
//   - 服务
func GetAllTools(ctx context.Context) []tool.BaseTool {
	groups := [][]tool.InvokableTool{
		cicd.NewCICDTools(ctx),
		deployment.NewDeploymentTools(ctx),
		governance.NewGovernanceTools(ctx),
		host.NewHostTools(ctx),
		infrastructure.NewInfrastructureTools(ctx),
		kubernetes.NewKubernetesTools(ctx),
		monitor.NewMonitorTools(ctx),
		service.NewServiceTools(ctx),
	}

	allTool := make([]tool.BaseTool, 0, 64)
	for _, group := range groups {
		for _, item := range group {
			allTool = append(allTool, item)
		}
	}
	return allTool
}

// NewDiagnosisTools 创建 Diagnosis Agent 专用只读工具集。
//
// 仅包含：
//   - Kubernetes 只读工具（query/list/events/logs）
//   - 监控工具（告警查询、指标查询）
//
// 不包含任何写操作工具，确保诊断过程不会意外修改集群状态。
//
// 参数:
//   - ctx: 上下文（应携带 common.PlatformDeps）
func NewDiagnosisTools(ctx context.Context) []tool.BaseTool {
	k8sTools := kubernetes.NewKubernetesTools(ctx)
	monitorTools := monitor.NewMonitorTools(ctx)

	result := make([]tool.BaseTool, 0, len(k8sTools)+len(monitorTools))
	for _, t := range k8sTools {
		result = append(result, t)
	}
	for _, t := range monitorTools {
		result = append(result, t)
	}
	return result
}

// NewChangeTools 创建 Change Agent 工具集（只读工具 + 写操作工具）。
//
// 包含：
//   - Kubernetes 只读工具（预检和验证步骤使用）
//   - 监控工具（变更前后对比）
//   - Kubernetes 写操作工具（Phase 2 实现后在此接入，每个写工具内置 approvalGate）
//
// Phase 1：与 NewDiagnosisTools 等同，写操作工具待 Phase 2 接入。
// Phase 2：在此函数中追加 kubernetes.NewKubernetesWriteTools(ctx) 即可，无需修改 Change Agent。
//
// 参数:
//   - ctx: 上下文（应携带 common.PlatformDeps）
func NewChangeTools(ctx context.Context) []tool.BaseTool {
	k8sTools := kubernetes.NewKubernetesTools(ctx)
	monitorTools := monitor.NewMonitorTools(ctx)

	result := make([]tool.BaseTool, 0, len(k8sTools)+len(monitorTools))
	for _, t := range k8sTools {
		result = append(result, t)
	}
	for _, t := range monitorTools {
		result = append(result, t)
	}
	// Phase 2: 追加写操作工具（含 approvalGate 包装）
	// writeTools := kubernetes.NewKubernetesWriteTools(ctx)
	// for _, t := range writeTools {
	// 	result = append(result, t)
	// }
	return result
}

// NewInspectionTools 创建 Inspection Agent 工具集。
//
// 包含：
//   - Kubernetes 只读工具（节点/工作负载/资源配额检查）
//   - 监控工具（告警规则查询、Prometheus 指标）
//   - 服务目录工具（服务健康状态概览）
//
// 参数:
//   - ctx: 上下文（应携带 common.PlatformDeps）
func NewInspectionTools(ctx context.Context) []tool.InvokableTool {
	k8sTools := kubernetes.NewKubernetesTools(ctx)
	monitorTools := monitor.NewMonitorTools(ctx)
	serviceTools := service.NewServiceTools(ctx)

	result := make([]tool.InvokableTool, 0, len(k8sTools)+len(monitorTools)+len(serviceTools))
	result = append(result, k8sTools...)
	result = append(result, monitorTools...)
	result = append(result, serviceTools...)
	return result
}

func ToolMiddleware(ctx context.Context) (adk.ChatModelAgentMiddleware, error) {
	return toolsearch.New(ctx, &toolsearch.Config{
		DynamicTools: GetAllTools(ctx),
	})
}
