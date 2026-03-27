// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现 AI 审批 Outbox 事件的数据访问对象。
package ai

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AIApprovalOutboxDAO 提供 AI 审批 Outbox 事件的数据访问功能。
type AIApprovalOutboxDAO struct {
	db *gorm.DB
}

// approvalOutboxProcessingLease 处理租约时长。
const approvalOutboxProcessingLease = 2 * time.Minute

// NewAIApprovalOutboxDAO 创建 AI 审批 Outbox DAO 实例。
func NewAIApprovalOutboxDAO(db *gorm.DB) *AIApprovalOutboxDAO {
	return &AIApprovalOutboxDAO{db: db}
}

// NextSequence 为指定 Run 分配下一个单调递增的序列号。
//
// 参数:
//   - ctx: 上下文
//   - runID: 运行 ID
//
// 返回: 序列号
func (d *AIApprovalOutboxDAO) NextSequence(ctx context.Context, runID string) (int64, error) {
	var sequence int64
	err := d.db.WithContext(ctx).
		Raw("SELECT COALESCE(MAX(sequence), 0) + 1 FROM ai_approval_outbox_events WHERE run_id = ?", runID).
		Scan(&sequence).Error
	if err != nil {
		return 0, err
	}
	return sequence, nil
}

// EnqueueOrTouch 插入新的 Outbox 事件或更新已有的幂等键。
//
// 使用 UPSERT 语义：如果 approval_id + event_type 组合已存在则更新。
// 仅当状态为 pending 时才更新字段，保证已处理的事件不被覆盖。
//
// 参数:
//   - ctx: 上下文
//   - event: 事件记录
//
// 返回: 错误信息
func (d *AIApprovalOutboxDAO) EnqueueOrTouch(ctx context.Context, event *model.AIApprovalOutboxEvent) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "approval_id"}, {Name: "event_type"}},
			DoUpdates: clause.Assignments(map[string]any{
				"run_id": gorm.Expr(
					"CASE WHEN status = ? THEN ? ELSE run_id END",
					"pending",
					event.RunID,
				),
				"session_id": gorm.Expr(
					"CASE WHEN status = ? THEN ? ELSE session_id END",
					"pending",
					event.SessionID,
				),
				"tool_call_id": gorm.Expr(
					"CASE WHEN status = ? THEN ? ELSE tool_call_id END",
					"pending",
					event.ToolCallID,
				),
				"payload_json": gorm.Expr(
					"CASE WHEN status = ? THEN ? ELSE payload_json END",
					"pending",
					event.PayloadJSON,
				),
				"status": gorm.Expr(
					"CASE WHEN status = ? THEN ? ELSE status END",
					"pending",
					event.Status,
				),
				"next_retry_at": gorm.Expr(
					"CASE WHEN status = ? THEN ? ELSE next_retry_at END",
					"pending",
					event.NextRetryAt,
				),
				"updated_at": gorm.Expr(
					"CASE WHEN status = ? THEN ? ELSE updated_at END",
					"pending",
					now,
				),
			}),
		}).
		Create(event).Error
}

// ClaimPending 声明一个待处理的 Outbox 事件。
//
// 使用乐观锁机制：先查询待处理事件，再尝试获取处理锁。
// 如果存在候选事件但竞争失败，会自动重试。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 事件记录或 nil（无待处理事件）
func (d *AIApprovalOutboxDAO) ClaimPending(ctx context.Context) (*model.AIApprovalOutboxEvent, error) {
	for {
		claimed, hadCandidate, err := d.claimPendingAttempt(ctx)
		if err != nil {
			return nil, err
		}
		if claimed != nil {
			return claimed, nil
		}
		if !hadCandidate {
			return nil, nil
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
}

// claimPendingAttempt 单次尝试声明待处理事件。
func (d *AIApprovalOutboxDAO) claimPendingAttempt(ctx context.Context) (*model.AIApprovalOutboxEvent, bool, error) {
	var claimed *model.AIApprovalOutboxEvent
	hadCandidate := false

	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		staleProcessingBefore := now.Add(-approvalOutboxProcessingLease)
		var event model.AIApprovalOutboxEvent
		query := tx.Where(
			"(status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND updated_at <= ?)",
			"pending", now, "processing", staleProcessingBefore,
		).
			Order("next_retry_at ASC").
			Order("created_at ASC").
			Order("id ASC")
		if err := query.First(&event).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		hadCandidate = true

		updates := map[string]any{
			"status":     "processing",
			"updated_at": now,
		}
		result := tx.Model(&model.AIApprovalOutboxEvent{}).
			Where(
				"id = ? AND ((status = ?) OR (status = ? AND updated_at <= ?))",
				event.ID, "pending", "processing", staleProcessingBefore,
			).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		event.Status = "processing"
		event.UpdatedAt = now
		claimed = &event
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return claimed, hadCandidate, nil
}

// MarkDone 标记事件为已处理。
//
// 参数:
//   - ctx: 上下文
//   - id: 事件 ID
//
// 返回: 错误信息
func (d *AIApprovalOutboxDAO) MarkDone(ctx context.Context, id uint64) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.AIApprovalOutboxEvent{}).
		Where("id = ? AND status = ?", id, "processing").
		Updates(map[string]any{
			"status":     "done",
			"updated_at": now,
		}).Error
}

// MarkRetry 标记事件需要重试。
//
// 增加重试计数并设置下次重试时间。
//
// 参数:
//   - ctx: 上下文
//   - id: 事件 ID
//   - nextRetryAt: 下次重试时间
//
// 返回: 错误信息
func (d *AIApprovalOutboxDAO) MarkRetry(ctx context.Context, id uint64, nextRetryAt time.Time) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.AIApprovalOutboxEvent{}).
		Where("id = ? AND status = ?", id, "processing").
		Updates(map[string]any{
			"status":        "pending",
			"retry_count":   gorm.Expr("retry_count + ?", 1),
			"next_retry_at": &nextRetryAt,
			"updated_at":    now,
		}).Error
}
