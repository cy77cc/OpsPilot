// Package tools 提供 AI 编排的工具集合入口。
//
// 本文件汇总所有领域工具，为规划器和执行器提供统一的工具访问入口。
// 工具按领域划分，包括 CI/CD、部署、治理、主机、基础设施、K8s、监控和服务等。
package tools

import (
	"context"
	"strings"

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
	"github.com/cy77cc/OpsPilot/internal/logger"
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

// NewAllTools 创建完整工具集合，并为注册表中的工具包装审批 Gate。
//
// 流程：
//  1. 创建与 NewCommonTools 相同的基础工具列表
//  2. 通过 ToolRegistry 查找每个工具的元数据（Mode/Risk/Category）
//  3. 对需要审批的工具用 ApprovalGate 包装，触发时自动中断执行并等待人工确认
//  4. 未在注册表中登记的工具保持原样，不添加审批拦截
func NewAllTools(ctx context.Context, deps common.PlatformDeps) []tool.BaseTool {
	base := collectAllTools(ctx, deps)
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
			logToolWrap("skip_non_invokable", "", "", "", false, nil)
			out = append(out, current)
			continue
		}
		info, err := invokable.Info(ctx)
		if err != nil || info == nil {
			logToolWrap("skip_info_unavailable", "", "", "", false, err)
			out = append(out, current)
			continue
		}
		spec, ok := registry.Get(info.Name)
		if !ok {
			mode, risk := inferToolSpec(info.Name)
			if mode != string(ModeMutating) {
				logToolWrap("skip_registry_miss", info.Name, mode, risk, false, nil)
				out = append(out, current)
				continue
			}
			logToolWrap("wrap_registry_fallback", info.Name, mode, risk, true, nil)
			out = append(out, approvaltools.NewGate(invokable, airuntime.ApprovalToolSpec{
				Name:        strings.TrimSpace(info.Name),
				DisplayName: strings.TrimSpace(info.Name),
				Mode:        mode,
				Risk:        risk,
			}, decisionMaker, renderer))
			continue
		}
		logToolWrap("wrap_registry_match", spec.Name, string(spec.Mode), string(spec.Risk), true, nil)
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

func collectAllTools(ctx context.Context, deps common.PlatformDeps) []tool.BaseTool {
	groups := [][]tool.InvokableTool{
		cicd.NewCICDTools(ctx, deps),
		deployment.NewDeploymentTools(ctx, deps),
		governance.NewGovernanceTools(ctx, deps),
		host.NewHostTools(ctx, deps),
		infrastructure.NewInfrastructureTools(ctx, deps),
		kubernetes.NewKubernetesTools(ctx, deps),
		monitor.NewMonitorTools(ctx, deps),
		service.NewServiceTools(ctx, deps),
	}
	var out []tool.BaseTool
	for _, group := range groups {
		for _, item := range group {
			out = append(out, item)
		}
	}
	return out
}

func inferToolSpec(name string) (mode string, risk string) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	mutatingPatterns := []string{"_apply", "_exec", "_delete", "_update", "_create", "_restart", "_scale", "_trigger", "_run"}
	for _, pattern := range mutatingPatterns {
		if strings.Contains(normalized, pattern) {
			if strings.Contains(normalized, "delete") || strings.Contains(normalized, "exec") || strings.Contains(normalized, "restart") {
				return string(ModeMutating), string(RiskHigh)
			}
			return string(ModeMutating), string(RiskMedium)
		}
	}
	return string(ModeReadonly), string(RiskLow)
}

func logToolWrap(action, name, mode, risk string, wrapped bool, err error) {
	l := logger.L()
	if l == nil {
		return
	}
	fields := []logger.Field{
		logger.String("action", action),
		logger.String("tool_name", strings.TrimSpace(name)),
		logger.String("mode", strings.TrimSpace(mode)),
		logger.String("risk", strings.TrimSpace(risk)),
		{Key: "wrapped", Value: wrapped},
	}
	if err != nil {
		fields = append(fields, logger.Error(err))
		l.Warn("ai tool approval wrap evaluation", fields...)
		return
	}
	l.Debug("ai tool approval wrap evaluation", fields...)
}
