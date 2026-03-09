package service

import (
	"context"

	. "github.com/cy77cc/k8s-manage/internal/ai/tools/core"
)

// Register 返回 service 领域的所有工具。
func Register(ctx context.Context, deps PlatformDeps) []RegisteredTool {
	return []RegisteredTool{
		{Meta: ToolMeta{Name: "service_get_detail", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "service"}, Tool: ServiceGetDetail(ctx, deps, ServiceDetailInput{})},
		{Meta: ToolMeta{Name: "service_status", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "service"}, Tool: ServiceStatus(ctx, deps, ServiceStatusInput{})},
		{Meta: ToolMeta{Name: "service_deploy_preview", Mode: ToolModeReadonly, Risk: ToolRiskMedium, Provider: "service"}, Tool: ServiceDeployPreview(ctx, deps, ServiceDeployPreviewInput{})},
		{Meta: ToolMeta{Name: "service_deploy_apply", Mode: ToolModeMutating, Risk: ToolRiskHigh, Provider: "service"}, Tool: ServiceDeployApply(ctx, deps, ServiceDeployApplyInput{})},
		{Meta: ToolMeta{Name: "service_deploy", Mode: ToolModeMutating, Risk: ToolRiskHigh, Provider: "service"}, Tool: ServiceDeploy(ctx, deps, ServiceDeployInput{})},
		{Meta: ToolMeta{Name: "service_catalog_list", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "service"}, Tool: ServiceCatalogList(ctx, deps, ServiceCatalogListInput{})},
		{Meta: ToolMeta{Name: "service_category_tree", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "service"}, Tool: ServiceCategoryTree(ctx, deps)},
		{Meta: ToolMeta{Name: "service_visibility_check", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "service"}, Tool: ServiceVisibilityCheck(ctx, deps, ServiceVisibilityCheckInput{})},
	}
}
