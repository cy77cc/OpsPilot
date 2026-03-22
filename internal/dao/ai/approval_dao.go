// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现审批任务的数据访问对象。
package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		"status":            status,
		"approved_by":       approvedBy,
		"disapprove_reason": reason,
		"comment":           comment,
		"decided_at":        time.Now(),
		"updated_at":        time.Now(),
	}
	return d.db.WithContext(ctx).
		Model(&model.AIApprovalTask{}).
		Where("approval_id = ? AND status = ?", approvalID, "pending").
		Updates(updates).Error
}

// ApproveWithLease atomically approves a pending task and sets the decision lease.
func (d *AIApprovalTaskDAO) ApproveWithLease(ctx context.Context, approvalID string, approvedBy uint64, comment string, leaseWindow time.Duration) (*model.AIApprovalTask, error) {
	if approvalID == "" {
		return nil, fmt.Errorf("approval id is required")
	}

	now := time.Now()
	var updated *model.AIApprovalTask
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.AIApprovalTask
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("approval_id = ?", approvalID).
			First(&task).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if task.Status != "pending" {
			updated = &task
			return nil
		}
		if task.ExpiresAt != nil && task.ExpiresAt.Before(now) {
			expiredAt := now
			res := tx.Model(&model.AIApprovalTask{}).
				Where("approval_id = ? AND status = ?", approvalID, "pending").
				Updates(map[string]any{
					"status":            "expired",
					"approved_by":       approvedBy,
					"comment":           comment,
					"disapprove_reason": "",
					"decided_at":        expiredAt,
					"lock_expires_at":   nil,
					"updated_at":        expiredAt,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return nil
			}
			task.Status = "expired"
			task.ApprovedBy = approvedBy
			task.Comment = comment
			task.DisapproveReason = ""
			task.DecidedAt = &expiredAt
			task.LockExpiresAt = nil
			task.UpdatedAt = expiredAt
			updated = &task
			return nil
		}

		leaseExpires := now.Add(leaseWindow)
		decisionTime := now
		res := tx.Model(&model.AIApprovalTask{}).
			Where("approval_id = ? AND status = ?", approvalID, "pending").
			Updates(map[string]any{
				"status":            "approved",
				"approved_by":       approvedBy,
				"comment":           comment,
				"disapprove_reason": "",
				"decided_at":        decisionTime,
				"lock_expires_at":   leaseExpires,
				"updated_at":        decisionTime,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		task.Status = "approved"
		task.ApprovedBy = approvedBy
		task.Comment = comment
		task.DisapproveReason = ""
		task.DecidedAt = &decisionTime
		task.LockExpiresAt = &leaseExpires
		task.UpdatedAt = decisionTime
		updated = &task
		return nil
	})
	return updated, err
}

// RejectPending rejects a pending approval task without mutating locked tasks.
func (d *AIApprovalTaskDAO) RejectPending(ctx context.Context, approvalID string, rejectedBy uint64, reason, comment string) (*model.AIApprovalTask, error) {
	if approvalID == "" {
		return nil, fmt.Errorf("approval id is required")
	}

	now := time.Now()
	var updated *model.AIApprovalTask
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.AIApprovalTask
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("approval_id = ?", approvalID).
			First(&task).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if task.Status != "pending" {
			updated = &task
			return nil
		}

		res := tx.Model(&model.AIApprovalTask{}).
			Where("approval_id = ? AND status = ?", approvalID, "pending").
			Updates(map[string]any{
				"status":            "rejected",
				"approved_by":       rejectedBy,
				"disapprove_reason": reason,
				"comment":           comment,
				"decided_at":        now,
				"lock_expires_at":   nil,
				"updated_at":        now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		task.Status = "rejected"
		task.ApprovedBy = rejectedBy
		task.DisapproveReason = reason
		task.Comment = comment
		task.DecidedAt = &now
		task.LockExpiresAt = nil
		task.UpdatedAt = now
		updated = &task
		return nil
	})
	return updated, err
}

// AcquireOrStealLease acquires a lease for approved tasks and refreshes expired locks.
func (d *AIApprovalTaskDAO) AcquireOrStealLease(ctx context.Context, approvalID string, leaseWindow time.Duration) (*model.AIApprovalTask, error) {
	if approvalID == "" {
		return nil, fmt.Errorf("approval id is required")
	}

	now := time.Now()
	var updated *model.AIApprovalTask
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task model.AIApprovalTask
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("approval_id = ?", approvalID).
			First(&task).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if task.Status != "approved" && task.Status != "approved_locked" {
			updated = &task
			return nil
		}
		if task.LockExpiresAt != nil && task.LockExpiresAt.After(now) {
			updated = &task
			return nil
		}

		leaseExpires := now.Add(leaseWindow)
		res := tx.Model(&model.AIApprovalTask{}).
			Where("approval_id = ? AND status IN ? AND (lock_expires_at IS NULL OR lock_expires_at <= ?)", approvalID, []string{"approved", "approved_locked"}, now).
			Updates(map[string]any{
				"status":          "approved_locked",
				"lock_expires_at": leaseExpires,
				"updated_at":      now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		task.Status = "approved_locked"
		task.LockExpiresAt = &leaseExpires
		task.UpdatedAt = now
		updated = &task
		return nil
	})
	return updated, err
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
