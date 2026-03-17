// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现审批任务的数据访问对象。
package ai

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIApprovalTaskDAO 审批任务数据访问对象。
type AIApprovalTaskDAO struct {
	db *gorm.DB
}

// NewAIApprovalTaskDAO 创建审批任务 DAO 实例。
func NewAIApprovalTaskDAO(db *gorm.DB) *AIApprovalTaskDAO {
	return &AIApprovalTaskDAO{db: db}
}

// Create 创建审批任务。
func (d *AIApprovalTaskDAO) Create(ctx context.Context, task *model.AIApprovalTask) error {
	return d.db.WithContext(ctx).Create(task).Error
}

// GetByApprovalID 根据 ApprovalID 获取审批任务。
func (d *AIApprovalTaskDAO) GetByApprovalID(ctx context.Context, approvalID string) (*model.AIApprovalTask, error) {
	var task model.AIApprovalTask
	err := d.db.WithContext(ctx).
		Where("approval_id = ?", approvalID).
		First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByCheckpointID 根据 CheckpointID 获取审批任务。
func (d *AIApprovalTaskDAO) GetByCheckpointID(ctx context.Context, checkpointID string) (*model.AIApprovalTask, error) {
	var task model.AIApprovalTask
	err := d.db.WithContext(ctx).
		Where("checkpoint_id = ? AND status = ?", checkpointID, "pending").
		First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateStatus 更新审批状态。
func (d *AIApprovalTaskDAO) UpdateStatus(ctx context.Context, approvalID string, status string, approvedBy uint64, reason, comment string) error {
	updates := map[string]any{
		"status":             status,
		"approved_by":        approvedBy,
		"disapprove_reason":  reason,
		"comment":            comment,
		"decided_at":         time.Now(),
		"updated_at":         time.Now(),
	}
	return d.db.WithContext(ctx).
		Model(&model.AIApprovalTask{}).
		Where("approval_id = ? AND status = ?", approvalID, "pending").
		Updates(updates).Error
}

// ListPendingByUserID 列出用户的待处理审批任务。
func (d *AIApprovalTaskDAO) ListPendingByUserID(ctx context.Context, userID uint64, limit int) ([]model.AIApprovalTask, error) {
	var tasks []model.AIApprovalTask
	query := d.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, "pending").
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&tasks).Error
	return tasks, err
}

// ListPending 列出所有待处理审批任务。
func (d *AIApprovalTaskDAO) ListPending(ctx context.Context, limit int) ([]model.AIApprovalTask, error) {
	var tasks []model.AIApprovalTask
	query := d.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&tasks).Error
	return tasks, err
}

// MarkExpired 标记已过期的审批任务。
func (d *AIApprovalTaskDAO) MarkExpired(ctx context.Context) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.AIApprovalTask{}).
		Where("status = ? AND expires_at < ?", "pending", now).
		Updates(map[string]any{
			"status":     "expired",
			"updated_at": now,
		}).Error
}
