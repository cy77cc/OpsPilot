package cicd

import (
	"context"

	. "github.com/cy77cc/k8s-manage/internal/ai/tools/core"
)

// Register 返回 cicd 领域的所有工具。
func Register(ctx context.Context, deps PlatformDeps) []RegisteredTool {
	return []RegisteredTool{
		{Meta: ToolMeta{Name: "cicd_pipeline_list", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "cicd"}, Tool: CICDPipelineList(ctx, deps, CICDPipelineListInput{})},
		{Meta: ToolMeta{Name: "cicd_pipeline_status", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "cicd"}, Tool: CICDPipelineStatus(ctx, deps, CICDPipelineStatusInput{})},
		{Meta: ToolMeta{Name: "cicd_pipeline_trigger", Mode: ToolModeMutating, Risk: ToolRiskHigh, Provider: "cicd"}, Tool: CICDPipelineTrigger(ctx, deps, CICDPipelineTriggerInput{})},
		{Meta: ToolMeta{Name: "job_list", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "cicd"}, Tool: JobList(ctx, deps, JobListInput{})},
		{Meta: ToolMeta{Name: "job_execution_status", Mode: ToolModeReadonly, Risk: ToolRiskLow, Provider: "cicd"}, Tool: JobExecutionStatus(ctx, deps, JobExecutionStatusInput{})},
		{Meta: ToolMeta{Name: "job_run", Mode: ToolModeMutating, Risk: ToolRiskMedium, Provider: "cicd"}, Tool: JobRun(ctx, deps, JobRunInput{})},
	}
}
