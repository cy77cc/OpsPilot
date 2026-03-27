// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件实现审批过期扫描器，负责检测并处理过期的审批任务。
package logic

import (
	"context"
	"fmt"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// approvalExpirerDefaultPollInterval 默认轮询间隔。
const approvalExpirerDefaultPollInterval = 2 * time.Second

// ApprovalExpirer 审批过期扫描器。
//
// 定期扫描待处理的审批任务，检测过期的任务并发布过期事件。
type ApprovalExpirer struct {
	logic *Logic
	now   func() time.Time
}

// NewApprovalExpirer 创建审批过期扫描器实例。
func NewApprovalExpirer(l *Logic) *ApprovalExpirer {
	return &ApprovalExpirer{
		logic: l,
		now:   time.Now,
	}
}

func WithApprovalExpirerClock(now func() time.Time) func(*ApprovalExpirer) {
	return func(expirer *ApprovalExpirer) {
		if now != nil {
			expirer.now = now
		}
	}
}

func (e *ApprovalExpirer) RunLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = approvalExpirerDefaultPollInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		claimed, _ := e.RunOnce(ctx)
		if claimed {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (e *ApprovalExpirer) RunOnce(ctx context.Context) (bool, error) {
	if e == nil || e.logic == nil || e.logic.svcCtx == nil || e.logic.svcCtx.DB == nil || e.logic.ApprovalDAO == nil {
		return false, nil
	}

	now := e.now().UTC()
	tasks, err := e.logic.ApprovalDAO.ListPending(ctx, 200)
	if err != nil {
		return false, err
	}

	expiredAny := false
	for i := range tasks {
		task := tasks[i]
		if task.ExpiresAt == nil || !task.ExpiresAt.Before(now) {
			continue
		}
		if err := e.expireTask(ctx, &task, now); err != nil {
			return expiredAny, err
		}
		expiredAny = true
	}
	return expiredAny, nil
}

func (e *ApprovalExpirer) expireTask(ctx context.Context, snapshot *model.AIApprovalTask, now time.Time) error {
	if snapshot == nil {
		return nil
	}

	return e.logic.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		approvalDAO := aidao.NewAIApprovalTaskDAO(tx)
		outboxDAO := aidao.NewAIApprovalOutboxDAO(tx)

		task, err := approvalDAO.GetByApprovalID(ctx, snapshot.ApprovalID)
		if err != nil {
			return fmt.Errorf("load approval task: %w", err)
		}
		if task == nil || task.Status != "pending" || task.ExpiresAt == nil || !task.ExpiresAt.Before(now) {
			return nil
		}

		result := tx.Model(&model.AIApprovalTask{}).
			Where("approval_id = ? AND status = ?", task.ApprovalID, "pending").
			Updates(map[string]any{
				"status":     "expired",
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		task, err = approvalDAO.GetByApprovalID(ctx, task.ApprovalID)
		if err != nil {
			return fmt.Errorf("reload expired approval task: %w", err)
		}
		if task == nil {
			return fmt.Errorf("approval task not found")
		}

		return NewApprovalWriteModel(tx).writeApprovalEvent(ctx, tx, outboxDAO, task, ApprovalEventTypeExpired, taskStatusExpiredPayload(task))
	})
}
