package infrastructure

import (
	"context"

	. "github.com/cy77cc/k8s-manage/internal/ai/tools/core"
)

// Register 返回 infrastructure 领域的所有工具。
func Register(ctx context.Context, deps PlatformDeps) []RegisteredTool {
	return []RegisteredTool{
		{Meta: ToolMeta{Name: "credential_list", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "infrastructure"}, Tool: CredentialList(ctx, deps, CredentialListInput{})},
		{Meta: ToolMeta{Name: "credential_test", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "infrastructure"}, Tool: CredentialTest(ctx, deps, CredentialTestInput{})},
	}
}
