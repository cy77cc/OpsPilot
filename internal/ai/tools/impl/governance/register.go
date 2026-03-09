package governance

import (
	"context"

	. "github.com/cy77cc/k8s-manage/internal/ai/tools/core"
)

// Register 返回 governance 领域的所有工具。
func Register(ctx context.Context, deps PlatformDeps) []RegisteredTool {
	return []RegisteredTool{
		{Meta: ToolMeta{Name: "user_list", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "governance"}, Tool: UserList(ctx, deps, UserListInput{})},
		{Meta: ToolMeta{Name: "role_list", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "governance"}, Tool: RoleList(ctx, deps, RoleListInput{})},
		{Meta: ToolMeta{Name: "permission_check", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "governance"}, Tool: PermissionCheck(ctx, deps, PermissionCheckInput{})},
		{Meta: ToolMeta{Name: "topology_get", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "governance"}, Tool: TopologyGet(ctx, deps, TopologyGetInput{})},
		{Meta: ToolMeta{Name: "audit_log_search", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "governance"}, Tool: AuditLogSearch(ctx, deps, AuditLogSearchInput{})},
	}
}
