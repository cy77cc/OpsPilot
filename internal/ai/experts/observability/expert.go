// Package observability 提供可观测性专家实现。
//
// 本文件实现可观测性专家，负责告警查询、指标分析、拓扑检查和审计日志搜索等操作。
// 该专家整合了监控工具和治理工具，提供全面的可观测性支持。
package observability

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	expertspec "github.com/cy77cc/OpsPilot/internal/ai/experts/spec"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	governancetools "github.com/cy77cc/OpsPilot/internal/ai/tools/governance"
	monitortools "github.com/cy77cc/OpsPilot/internal/ai/tools/monitor"
)

// Expert 可观测性专家，处理监控和审计相关的运维任务。
//
// 提供的能力包括：
//   - 告警规则列表和告警状态查询
//   - 指标时间序列查询
//   - 服务拓扑关系检查
//   - 审计日志搜索
type Expert struct {
	deps common.PlatformDeps // 平台依赖
}

// New 创建新的可观测性专家实例。
func New(deps common.PlatformDeps) *Expert {
	return &Expert{deps: deps}
}

// Name 返回专家名称标识。
func (e *Expert) Name() string { return "observability" }

// Description 返回专家功能描述。
func (e *Expert) Description() string {
	return "Observability expert for alerts, metrics, topology, and audit evidence."
}

// Tools 返回专家提供的可调用工具列表。
//
// 整合了两类工具：
//   - 监控工具：告警和指标查询
//   - 治理工具：拓扑获取和审计日志搜索
func (e *Expert) Tools(ctx context.Context) []tool.InvokableTool {
	out := make([]tool.InvokableTool, 0, 8)
	out = append(out, monitortools.NewMonitorTools(ctx, e.deps)...)
	out = append(out, governancetools.TopologyGet(ctx, e.deps))
	out = append(out, governancetools.AuditLogSearch(ctx, e.deps))
	return out
}

// Capabilities 返回专家的工具能力清单。
//
// 所有能力均为只读模式、低风险等级：
//   - 告警相关：规则列表、告警检查、活跃告警
//   - 指标相关：时间序列查询、表达式查询
//   - 治理相关：拓扑检查、审计日志搜索
func (e *Expert) Capabilities() []expertspec.ToolCapability {
	return []expertspec.ToolCapability{
		{Name: "monitor_alert_rule_list", Mode: "readonly", Risk: "low", Description: "List alerting rules."},
		{Name: "monitor_alert", Mode: "readonly", Risk: "low", Description: "Inspect firing alerts."},
		{Name: "monitor_alert_active", Mode: "readonly", Risk: "low", Description: "Inspect active alerts."},
		{Name: "monitor_metric", Mode: "readonly", Risk: "low", Description: "Query metric time series."},
		{Name: "monitor_metric_query", Mode: "readonly", Risk: "low", Description: "Query metric expressions."},
		{Name: "topology_get", Mode: "readonly", Risk: "low", Description: "Inspect service topology."},
		{Name: "audit_log_search", Mode: "readonly", Risk: "low", Description: "Search audit evidence."},
	}
}

// AsTool 将专家导出为工具目录条目，供规划器使用。
func (e *Expert) AsTool() expertspec.ToolExport {
	return expertspec.ToolExport{
		Name:         "observability_expert",
		Description:  e.Description(),
		Capabilities: e.Capabilities(),
	}
}
