// Package tools 提供 AI 编排的工具集合入口。
//
// 本文件汇总所有领域工具，为规划器和执行器提供统一的工具访问入口。
// 工具按领域划分，包括 CI/CD、部署、治理、主机、基础设施、K8s、监控和服务等。
package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/cicd"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
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
//   - 获取资源清单和目录
//   - 检查权限
//   - 搜索审计日志
//
// 参数:
//   - ctx: 上下文
//   - deps: 平台依赖
//
// 返回:
//   - 基础工具列表
func NewCommonTools(ctx context.Context, deps common.PlatformDeps) []tool.BaseTool {
	return []tool.BaseTool{
		cicd.CICDPipelineList(ctx, deps),
		deployment.ClusterListInventory(ctx, deps),
		governance.AuditLogSearch(ctx, deps),
		governance.PermissionCheck(ctx, deps),
		host.HostListInventory(ctx, deps),
		infrastructure.CredentialList(ctx, deps),
		kubernetes.K8sListResources(ctx, deps),
		monitor.MonitorAlertRuleList(ctx, deps),
		service.ServiceCatalogList(ctx, deps),
	}
}
