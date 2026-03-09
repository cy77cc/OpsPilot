package deployment

import (
	"context"

	. "github.com/cy77cc/k8s-manage/internal/ai/tools/core"
)

// Register 返回 deployment 领域的所有工具。
func Register(ctx context.Context, deps PlatformDeps) []RegisteredTool {
	return []RegisteredTool{
		{Meta: ToolMeta{Name: "deployment_target_list", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "deployment"}, Tool: DeploymentTargetList(ctx, deps, DeploymentTargetListInput{})},
		{Meta: ToolMeta{Name: "deployment_target_detail", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "deployment"}, Tool: DeploymentTargetDetail(ctx, deps, DeploymentTargetDetailInput{})},
		{Meta: ToolMeta{Name: "deployment_bootstrap_status", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "deployment"}, Tool: DeploymentBootstrapStatus(ctx, deps, DeploymentBootstrapStatusInput{})},
		{Meta: ToolMeta{Name: "config_app_list", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "deployment"}, Tool: ConfigAppList(ctx, deps, ConfigAppListInput{})},
		{Meta: ToolMeta{Name: "config_item_get", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "deployment"}, Tool: ConfigItemGet(ctx, deps, ConfigItemGetInput{})},
		{Meta: ToolMeta{Name: "config_diff", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "deployment"}, Tool: ConfigDiff(ctx, deps, ConfigDiffInput{})},
		{Meta: ToolMeta{Name: "cluster_list_inventory", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "deployment"}, Tool: ClusterListInventory(ctx, deps, ClusterInventoryInput{})},
		{Meta: ToolMeta{Name: "service_list_inventory", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "deployment"}, Tool: ServiceListInventory(ctx, deps, ServiceInventoryInput{})},
	}
}
