package cicd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	. "github.com/cy77cc/k8s-manage/internal/ai/tools/core"
	"github.com/cy77cc/k8s-manage/internal/model"
)

type CICDPipelineListOutput struct {
	Total int `json:"total"`
	List  []model.CICDServiceCIConfig
}

func CICDPipelineList(ctx context.Context, deps PlatformDeps, input CICDPipelineListInput) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"cicd_pipeline_list",
		"查询 CI 流水线列表。可选参数 status/keyword/limit。示例: {\"status\":\"active\"}。",
		func(ctx context.Context, input *CICDPipelineListInput, opts ...tool.Option) (*CICDPipelineListOutput, error) {
			if deps.DB == nil {
				return nil, fmt.Errorf("db unavailable")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := deps.DB.Model(&model.CICDServiceCIConfig{})
			if status := strings.TrimSpace(input.Status); status != "" {
				query = query.Where("status = ?", status)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("repo_url LIKE ? OR branch LIKE ?", pattern, pattern)
			}
			var rows []model.CICDServiceCIConfig
			if err := query.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
				return nil, err
			}
			return &CICDPipelineListOutput{
				Total: len(rows),
				List:  rows,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func CICDPipelineStatus(ctx context.Context, deps PlatformDeps, input CICDPipelineStatusInput) (ToolResult, error) {
	return RunWithPolicyAndEvent(
		ctx,
		ToolMeta{
			Name:        "cicd_pipeline_status",
			Description: "查询流水线状态。pipeline_id 必填。示例: {\"pipeline_id\":3}。",
			Mode:        ToolModeReadonly,
			Risk:        ToolRiskLow,
			Provider:    "local",
			Permission:  "ai:tool:read",
			Required:    []string{"pipeline_id"},
			EnumSources: map[string]string{"pipeline_id": "cicd_pipeline_list"},
			SceneScope:  []string{"cicd"},
		},
		input,
		func(in CICDPipelineStatusInput) (any, string, error) {
			if deps.DB == nil {
				return nil, "db", fmt.Errorf("db unavailable")
			}
			if in.PipelineID <= 0 {
				return nil, "validation", NewMissingParam("pipeline_id", "pipeline_id is required")
			}
			var cfg model.CICDServiceCIConfig
			if err := deps.DB.First(&cfg, in.PipelineID).Error; err != nil {
				return nil, "db", err
			}
			var runs []model.CICDServiceCIRun
			_ = deps.DB.Where("ci_config_id = ?", cfg.ID).Order("id desc").Limit(10).Find(&runs).Error
			return map[string]any{"pipeline": cfg, "recent_runs": runs}, "db", nil
		},
	)
}

func CICDPipelineTrigger(ctx context.Context, deps PlatformDeps, input CICDPipelineTriggerInput) (ToolResult, error) {
	return RunWithPolicyAndEvent(
		ctx,
		ToolMeta{
			Name:        "cicd_pipeline_trigger",
			Description: "触发流水线构建。pipeline_id/branch 必填。示例: {\"pipeline_id\":3,\"branch\":\"main\"}。",
			Mode:        ToolModeMutating,
			Risk:        ToolRiskHigh,
			Provider:    "local",
			Permission:  "ai:tool:execute",
			Required:    []string{"pipeline_id", "branch"},
			EnumSources: map[string]string{"pipeline_id": "cicd_pipeline_list"},
			SceneScope:  []string{"cicd"},
		},
		input,
		func(in CICDPipelineTriggerInput) (any, string, error) {
			if deps.DB == nil {
				return nil, "db", fmt.Errorf("db unavailable")
			}
			if in.PipelineID <= 0 {
				return nil, "validation", NewMissingParam("pipeline_id", "pipeline_id is required")
			}
			if strings.TrimSpace(in.Branch) == "" {
				return nil, "validation", NewMissingParam("branch", "branch is required")
			}
			var cfg model.CICDServiceCIConfig
			if err := deps.DB.First(&cfg, in.PipelineID).Error; err != nil {
				return nil, "db", err
			}
			run := model.CICDServiceCIRun{
				ServiceID:   cfg.ServiceID,
				CIConfigID:  cfg.ID,
				TriggerType: "manual",
				Status:      "queued",
				Reason:      "triggered by ai tool",
				TriggeredAt: time.Now(),
			}
			if err := deps.DB.Create(&run).Error; err != nil {
				return nil, "db", err
			}
			return map[string]any{"pipeline_id": cfg.ID, "run_id": run.ID, "branch": strings.TrimSpace(in.Branch), "status": run.Status}, "db", nil
		},
	)
}

func JobList(ctx context.Context, deps PlatformDeps, input JobListInput) (ToolResult, error) {
	return RunWithPolicyAndEvent(
		ctx,
		ToolMeta{
			Name:        "job_list",
			Description: "查询任务列表。可选参数 status/keyword/limit。示例: {\"status\":\"running\"}。",
			Mode:        ToolModeReadonly,
			Risk:        ToolRiskLow,
			Provider:    "local",
			Permission:  "ai:tool:read",
			DefaultHint: map[string]any{"limit": 50},
			SceneScope:  []string{"jobs"},
		},
		input,
		func(in JobListInput) (any, string, error) {
			if deps.DB == nil {
				return nil, "db", fmt.Errorf("db unavailable")
			}
			limit := in.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := deps.DB.Model(&model.Job{})
			if status := strings.TrimSpace(in.Status); status != "" {
				query = query.Where("status = ?", status)
			}
			if kw := strings.TrimSpace(in.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ? OR type LIKE ?", pattern, pattern)
			}
			var jobs []model.Job
			if err := query.Order("id desc").Limit(limit).Find(&jobs).Error; err != nil {
				return nil, "db", err
			}
			return map[string]any{"total": len(jobs), "list": jobs}, "db", nil
		},
	)
}

func JobExecutionStatus(ctx context.Context, deps PlatformDeps, input JobExecutionStatusInput) (ToolResult, error) {
	return RunWithPolicyAndEvent(
		ctx,
		ToolMeta{
			Name:        "job_execution_status",
			Description: "查询任务执行状态。job_id 必填，可选 execution_id。示例: {\"job_id\":12}。",
			Mode:        ToolModeReadonly,
			Risk:        ToolRiskLow,
			Provider:    "local",
			Permission:  "ai:tool:read",
			Required:    []string{"job_id"},
			EnumSources: map[string]string{"job_id": "job_list"},
			SceneScope:  []string{"jobs"},
		},
		input,
		func(in JobExecutionStatusInput) (any, string, error) {
			if deps.DB == nil {
				return nil, "db", fmt.Errorf("db unavailable")
			}
			if in.JobID <= 0 {
				return nil, "validation", NewMissingParam("job_id", "job_id is required")
			}
			query := deps.DB.Model(&model.JobExecution{}).Where("job_id = ?", in.JobID)
			if in.ExecutionID > 0 {
				query = query.Where("id = ?", in.ExecutionID)
			}
			var rows []model.JobExecution
			if err := query.Order("id desc").Limit(20).Find(&rows).Error; err != nil {
				return nil, "db", err
			}
			return map[string]any{"total": len(rows), "list": rows}, "db", nil
		},
	)
}

func JobRun(ctx context.Context, deps PlatformDeps, input JobRunInput) (ToolResult, error) {
	return RunWithPolicyAndEvent(
		ctx,
		ToolMeta{
			Name:        "job_run",
			Description: "手动触发任务执行。job_id 必填。示例: {\"job_id\":12}。",
			Mode:        ToolModeMutating,
			Risk:        ToolRiskMedium,
			Provider:    "local",
			Permission:  "ai:tool:execute",
			Required:    []string{"job_id"},
			EnumSources: map[string]string{"job_id": "job_list"},
			SceneScope:  []string{"jobs"},
		},
		input,
		func(in JobRunInput) (any, string, error) {
			if deps.DB == nil {
				return nil, "db", fmt.Errorf("db unavailable")
			}
			if in.JobID <= 0 {
				return nil, "validation", NewMissingParam("job_id", "job_id is required")
			}
			var job model.Job
			if err := deps.DB.First(&job, in.JobID).Error; err != nil {
				return nil, "db", err
			}
			now := time.Now()
			exec := model.JobExecution{
				JobID:     job.ID,
				Status:    "running",
				ExitCode:  0,
				Output:    "triggered by ai tool",
				StartTime: now,
			}
			if err := deps.DB.Create(&exec).Error; err != nil {
				return nil, "db", err
			}
			_ = deps.DB.Model(&job).Updates(map[string]any{"status": "running", "last_run": now}).Error
			return map[string]any{"job_id": job.ID, "execution_id": exec.ID, "status": exec.Status}, "db", nil
		},
	)
}
