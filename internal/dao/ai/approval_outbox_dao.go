package ai

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AIApprovalOutboxDAO manages durable approval outbox events.
type AIApprovalOutboxDAO struct {
	db *gorm.DB
}

const approvalOutboxProcessingLease = 2 * time.Minute

// NewAIApprovalOutboxDAO creates an approval outbox DAO.
func NewAIApprovalOutboxDAO(db *gorm.DB) *AIApprovalOutboxDAO {
	return &AIApprovalOutboxDAO{db: db}
}

// EnqueueOrTouch inserts a new outbox row or updates the existing idempotency key.
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

// ClaimPending claims the oldest pending event ready for processing.
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

// MarkDone marks the outbox event as delivered.
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

// MarkRetry increments retry_count and schedules the next attempt.
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
