// Package tools 提供 AI 编排的工具集合入口。
//
// 本文件汇总所有领域工具，为规划器和执行器提供统一的工具访问入口。
// 工具按领域划分，包括 CI/CD、部署、治理、主机、基础设施、K8s、监控和服务等。
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

// NewAllTools 创建完整工具集合。
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


func ToolMiddleware(ctx context.Context) (adk.ChatModelAgentMiddleware, error) {
	return toolsearch.New(ctx, &toolsearch.Config{
		DynamicTools: GetAllTools(ctx),
	})
}
