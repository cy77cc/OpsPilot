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

// AIApprovalOutboxDAO persists approval side effects for asynchronous processing.
type AIApprovalOutboxDAO struct {
	db *gorm.DB
}

var approvalOutboxClaimUpdater = func(tx *gorm.DB, id uint64, updatedAt time.Time) (int64, error) {
	res := tx.Model(&model.AIApprovalOutboxEvent{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]any{
			"status":        "processing",
			"next_retry_at": nil,
			"updated_at":    updatedAt,
		})
	return res.RowsAffected, res.Error
}

// NewAIApprovalOutboxDAO creates an approval outbox DAO.
func NewAIApprovalOutboxDAO(db *gorm.DB) *AIApprovalOutboxDAO {
	return &AIApprovalOutboxDAO{db: db}
}

// EnqueueOrTouch inserts a new outbox event or refreshes the existing record without
// resurrecting completed work.
func (d *AIApprovalOutboxDAO) EnqueueOrTouch(ctx context.Context, event *model.AIApprovalOutboxEvent) error {
	if event == nil {
		return fmt.Errorf("event is required")
	}
	if event.ApprovalID == "" || event.EventType == "" {
		return fmt.Errorf("approval_id and event_type are required")
	}

	now := time.Now()
	if event.Status == "" {
		event.Status = "pending"
	}
	if err := d.db.WithContext(ctx).Create(event).Error; err == nil {
		return nil
	} else if !isDuplicateKeyError(err) {
		return err
	}

	var existing model.AIApprovalOutboxEvent
	if err := d.db.WithContext(ctx).
		Where("approval_id = ? AND event_type = ?", event.ApprovalID, event.EventType).
		First(&existing).Error; err != nil {
		return err
	}

	event.ID = existing.ID
	event.RetryCount = existing.RetryCount
	event.CreatedAt = existing.CreatedAt
	event.UpdatedAt = now

	updates := map[string]any{
		"run_id":       event.RunID,
		"session_id":   event.SessionID,
		"payload_json": event.PayloadJSON,
		"updated_at":   now,
	}
	if existing.Status == "pending" {
		updates["status"] = "pending"
		updates["next_retry_at"] = nil
	}
	return d.db.WithContext(ctx).
		Model(&model.AIApprovalOutboxEvent{}).
		Where("id = ?", existing.ID).
		Updates(updates).Error
}

// ClaimPending returns the oldest claimable pending event, using update-by-condition
// so that only one worker can own each event.
func (d *AIApprovalOutboxDAO) ClaimPending(ctx context.Context) (*model.AIApprovalOutboxEvent, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var (
			claimed      *model.AIApprovalOutboxEvent
			sawCandidate bool
			queueEmpty   bool
		)
		err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			now := time.Now()
			var candidate model.AIApprovalOutboxEvent
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)", "pending", now).
				Order("created_at ASC, id ASC").
				First(&candidate).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				queueEmpty = true
				return nil
			}
			if err != nil {
				return err
			}
			sawCandidate = true

			updatedAt := time.Now()
			rowsAffected, updateErr := approvalOutboxClaimUpdater(tx, candidate.ID, updatedAt)
			if updateErr != nil {
				return updateErr
			}
			if rowsAffected == 0 {
				return nil
			}

			candidate.Status = "processing"
			candidate.NextRetryAt = nil
			candidate.UpdatedAt = updatedAt
			claimed = &candidate
			return nil
		})
		if err != nil {
			return nil, err
		}
		if claimed != nil {
			return claimed, nil
		}
		if queueEmpty {
			return nil, nil
		}
		if sawCandidate {
			continue
		}
	}
}

// MarkDone marks an outbox event as delivered.
func (d *AIApprovalOutboxDAO) MarkDone(ctx context.Context, id uint64) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.AIApprovalOutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        "done",
			"next_retry_at": nil,
			"updated_at":    now,
		}).Error
}

// MarkRetry returns the event to the queue with a new retry schedule.
func (d *AIApprovalOutboxDAO) MarkRetry(ctx context.Context, id uint64, nextRetryAt time.Time) error {
	now := time.Now()
	return d.db.WithContext(ctx).
		Model(&model.AIApprovalOutboxEvent{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        "pending",
			"retry_count":   gorm.Expr("retry_count + 1"),
			"next_retry_at": nextRetryAt,
			"updated_at":    now,
		}).Error
}
