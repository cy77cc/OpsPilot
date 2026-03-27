// Package logic 提供任务管理的业务逻辑层。
//
// 本文件包含任务管理的核心业务逻辑实现：
//   - ListJobs: 分页查询任务列表
//   - GetJob: 获取单个任务详情
//   - CreateJob: 创建新任务
//   - UpdateJob: 更新任务信息
//   - DeleteJob: 删除任务
//   - StartJob: 启动任务执行
//   - StopJob: 停止任务执行
//   - GetJobExecutions: 查询执行记录
//   - GetJobLogs: 查询执行日志
//   - simulateExecution: 模拟任务执行 (演示用)
package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// Logic 是任务管理的业务逻辑层。
//
// 职责:
//   - 封装任务管理的所有业务逻辑
//   - 提供给 Handler 层调用
//
// 依赖:
//   - svcCtx: 服务上下文，包含数据库连接等资源
type Logic struct {
	svcCtx *svc.ServiceContext
}

// NewLogic 创建任务管理 Logic 实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等资源
//
// 返回: 初始化完成的 Logic 实例
func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{svcCtx: svcCtx}
}

// ListJobs 分页获取任务列表。
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期
//   - page: 页码，从 1 开始
//   - pageSize: 每页数量
//
// 返回:
//   - []model.Job: 任务列表
//   - int64: 总记录数
//   - error: 数据库操作错误
func (l *Logic) ListJobs(ctx context.Context, page, pageSize int) ([]model.Job, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var total int64
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.Job{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var jobs []model.Job
	offset := (page - 1) * pageSize
	if err := l.svcCtx.DB.WithContext(ctx).Order("id desc").Offset(offset).Limit(pageSize).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// GetJob 获取单个任务详情。
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期
//   - id: 任务 ID
//
// 返回:
//   - *model.Job: 任务详情
//   - error: 数据库操作错误 (包括记录不存在)
func (l *Logic) GetJob(ctx context.Context, id uint) (*model.Job, error) {
	var job model.Job
	if err := l.svcCtx.DB.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// CreateJob 创建新任务。
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期
//   - actor: 创建者用户 ID
//   - req: 任务创建请求参数
//
// 返回:
//   - *model.Job: 创建成功的任务
//   - error: 数据库操作错误
//
// 副作用:
//   - 向数据库插入新记录
//   - 设置默认值: Type="shell", Timeout=300
func (l *Logic) CreateJob(ctx context.Context, actor uint, req CreateJobReq) (*model.Job, error) {
	job := model.Job{
		Name:        strings.TrimSpace(req.Name),
		Type:        req.Type,
		Command:     req.Command,
		HostIDs:     req.HostIDs,
		Cron:        req.Cron,
		Status:      "pending",
		Timeout:     req.Timeout,
		Priority:    req.Priority,
		Description: req.Description,
		CreatedBy:   actor,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if job.Type == "" {
		job.Type = "shell"
	}
	if job.Timeout == 0 {
		job.Timeout = 300
	}

	if err := l.svcCtx.DB.WithContext(ctx).Create(&job).Error; err != nil {
		return nil, err
	}

	return &job, nil
}

// UpdateJob 更新任务信息。
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期
//   - id: 任务 ID
//   - req: 任务更新请求参数 (仅更新非零值字段)
//
// 返回:
//   - *model.Job: 更新后的任务详情
//   - error: 数据库操作错误 (包括记录不存在)
//
// 副作用:
//   - 更新数据库记录
//   - 自动更新 updated_at 字段
func (l *Logic) UpdateJob(ctx context.Context, id uint, req UpdateJobReq) (*model.Job, error) {
	var job model.Job
	if err := l.svcCtx.DB.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}

	updates := map[string]any{
		"updated_at": time.Now(),
	}

	if req.Name != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.Command != "" {
		updates["command"] = req.Command
	}
	if req.HostIDs != "" {
		updates["host_ids"] = req.HostIDs
	}
	if req.Cron != "" {
		updates["cron"] = req.Cron
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Timeout > 0 {
		updates["timeout"] = req.Timeout
	}
	if req.Priority != 0 {
		updates["priority"] = req.Priority
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}

	if err := l.svcCtx.DB.WithContext(ctx).Model(&job).Updates(updates).Error; err != nil {
		return nil, err
	}

	return l.GetJob(ctx, id)
}

// DeleteJob 删除任务。
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期
//   - id: 任务 ID
//
// 返回:
//   - error: 数据库操作错误
//
// 副作用:
//   - 从数据库删除记录
func (l *Logic) DeleteJob(ctx context.Context, id uint) error {
	return l.svcCtx.DB.WithContext(ctx).Delete(&model.Job{}, id).Error
}

// StartJob 启动任务执行。
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期
//   - id: 任务 ID
//
// 返回:
//   - error: 数据库操作错误 (包括任务不存在)
//
// 副作用:
//   - 更新任务状态为 running
//   - 创建执行记录
//   - 记录启动日志
//   - 异步启动模拟执行 (实际项目中应由后台 worker 执行)
func (l *Logic) StartJob(ctx context.Context, id uint) error {
	var job model.Job
	if err := l.svcCtx.DB.WithContext(ctx).First(&job, id).Error; err != nil {
		return err
	}

	now := time.Now()
	job.Status = "running"
	job.LastRun = &now
	job.UpdatedAt = now

	if err := l.svcCtx.DB.WithContext(ctx).Save(&job).Error; err != nil {
		return err
	}

	// 创建执行记录
	execution := model.JobExecution{
		JobID:     id,
		Status:    "running",
		StartTime: now,
		CreatedAt: now,
	}
	if err := l.svcCtx.DB.WithContext(ctx).Create(&execution).Error; err != nil {
		return err
	}

	// 记录日志
	_ = l.svcCtx.DB.WithContext(ctx).Create(&model.JobLog{
		JobID:       id,
		ExecutionID: execution.ID,
		Level:       "info",
		Message:     fmt.Sprintf("Job %d started", id),
		CreatedAt:   now,
	}).Error

	// 模拟执行完成 (实际项目中应该由后台任务执行)
	go l.simulateExecution(id, execution.ID)

	return nil
}

// StopJob 停止任务执行。
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期
//   - id: 任务 ID
//
// 返回:
//   - error: 数据库操作错误 (包括任务不存在)
//
// 副作用:
//   - 更新任务状态为 stopped
//   - 记录停止日志
func (l *Logic) StopJob(ctx context.Context, id uint) error {
	var job model.Job
	if err := l.svcCtx.DB.WithContext(ctx).First(&job, id).Error; err != nil {
		return err
	}

	now := time.Now()
	job.Status = "stopped"
	job.UpdatedAt = now

	if err := l.svcCtx.DB.WithContext(ctx).Save(&job).Error; err != nil {
		return err
	}

	// 记录日志
	_ = l.svcCtx.DB.WithContext(ctx).Create(&model.JobLog{
		JobID:     id,
		Level:     "info",
		Message:   fmt.Sprintf("Job %d stopped", id),
		CreatedAt: now,
	}).Error

	return nil
}

// GetJobExecutions 分页获取任务执行记录。
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期
//   - jobID: 任务 ID
//   - page: 页码，从 1 开始
//   - pageSize: 每页数量
//
// 返回:
//   - []model.JobExecution: 执行记录列表
//   - int64: 总记录数
//   - error: 数据库操作错误
func (l *Logic) GetJobExecutions(ctx context.Context, jobID uint, page, pageSize int) ([]model.JobExecution, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var total int64
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.JobExecution{}).Where("job_id = ?", jobID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var executions []model.JobExecution
	offset := (page - 1) * pageSize
	if err := l.svcCtx.DB.WithContext(ctx).Where("job_id = ?", jobID).Order("id desc").Offset(offset).Limit(pageSize).Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// GetJobLogs 分页获取任务执行日志。
//
// 参数:
//   - ctx: 上下文，用于控制请求生命周期
//   - jobID: 任务 ID
//   - page: 页码，从 1 开始
//   - pageSize: 每页数量
//
// 返回:
//   - []model.JobLog: 日志列表
//   - int64: 总记录数
//   - error: 数据库操作错误
func (l *Logic) GetJobLogs(ctx context.Context, jobID uint, page, pageSize int) ([]model.JobLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	var total int64
	if err := l.svcCtx.DB.WithContext(ctx).Model(&model.JobLog{}).Where("job_id = ?", jobID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []model.JobLog
	offset := (page - 1) * pageSize
	if err := l.svcCtx.DB.WithContext(ctx).Where("job_id = ?", jobID).Order("id desc").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// simulateExecution 模拟任务执行。
//
// 这是一个演示用的模拟函数，实际项目中应该由后台 worker 执行真实任务。
//
// 参数:
//   - jobID: 任务 ID
//   - executionID: 执行记录 ID
//
// 副作用:
//   - 等待 2 秒后更新执行状态为 success
//   - 更新任务状态为 success
//   - 记录完成日志
func (l *Logic) simulateExecution(jobID uint, executionID uint) {
	time.Sleep(2 * time.Second)

	ctx := context.Background()
	now := time.Now()

	// 更新执行记录
	l.svcCtx.DB.WithContext(ctx).Model(&model.JobExecution{}).Where("id = ?", executionID).Updates(map[string]any{
		"status":    "success",
		"exit_code": 0,
		"output":    "Task completed successfully",
		"end_time":  now,
	})

	// 更新任务状态
	l.svcCtx.DB.WithContext(ctx).Model(&model.Job{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":     "success",
		"updated_at": now,
	})

	// 记录日志
	l.svcCtx.DB.WithContext(ctx).Create(&model.JobLog{
		JobID:       jobID,
		ExecutionID: executionID,
		Level:       "info",
		Message:     fmt.Sprintf("Job %d completed successfully", jobID),
		CreatedAt:   now,
	})
}
