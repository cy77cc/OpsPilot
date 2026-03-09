package kubernetes

import (
	"context"

	. "github.com/cy77cc/k8s-manage/internal/ai/tools/core"
)

// Register 返回 kubernetes 领域的所有工具。
func Register(ctx context.Context, deps PlatformDeps) []RegisteredTool {
	return []RegisteredTool{
		{Meta: ToolMeta{Name: "k8s_query", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "kubernetes"}, Tool: K8sQuery(ctx, deps, K8sQueryInput{})},
		{Meta: ToolMeta{Name: "k8s_list_resources", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "kubernetes"}, Tool: K8sListResources(ctx, deps, K8sListInput{})},
		{Meta: ToolMeta{Name: "k8s_events", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "kubernetes"}, Tool: K8sEvents(ctx, deps, K8sEventsQueryInput{})},
		{Meta: ToolMeta{Name: "k8s_get_events", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "kubernetes"}, Tool: K8sGetEvents(ctx, deps, K8sEventsInput{})},
		{Meta: ToolMeta{Name: "k8s_logs", Mode: ToolModeReadonly, Risk: ToolRiskMedium, Provider: "kubernetes"}, Tool: K8sLogs(ctx, deps, K8sLogsInput{})},
		{Meta: ToolMeta{Name: "k8s_get_pod_logs", Mode: ToolModeReadonly, Risk: ToolRiskMedium, Provider: "kubernetes"}, Tool: K8sGetPodLogs(ctx, deps, K8sPodLogsInput{})},
	}
}
