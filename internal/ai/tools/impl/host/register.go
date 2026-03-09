package host

import (
	"context"

	. "github.com/cy77cc/k8s-manage/internal/ai/tools/core"
)

// Register 返回 host 领域的所有工具。
func Register(ctx context.Context, deps PlatformDeps) []RegisteredTool {
	return []RegisteredTool{
		// OS 工具
		{Meta: ToolMeta{Name: "os_get_cpu_mem", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "host"}, Tool: OSGetCPUMem(ctx, deps, OSCPUMemInput{})},
		{Meta: ToolMeta{Name: "os_get_disk_fs", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "host"}, Tool: OSGetDiskFS(ctx, deps, OSDiskInput{})},
		{Meta: ToolMeta{Name: "os_get_net_stat", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "host"}, Tool: OSGetNetStat(ctx, deps, OSNetInput{})},
		{Meta: ToolMeta{Name: "os_get_process_top", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "host"}, Tool: OSGetProcessTop(ctx, deps, OSProcessTopInput{})},
		{Meta: ToolMeta{Name: "os_get_journal_tail", Mode: ToolModeReadonly, Risk: ToolRiskMedium, Provider: "host"}, Tool: OSGetJournalTail(ctx, deps, OSJournalInput{})},
		{Meta: ToolMeta{Name: "os_get_container_runtime", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "host"}, Tool: OSGetContainerRuntime(ctx, deps, OSContainerRuntimeInput{})},
		// Host 工具
		{Meta: ToolMeta{Name: "host_ssh_exec_readonly", Mode: ToolModeReadonly, Risk: ToolRiskMedium, Provider: "host"}, Tool: HostSSHReadonly(ctx, deps, HostSSHReadonlyInput{})},
		{Meta: ToolMeta{Name: "host_exec", Mode: ToolModeReadonly, Risk: ToolRiskMedium, Provider: "host"}, Tool: HostExec(ctx, deps, HostExecInput{})},
		{Meta: ToolMeta{Name: "host_list_inventory", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "host"}, Tool: HostListInventory(ctx, deps, HostInventoryInput{})},
		{Meta: ToolMeta{Name: "host_batch", Mode: ToolModeMutating, Risk: ToolRiskHigh, Provider: "host"}, Tool: HostBatch(ctx, deps, HostBatchInput{})},
		{Meta: ToolMeta{Name: "host_batch_exec_preview", Mode: ToolModeReadonly, Risk: ToolRiskMedium, Provider: "host"}, Tool: HostBatchExecPreview(ctx, deps, HostBatchExecPreviewInput{})},
		{Meta: ToolMeta{Name: "host_batch_exec_apply", Mode: ToolModeMutating, Risk: ToolRiskHigh, Provider: "host"}, Tool: HostBatchExecApply(ctx, deps, HostBatchExecApplyInput{})},
		{Meta: ToolMeta{Name: "host_batch_status_update", Mode: ToolModeMutating, Risk: ToolRiskMedium, Provider: "host"}, Tool: HostBatchStatusUpdate(ctx, deps, HostBatchStatusInput{})},
	}
}
