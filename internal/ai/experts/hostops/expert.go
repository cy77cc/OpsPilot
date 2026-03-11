// Package hostops 提供主机运维专家实现。
//
// 本文件实现主机运维专家，负责主机清单查询、只读诊断和批量执行等操作。
// 该专家通过 SSH 等协议与主机交互，支持受控的运维操作。
package hostops

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	expertspec "github.com/cy77cc/OpsPilot/internal/ai/experts/spec"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	hosttools "github.com/cy77cc/OpsPilot/internal/ai/tools/host"
)

// Expert 主机运维专家，处理主机相关的运维任务。
//
// 提供的能力包括：
//   - 主机清单查询
//   - 只读诊断命令执行（CPU、内存、磁盘等）
//   - 批量操作预览和执行（需要审批）
type Expert struct {
	deps common.PlatformDeps // 平台依赖
}

// New 创建新的主机运维专家实例。
func New(deps common.PlatformDeps) *Expert {
	return &Expert{deps: deps}
}

// Name 返回专家名称标识。
func (e *Expert) Name() string { return "hostops" }

// Description 返回专家功能描述。
func (e *Expert) Description() string {
	return "Host operations expert for host inventory, readonly diagnostics, and governed batch execution."
}

// Tools 返回专家提供的可调用工具列表。
//
// 保留 host_list_inventory，允许专家先做主机清单/解析再决定是否执行诊断命令。
// 仅排除聚合包装工具 host_batch，避免和更明确的 preview/apply 流程重叠。
func (e *Expert) Tools(ctx context.Context) []tool.InvokableTool {
	return expertspec.FilterToolsByName(ctx, hosttools.NewHostTools(ctx, e.deps),
		"host_batch",
	)
}

// Capabilities 返回专家的工具能力清单。
//
// 能力包括：
//   - host_exec: 只读主机命令执行
//   - host_batch_exec_preview: 批量执行预览
//   - host_batch_exec_apply: 批量执行应用（高风险）
//   - host_batch_status_update: 批量状态更新（高风险）
//   - os_get_cpu_mem: CPU/内存诊断
//   - os_get_disk_fs: 磁盘文件系统诊断
func (e *Expert) Capabilities() []expertspec.ToolCapability {
	return []expertspec.ToolCapability{
		{Name: "host_list_inventory", Mode: "readonly", Risk: "low", Description: "List and resolve host inventory before diagnostics."},
		{Name: "host_exec", Mode: "readonly", Risk: "low", Description: "Run readonly host commands."},
		{Name: "host_exec_by_target", Mode: "readonly", Risk: "low", Description: "Resolve a host target and run readonly host commands."},
		{Name: "host_batch_exec_preview", Mode: "readonly", Risk: "medium", Description: "Preview batch host execution."},
		{Name: "host_batch_exec_apply", Mode: "mutating", Risk: "high", Description: "Apply batch host execution."},
		{Name: "host_batch_status_update", Mode: "mutating", Risk: "high", Description: "Change host status in batch."},
		{Name: "os_get_cpu_mem", Mode: "readonly", Risk: "low", Description: "Collect CPU and memory diagnostics."},
		{Name: "os_get_disk_fs", Mode: "readonly", Risk: "low", Description: "Collect disk filesystem diagnostics."},
	}
}

// AsTool 将专家导出为工具目录条目，供规划器使用。
func (e *Expert) AsTool() expertspec.ToolExport {
	return expertspec.ToolExport{
		Name:         "hostops_expert",
		Description:  e.Description(),
		Capabilities: e.Capabilities(),
	}
}
