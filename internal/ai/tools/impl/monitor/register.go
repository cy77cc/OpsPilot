package monitor

import (
	"context"

	. "github.com/cy77cc/k8s-manage/internal/ai/tools/core"
)

// Register 返回 monitor 领域的所有工具。
func Register(ctx context.Context, deps PlatformDeps) []RegisteredTool {
	return []RegisteredTool{
		{Meta: ToolMeta{Name: "monitor_alert_rule_list", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "monitor"}, Tool: MonitorAlertRuleList(ctx, deps, MonitorAlertRuleListInput{})},
		{Meta: ToolMeta{Name: "monitor_alert", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "monitor"}, Tool: MonitorAlert(ctx, deps, MonitorAlertInput{})},
		{Meta: ToolMeta{Name: "monitor_alert_active", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "monitor"}, Tool: MonitorAlertActive(ctx, deps, MonitorAlertActiveInput{})},
		{Meta: ToolMeta{Name: "monitor_metric", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "monitor"}, Tool: MonitorMetric(ctx, deps, MonitorMetricInput{})},
		{Meta: ToolMeta{Name: "monitor_metric_query", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "monitor"}, Tool: MonitorMetricQuery(ctx, deps, MonitorMetricQueryInput{})},
	}
}
