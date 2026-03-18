// Package tools 提供 AI 编排的工具集合入口。
//
// 本文件汇总所有领域工具，为规划器和执行器提供统一的工具访问入口。
// 工具按领域划分，包括 CI/CD、部署、治理、主机、基础设施、K8s、监控和服务等。
//
// 工具子集说明（按 Agent 用途划分）：
//   - NewDiagnosisTools：只读 K8s + 监控 + 主机诊断工具，供 Diagnosis Agent 使用
//   - NewChangeTools：只读工具 + 写操作工具（Phase 2 接入），供 Change Agent 使用
//   - NewInspectionTools：只读 K8s + 监控 + 主机诊断 + 服务目录，供 Inspection Agent 使用
package tools

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/cicd"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/deployment"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/governance"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/host"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/infrastructure"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/middleware"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/monitor"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/platform"
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
		platform.PlatformDiscoverResources(ctx), // 资源发现工具（优先级最高）
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
// 包含：
//   - Kubernetes 只读工具（query/list/events/logs）
//   - 监控工具（告警查询、指标查询）
//   - 主机诊断工具（CPU/内存、磁盘、网络、进程、日志、容器运行时）
//   - 部署清单工具（集群清单、服务清单）
//
// 不包含任何写操作工具，确保诊断过程不会意外修改集群或主机状态。
//
// 参数:
//   - ctx: 上下文（应携带 common.PlatformDeps）
func NewDiagnosisTools(ctx context.Context) []tool.BaseTool {
	platformTools := []tool.InvokableTool{
		platform.PlatformDiscoverResources(ctx),
	}
	// K8s 只读工具
	k8sTools := kubernetes.NewKubernetesTools(ctx)
	// 监控工具（全部只读）
	monitorTools := monitor.NewMonitorTools(ctx)
	// 主机只读诊断工具
	hostReadonly := host.NewHostReadonlyTools(ctx)
	// 部署清单工具（只读）
	deploymentTools := []tool.InvokableTool{
		deployment.ClusterListInventory(ctx),
		deployment.ServiceListInventory(ctx),
	}

	result := make([]tool.BaseTool, 0, len(platformTools)+len(k8sTools)+len(monitorTools)+len(hostReadonly)+len(deploymentTools))
	for _, t := range platformTools {
		result = append(result, t)
	}
	for _, t := range k8sTools {
		result = append(result, t)
	}
	for _, t := range monitorTools {
		result = append(result, t)
	}
	for _, t := range hostReadonly {
		result = append(result, t)
	}
	for _, t := range deploymentTools {
		result = append(result, t)
	}
	return result
}

// NewChangeTools 创建 Change Agent 工具集（只读工具 + 写操作工具）。
//
// 包含：
//   - Kubernetes 只读工具（预检和验证步骤使用）
//   - 监控工具（变更前后对比）
//   - 主机诊断工具
//   - 资源发现工具
//   - 写操作工具（需审批）
//
// 参数:
//   - ctx: 上下文（应携带 common.PlatformDeps）
func NewChangeTools(ctx context.Context) []tool.BaseTool {
	// 只读工具（诊断 + 资源发现）
	readonly := NewDiagnosisTools(ctx)

	// 写操作工具（需审批中间件）
	writeTools := [][]tool.InvokableTool{
		kubernetes.NewKubernetesWriteTools(ctx),
		host.NewHostWriteTools(ctx),
	}

	result := make([]tool.BaseTool, 0, len(readonly)+len(writeTools)*5)
	result = append(result, readonly...)
	for _, group := range writeTools {
		for _, t := range group {
			result = append(result, t)
		}
	}
	return result
}

// NewInspectionTools 创建 Inspection Agent 工具集。
//
// 包含：
//   - Kubernetes 只读工具（节点/工作负载/资源配额检查）
//   - 监控工具（告警规则查询、Prometheus 指标）
//   - 主机诊断工具（CPU/内存、磁盘、进程、主机清单）
//   - 服务目录工具（服务健康状态概览）
//
// 参数:
//   - ctx: 上下文（应携带 common.PlatformDeps）
func NewInspectionTools(ctx context.Context) []tool.InvokableTool {
	// K8s 只读工具
	k8sTools := kubernetes.NewKubernetesTools(ctx)
	// 监控工具
	monitorTools := monitor.NewMonitorTools(ctx)
	// 主机诊断工具（筛选适合巡检的工具）
	hostTools := []tool.InvokableTool{
		host.OSGetCPUMem(ctx),
		host.OSGetDiskFS(ctx),
		host.OSGetProcessTop(ctx),
		host.HostListInventory(ctx),
	}
	// 服务目录工具
	serviceTools := []tool.InvokableTool{
		service.ServiceCatalogList(ctx),
		service.ServiceCategoryTree(ctx),
		service.ServiceStatus(ctx),
	}

	result := make([]tool.InvokableTool, 0, len(k8sTools)+len(monitorTools)+len(hostTools)+len(serviceTools))
	result = append(result, k8sTools...)
	result = append(result, monitorTools...)
	result = append(result, hostTools...)
	result = append(result, serviceTools...)
	return result
}

func ToolMiddleware(ctx context.Context) (adk.ChatModelAgentMiddleware, error) {
	return toolsearch.New(ctx, &toolsearch.Config{
		DynamicTools: GetAllTools(ctx),
	})
}

// ApprovalMiddleware 创建审批中间件。
//
// 该中间件拦截高风险工具调用，通过 Eino 的 Interrupt/Resume 机制
// 实现 Human-in-the-Loop (HITL) 工作流。
//
// 使用示例:
//
//	mw := tools.ApprovalMiddleware(nil) // 使用默认配置
//	agent := adk.WithMiddleware(baseAgent, mw)
//
// 参数:
//   - cfg: 中间件配置，nil 表示使用默认配置
//
// 返回: 可应用到 Agent 的中间件实例
func ApprovalMiddleware(cfg *middleware.ApprovalMiddlewareConfig) adk.ChatModelAgentMiddleware {
	return middleware.ApprovalMiddleware(cfg)
}

// ApprovalToolMiddleware 创建用于 ToolsConfig 的审批中间件。
//
// 该函数返回一个 compose.ToolMiddleware，可以直接在 ToolsConfig.ToolCallMiddlewares 中使用。
// 适用于 planexecute.Executor 等 Agent 配置。
//
// 使用示例:
//
//	approvalMW := tools.ApprovalToolMiddleware(nil)
//	toolsConfig := adk.ToolsConfig{
//	    ToolsNodeConfig: compose.ToolsNodeConfig{
//	        Tools: toolset,
//	        ToolCallMiddlewares: []compose.ToolMiddleware{approvalMW},
//	    },
//	}
//	executor, _ := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
//	    Model:       model,
//	    ToolsConfig: toolsConfig,
//	})
//
// 参数:
//   - cfg: 中间件配置，nil 表示使用默认配置
//
// 返回: 可添加到 ToolCallMiddlewares 的中间件
func ApprovalToolMiddleware(cfg *middleware.ApprovalMiddlewareConfig) compose.ToolMiddleware {
	return middleware.NewApprovalToolMiddleware(cfg)
}

// ApprovalToolMiddlewares 将多个审批中间件转换为 ToolMiddleware 列表。
//
// 参数:
//   - middlewares: 审批中间件列表
//
// 返回: 可添加到 ToolCallMiddlewares 的中间件列表
func ApprovalToolMiddlewares(middlewares ...adk.ChatModelAgentMiddleware) []compose.ToolMiddleware {
	return middleware.AsToolMiddlewares(middlewares...)
}
