package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

type Logic struct {
	svcCtx *svc.ServiceContext
}

func NewLogic(svcCtx *svc.ServiceContext) *Logic {
	return &Logic{svcCtx: svcCtx}
}

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

func (l *Logic) GetJob(ctx context.Context, id uint) (*model.Job, error) {
	var job model.Job
	if err := l.svcCtx.DB.WithContext(ctx).First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

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

func (l *Logic) DeleteJob(ctx context.Context, id uint) error {
	return l.svcCtx.DB.WithContext(ctx).Delete(&model.Job{}, id).Error
}

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

// simulateExecution 模拟任务执行 (实际项目中应该由后台 worker 执行)
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
