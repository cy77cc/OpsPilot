// Package tools 提供 AI 编排的工具集合入口。
//
// 本文件汇总所有领域工具，为规划器和执行器提供统一的工具访问入口。
// 工具按领域划分，包括 CI/CD、部署、治理、主机、基础设施、K8s、监控和服务等。
package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	approvaltools "github.com/cy77cc/OpsPilot/internal/ai/tools/approval"
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

func NewAllTools(ctx context.Context, deps common.PlatformDeps) []tool.BaseTool {
	base := []tool.BaseTool{
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
	registry := NewRegistry(deps)
	decisionMaker := airuntime.NewApprovalDecisionMaker(airuntime.ApprovalDecisionMakerOptions{
		ResolveScene: airuntime.NewSceneConfigResolver(nil).Resolve,
		LookupTool: func(name string) (airuntime.ApprovalToolSpec, bool) {
			spec, ok := registry.Get(name)
			if !ok {
				return airuntime.ApprovalToolSpec{}, false
			}
			return airuntime.ApprovalToolSpec{
				Name:        spec.Name,
				DisplayName: spec.DisplayName,
				Description: spec.Description,
				Mode:        string(spec.Mode),
				Risk:        string(spec.Risk),
				Category:    spec.Category,
			}, true
		},
	})
	renderer := approvaltools.NewSummaryRenderer()
	out := make([]tool.BaseTool, 0, len(base))
	for _, current := range base {
		invokable, ok := current.(tool.InvokableTool)
		if !ok {
			out = append(out, current)
			continue
		}
		info, err := invokable.Info(ctx)
		if err != nil || info == nil {
			out = append(out, current)
			continue
		}
		spec, ok := registry.Get(info.Name)
		if !ok {
			out = append(out, current)
			continue
		}
		out = append(out, approvaltools.NewGate(invokable, airuntime.ApprovalToolSpec{
			Name:        spec.Name,
			DisplayName: spec.DisplayName,
			Description: spec.Description,
			Mode:        string(spec.Mode),
			Risk:        string(spec.Risk),
			Category:    spec.Category,
		}, decisionMaker, renderer))
	}
	return out
}
