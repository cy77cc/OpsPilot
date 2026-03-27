// Package cicd 提供 CI/CD 和作业管理相关的工具实现。
//
// 本文件实现 CI/CD 操作工具集，包括：
//   - 流水线列表查询和状态检查
//   - 流水线触发执行
//   - 作业列表查询和执行状态
//   - 作业手动触发
package cicd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

func serviceContextFromRuntime(ctx context.Context) *svc.ServiceContext {
	svcCtx, _ := runtimectx.ServicesAs[*svc.ServiceContext](ctx)
	return svcCtx
}

// =============================================================================
// 输入类型定义
// =============================================================================

// CICDPipelineListInput 流水线列表查询输入。
type CICDPipelineListInput struct {
	Status  string `json:"status,omitempty" jsonschema_description:"optional status filter"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on repo/branch"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max pipelines,default=50"`
}

// CICDPipelineStatusInput 流水线状态查询输入。
type CICDPipelineStatusInput struct {
	PipelineID int `json:"pipeline_id" jsonschema_description:"required,pipeline config id"`
}

// CICDPipelineTriggerInput 流水线触发输入。
type CICDPipelineTriggerInput struct {
	PipelineID int               `json:"pipeline_id" jsonschema_description:"required,pipeline config id"`
	Branch     string            `json:"branch" jsonschema_description:"required,branch to build"`
	Params     map[string]string `json:"params,omitempty" jsonschema_description:"optional trigger params"`
}

// JobListInput 作业列表查询输入。
type JobListInput struct {
	Status  string `json:"status,omitempty" jsonschema_description:"optional status filter"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on name/type"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max jobs,default=50"`
}

// JobExecutionStatusInput 作业执行状态查询输入。
type JobExecutionStatusInput struct {
	JobID       int `json:"job_id" jsonschema_description:"required,job id"`
	ExecutionID int `json:"execution_id,omitempty" jsonschema_description:"optional execution id"`
}

// JobRunInput 作业触发输入。
type JobRunInput struct {
	JobID  int            `json:"job_id" jsonschema_description:"required,job id"`
	Params map[string]any `json:"params,omitempty" jsonschema_description:"optional run params"`
}

// NewCICDTools 创建所有 CI/CD 工具。
func NewCICDTools(ctx context.Context) []tool.InvokableTool {
	readonly := NewCICDReadonlyTools(ctx)
	write := NewCICDWriteTools(ctx)
	result := make([]tool.InvokableTool, 0, len(readonly)+len(write))
	result = append(result, readonly...)
	result = append(result, write...)
	return result
}

// NewCICDReadonlyTools 创建 CI/CD 只读工具子集。
//
// 返回只读工具列表，包括：
//   - 流水线列表和状态查询（cicd_pipeline_list, cicd_pipeline_status）
//   - 作业列表和执行状态查询（job_list, job_execution_status）
//
// 这些工具不触发任何构建或作业，可安全用于诊断场景。
func NewCICDReadonlyTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		CICDPipelineList(ctx),
		CICDPipelineStatus(ctx),
		JobList(ctx),
		JobExecutionStatus(ctx),
	}
}

// NewCICDWriteTools 创建 CI/CD 写操作工具子集。
//
// 返回写操作工具列表，包括：
//   - 流水线触发（cicd_pipeline_trigger）
//   - 作业手动触发（job_run）
//
// 这些工具会触发构建或作业执行，需要审批机制（Phase 2 实现）。
func NewCICDWriteTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		CICDPipelineTrigger(ctx),
		JobRun(ctx),
	}
}

type CICDPipelineListOutput struct {
	Total int                         `json:"total"`
	List  []model.CICDServiceCIConfig `json:"list"`
}

func CICDPipelineList(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"cicd_pipeline_list",
		"Query CI pipeline configuration list from the CI/CD system. Optional parameters: status filters by pipeline status (active/inactive/queued), keyword searches by repository URL or branch name using fuzzy matching, limit controls max results (default 50, max 200). Returns pipelines with repository info, branch, build configuration, and status. Use pipeline IDs for triggering builds or checking status. Example: {\"status\":\"active\",\"keyword\":\"main\",\"limit\":20}.",
		func(ctx context.Context, input *CICDPipelineListInput, opts ...tool.Option) (*CICDPipelineListOutput, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := svcCtx.DB.Model(&model.CICDServiceCIConfig{})
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

type CICDPipelineStatusOutput struct {
	Pipeline   model.CICDServiceCIConfig `json:"pipeline"`
	RecentRuns []model.CICDServiceCIRun  `json:"recent_runs"`
}

func CICDPipelineStatus(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"cicd_pipeline_status",
		"Query detailed pipeline status including configuration and recent build runs. pipeline_id is required and can be obtained from cicd_pipeline_list. Returns the pipeline configuration (repository URL, branch, build settings) and up to 10 most recent run records with status, duration, and timestamps. Use this to check pipeline health or investigate build failures. Example: {\"pipeline_id\":3}.",
		func(ctx context.Context, input *CICDPipelineStatusInput, opts ...tool.Option) (*CICDPipelineStatusOutput, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.PipelineID <= 0 {
				return nil, fmt.Errorf("pipeline_id is required")
			}
			var cfg model.CICDServiceCIConfig
			if err := svcCtx.DB.First(&cfg, input.PipelineID).Error; err != nil {
				return nil, err
			}
			var runs []model.CICDServiceCIRun
			_ = svcCtx.DB.Where("ci_config_id = ?", cfg.ID).Order("id desc").Limit(10).Find(&runs).Error
			return &CICDPipelineStatusOutput{
				Pipeline:   cfg,
				RecentRuns: runs,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type CICDPipelineTriggerOutput struct {
	PipelineID uint   `json:"pipeline_id"`
	RunID      uint   `json:"run_id"`
	Branch     string `json:"branch"`
	Status     string `json:"status"`
}

func CICDPipelineTrigger(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"cicd_pipeline_trigger",
		"Trigger a new build for a CI/CD pipeline. pipeline_id and branch are required parameters. pipeline_id can be obtained from cicd_pipeline_list. The branch parameter specifies which branch to build (e.g., 'main', 'develop', 'feature/xyz'). Optional params can pass additional build parameters as key-value pairs. This is a mutating operation that queues a new build run. Returns the created run ID and initial status (queued). Example: {\"pipeline_id\":3,\"branch\":\"main\"}.",
		func(ctx context.Context, input *CICDPipelineTriggerInput, opts ...tool.Option) (*CICDPipelineTriggerOutput, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.PipelineID <= 0 {
				return nil, fmt.Errorf("pipeline_id is required")
			}
			if strings.TrimSpace(input.Branch) == "" {
				return nil, fmt.Errorf("branch is required")
			}
			var cfg model.CICDServiceCIConfig
			if err := svcCtx.DB.First(&cfg, input.PipelineID).Error; err != nil {
				return nil, err
			}
			run := model.CICDServiceCIRun{
				ServiceID:   cfg.ServiceID,
				CIConfigID:  cfg.ID,
				TriggerType: "manual",
				Status:      "queued",
				Reason:      "triggered by ai tool",
				TriggeredAt: time.Now(),
			}
			if err := svcCtx.DB.Create(&run).Error; err != nil {
				return nil, err
			}
			return &CICDPipelineTriggerOutput{
				PipelineID: cfg.ID,
				RunID:      run.ID,
				Branch:     strings.TrimSpace(input.Branch),
				Status:     run.Status,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type JobListOutput struct {
	Total int         `json:"total"`
	List  []model.Job `json:"list"`
}

func JobList(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"job_list",
		"Query scheduled job list from the job management system. Optional parameters: status filters by job status (running/scheduled/paused/completed/failed), keyword searches by job name or job type using fuzzy matching, limit controls max results (default 50, max 200). Returns jobs with name, type, schedule (cron expression), next run time, and status. Use job IDs for checking execution status or triggering manual runs. Example: {\"status\":\"running\",\"keyword\":\"backup\"}.",
		func(ctx context.Context, input *JobListInput, opts ...tool.Option) (*JobListOutput, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := svcCtx.DB.Model(&model.Job{})
			if status := strings.TrimSpace(input.Status); status != "" {
				query = query.Where("status = ?", status)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ? OR type LIKE ?", pattern, pattern)
			}
			var jobs []model.Job
			if err := query.Order("id desc").Limit(limit).Find(&jobs).Error; err != nil {
				return nil, err
			}
			return &JobListOutput{
				Total: len(jobs),
				List:  jobs,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type JobExecutionStatusOutput struct {
	Total int                  `json:"total"`
	List  []model.JobExecution `json:"list"`
}

func JobExecutionStatus(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"job_execution_status",
		"Query execution history and status for a specific scheduled job. job_id is required and can be obtained from job_list. Optional execution_id filters to a specific execution run. Returns up to 20 most recent execution records with start/end time, duration, exit code, output logs, and status (running/success/failed). Use this to investigate job failures or monitor long-running jobs. Example: {\"job_id\":12}.",
		func(ctx context.Context, input *JobExecutionStatusInput, opts ...tool.Option) (*JobExecutionStatusOutput, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.JobID <= 0 {
				return nil, fmt.Errorf("job_id is required")
			}
			query := svcCtx.DB.Model(&model.JobExecution{}).Where("job_id = ?", input.JobID)
			if input.ExecutionID > 0 {
				query = query.Where("id = ?", input.ExecutionID)
			}
			var rows []model.JobExecution
			if err := query.Order("id desc").Limit(20).Find(&rows).Error; err != nil {
				return nil, err
			}
			return &JobExecutionStatusOutput{
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

type JobRunOutput struct {
	JobID       uint   `json:"job_id"`
	ExecutionID uint   `json:"execution_id"`
	Status      string `json:"status"`
}

func JobRun(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"job_run",
		"Manually trigger a scheduled job to run immediately, bypassing its normal schedule. job_id is required and can be obtained from job_list. Optional params can override default job parameters as key-value pairs. This is a mutating operation that creates a new execution run with 'running' status. Returns the created execution ID and initial status. Use this for on-demand job execution or testing. Example: {\"job_id\":12}.",
		func(ctx context.Context, input *JobRunInput, opts ...tool.Option) (*JobRunOutput, error) {
			svcCtx := serviceContextFromRuntime(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.JobID <= 0 {
				return nil, fmt.Errorf("job_id is required")
			}
			var job model.Job
			if err := svcCtx.DB.First(&job, input.JobID).Error; err != nil {
				return nil, err
			}
			now := time.Now()
			exec := model.JobExecution{
				JobID:     job.ID,
				Status:    "running",
				ExitCode:  0,
				Output:    "triggered by ai tool",
				StartTime: now,
			}
			if err := svcCtx.DB.Create(&exec).Error; err != nil {
				return nil, err
			}
			_ = svcCtx.DB.Model(&job).Updates(map[string]any{"status": "running", "last_run": now}).Error
			return &JobRunOutput{
				JobID:       job.ID,
				ExecutionID: exec.ID,
				Status:      exec.Status,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}
