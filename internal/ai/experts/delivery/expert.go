// Package delivery 提供交付专家实现。
//
// 本文件实现交付专家，负责 CI/CD 流水线管理、发布作业和自动化任务状态查询等操作。
// 该专家支持流水线的只读查询和触发执行（高风险需审批）。
package delivery

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	expertspec "github.com/cy77cc/OpsPilot/internal/ai/experts/spec"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/cicd"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

// Expert 交付专家，处理 CI/CD 和自动化相关的运维任务。
//
// 提供的能力包括：
//   - 流水线配置和状态查询
//   - 流水线触发执行（高风险需审批）
//   - 平台作业列表和状态查询
//   - 作业触发执行（高风险需审批）
type Expert struct {
	deps common.PlatformDeps // 平台依赖
}

// New 创建新的交付专家实例。
func New(deps common.PlatformDeps) *Expert {
	return &Expert{deps: deps}
}

// Name 返回专家名称标识。
func (e *Expert) Name() string { return "delivery" }

// Description 返回专家功能描述。
func (e *Expert) Description() string {
	return "Delivery expert for CI/CD pipelines, release jobs, and automation run status."
}

// Tools 返回专家提供的可调用工具列表。
func (e *Expert) Tools(ctx context.Context) []tool.InvokableTool {
	return cicd.NewCICDTools(ctx, e.deps)
}

// Capabilities 返回专家的工具能力清单。
//
// 能力覆盖 CI/CD 和自动化：
//   - 只读操作：流水线列表、状态查询、作业列表、执行状态
//   - 高风险操作：流水线触发、作业执行（需审批）
func (e *Expert) Capabilities() []expertspec.ToolCapability {
	return []expertspec.ToolCapability{
		{Name: "cicd_pipeline_list", Mode: "readonly", Risk: "low", Description: "List pipeline configs."},
		{Name: "cicd_pipeline_status", Mode: "readonly", Risk: "low", Description: "Inspect pipeline status and runs."},
		{Name: "cicd_pipeline_trigger", Mode: "mutating", Risk: "high", Description: "Trigger a pipeline run."},
		{Name: "job_list", Mode: "readonly", Risk: "low", Description: "List platform jobs."},
		{Name: "job_execution_status", Mode: "readonly", Risk: "low", Description: "Inspect job execution status."},
		{Name: "job_run", Mode: "mutating", Risk: "high", Description: "Trigger a job run."},
	}
}

// AsTool 将专家导出为工具目录条目，供规划器使用。
func (e *Expert) AsTool() expertspec.ToolExport {
	return expertspec.ToolExport{
		Name:         "delivery_expert",
		Description:  e.Description(),
		Capabilities: e.Capabilities(),
	}
}
