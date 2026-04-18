package alertheal

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DAO 提供 AI 告警自愈数据访问。
type DAO struct {
	db *gorm.DB
}

type AlertHealJobView struct {
	ID               string    `gorm:"column:id" json:"id"`
	EventID          string    `gorm:"column:event_id" json:"event_id"`
	Scene            string    `gorm:"column:scene" json:"scene"`
	Status           string    `gorm:"column:status" json:"status"`
	Decision         string    `gorm:"column:decision" json:"decision"`
	RetryCount       int       `gorm:"column:retry_count" json:"retry_count"`
	MaxRetry         int       `gorm:"column:max_retry" json:"max_retry"`
	LastError        string    `gorm:"column:last_error" json:"last_error"`
	LatestRunID      string    `gorm:"column:latest_run_id" json:"latest_run_id"`
	CreatedAt        time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at" json:"updated_at"`
	EventStatus      string    `gorm:"column:event_status" json:"event_status"`
	EventTitle       string    `gorm:"column:event_title" json:"event_title"`
	EventTarget      string    `gorm:"column:event_target" json:"event_target"`
	EventFingerprint string    `gorm:"column:event_fingerprint" json:"event_fingerprint"`
	EventReceivedAt  time.Time `gorm:"column:event_received_at" json:"event_received_at"`
}

const defaultAutoFixingLease = 2 * time.Minute

func NewDAO(db *gorm.DB) *DAO {
	return &DAO{db: db}
}

// UpsertIngestEvent 按 dedupe_key 幂等写入并返回落库结果。
func (d *DAO) UpsertIngestEvent(ctx context.Context, row *model.AIAlertIngestEvent) (*model.AIAlertIngestEvent, error) {
	return upsertIngestEventWithDB(d.db.WithContext(ctx), row)
}

// UpsertIngestEvents 按批次幂等写入，任何错误都会回滚整批。
func (d *DAO) UpsertIngestEvents(ctx context.Context, rows []*model.AIAlertIngestEvent) ([]model.AIAlertIngestEvent, error) {
	out := make([]model.AIAlertIngestEvent, 0, len(rows))
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			saved, err := upsertIngestEventWithDB(tx, row)
			if err != nil {
				return err
			}
			if saved == nil {
				saved = row
			}
			out = append(out, *saved)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func upsertIngestEventWithDB(db *gorm.DB, row *model.AIAlertIngestEvent) (*model.AIAlertIngestEvent, error) {
	if err := db.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "dedupe_key"}},
			DoUpdates: clause.Assignments(map[string]any{
				"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
			}),
		}).
		Create(row).Error; err != nil {
		return nil, err
	}

	var saved model.AIAlertIngestEvent
	if err := db.Where("dedupe_key = ?", row.DedupeKey).First(&saved).Error; err != nil {
		return nil, err
	}
	return &saved, nil
}

func (d *DAO) ClaimRunnableJob(ctx context.Context, now time.Time) (*model.AIAlertHealJob, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("alertheal dao not initialized")
	}
	now = now.UTC()
	staleBefore := now.Add(-defaultAutoFixingLease)

	for {
		var claimed *model.AIAlertHealJob
		hadCandidate := false
		err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var candidate model.AIAlertHealJob
			if err := tx.Where(
				"(status = ?) OR (status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?) OR (status = ? AND updated_at <= ?)",
				"pending", "retry_wait", now, "auto_fixing", staleBefore,
			).Order("next_retry_at ASC").Order("created_at ASC").Order("id ASC").Take(&candidate).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			hadCandidate = true

			result := tx.Model(&model.AIAlertHealJob{}).
				Where(
					"id = ? AND ((status = ?) OR (status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?) OR (status = ? AND updated_at <= ?))",
					candidate.ID, "pending", "retry_wait", now, "auto_fixing", staleBefore,
				).
				Updates(map[string]any{
					"status":        "auto_fixing",
					"next_retry_at": nil,
					"updated_at":    now,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}

			candidate.Status = "auto_fixing"
			candidate.NextRetryAt = nil
			claimed = &candidate
			return nil
		})
		if err != nil {
			return nil, err
		}
		if claimed != nil {
			return claimed, nil
		}
		if !hadCandidate {
			return nil, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
}

func (d *DAO) CancelIfResolved(ctx context.Context, jobID string) (bool, error) {
	if d == nil || d.db == nil {
		return false, errors.New("alertheal dao not initialized")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return false, errors.New("empty alert heal job id")
	}

	var canceled bool
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var job model.AIAlertHealJob
		if err := tx.Where("id = ?", jobID).Take(&job).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}

		var event model.AIAlertIngestEvent
		if err := tx.Where("id = ?", job.EventID).Take(&event).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		shouldCancel := strings.EqualFold(strings.TrimSpace(event.Status), "resolved")
		if !shouldCancel {
			var resolvedCount int64
			if err := tx.Model(&model.AIAlertIngestEvent{}).
				Where(
					"source = ? AND fingerprint = ? AND status = ? AND (received_at > ? OR (received_at = ? AND id <> ?))",
					event.Source, event.Fingerprint, "resolved", event.ReceivedAt, event.ReceivedAt, event.ID,
				).Count(&resolvedCount).Error; err != nil {
				return err
			}
			shouldCancel = resolvedCount > 0
		}
		if !shouldCancel {
			return nil
		}

		result := tx.Model(&model.AIAlertHealJob{}).
			Where("id = ? AND status NOT IN ?", jobID, []string{"succeeded", "canceled_resolved", "no_action", "failed_manual"}).
			Updates(map[string]any{
				"status":        "canceled_resolved",
				"next_retry_at": nil,
				"last_error":    "",
			})
		if result.Error != nil {
			return result.Error
		}
		canceled = result.RowsAffected > 0
		return nil
	})
	return canceled, err
}

func (d *DAO) MarkSucceeded(ctx context.Context, jobID, runID string) error {
	return d.updateAutoFixingJob(ctx, jobID, map[string]any{
		"status":        "succeeded",
		"latest_run_id": strings.TrimSpace(runID),
		"last_error":    "",
		"next_retry_at": nil,
	})
}

func (d *DAO) MarkWaitingApproval(ctx context.Context, jobID, runID, lastError string, consumeRetry bool) error {
	updates := map[string]any{
		"status":        "waiting_approval",
		"latest_run_id": strings.TrimSpace(runID),
		"last_error":    strings.TrimSpace(lastError),
		"next_retry_at": nil,
	}
	if consumeRetry {
		updates["retry_count"] = gorm.Expr("retry_count + ?", 1)
	}
	return d.updateAutoFixingJob(ctx, jobID, updates)
}

func (d *DAO) MarkRetryWait(ctx context.Context, jobID, lastError string, nextRetryAt time.Time) error {
	if nextRetryAt.IsZero() {
		nextRetryAt = time.Now().UTC()
	}
	return d.updateAutoFixingJob(ctx, jobID, map[string]any{
		"status":        "retry_wait",
		"last_error":    strings.TrimSpace(lastError),
		"next_retry_at": nextRetryAt.UTC(),
		"retry_count":   gorm.Expr("retry_count + ?", 1),
	})
}

func (d *DAO) RenewAutoFixingLease(ctx context.Context, jobID string, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return d.updateAutoFixingJob(ctx, jobID, map[string]any{
		"updated_at": now.UTC(),
	})
}

func (d *DAO) updateAutoFixingJob(ctx context.Context, jobID string, updates map[string]any) error {
	if d == nil || d.db == nil {
		return errors.New("alertheal dao not initialized")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return errors.New("empty alert heal job id")
	}

	result := d.db.WithContext(ctx).Model(&model.AIAlertHealJob{}).
		Where("id = ? AND status = ?", jobID, "auto_fixing").
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (d *DAO) ListJobViewsByFingerprint(ctx context.Context, fingerprint string) ([]AlertHealJobView, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("alertheal dao not initialized")
	}
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		return []AlertHealJobView{}, nil
	}

	rows := make([]AlertHealJobView, 0)
	if err := d.jobViewsQuery(ctx).
		Where("events.fingerprint = ?", fingerprint).
		Order("jobs.created_at DESC").
		Order("jobs.id DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (d *DAO) GetJobView(ctx context.Context, jobID string) (*AlertHealJobView, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("alertheal dao not initialized")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, gorm.ErrRecordNotFound
	}

	var row AlertHealJobView
	if err := d.jobViewsQuery(ctx).Where("jobs.id = ?", jobID).Take(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (d *DAO) RetryJob(ctx context.Context, jobID string, now time.Time) (*AlertHealJobView, error) {
	if d == nil || d.db == nil {
		return nil, errors.New("alertheal dao not initialized")
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row AlertHealJobView
		if err := jobViewsQuery(tx).Where("jobs.id = ?", jobID).Take(&row).Error; err != nil {
			return err
		}
		if row.EventStatus == "resolved" {
			return gorm.ErrInvalidData
		}
		if isActiveAlertHealStatus(row.Status) {
			return gorm.ErrInvalidData
		}

		var resolvedCount int64
		if err := tx.Model(&model.AIAlertIngestEvent{}).
			Where(
				"fingerprint = ? AND status = ? AND (received_at > ? OR (received_at = ? AND id <> ?))",
				row.EventFingerprint,
				"resolved",
				row.EventReceivedAt,
				row.EventReceivedAt,
				row.EventID,
			).
			Count(&resolvedCount).Error; err != nil {
			return err
		}
		if resolvedCount > 0 {
			return gorm.ErrInvalidData
		}

		result := tx.Model(&model.AIAlertHealJob{}).
			Where("id = ? AND status NOT IN ?", jobID, []string{"pending", "auto_fixing", "waiting_approval"}).
			Updates(map[string]any{
				"status":        "pending",
				"decision":      "",
				"retry_count":   0,
				"next_retry_at": nil,
				"last_error":    "",
				"updated_at":    now.UTC(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrInvalidData
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d.GetJobView(ctx, jobID)
}

func (d *DAO) jobViewsQuery(ctx context.Context) *gorm.DB {
	return jobViewsQuery(d.db.WithContext(ctx))
}

func jobViewsQuery(db *gorm.DB) *gorm.DB {
	return db.Table("ai_alert_heal_jobs AS jobs").
		Select(
			"jobs.id, jobs.event_id, jobs.scene, jobs.status, jobs.decision, jobs.retry_count, jobs.max_retry, jobs.last_error, jobs.latest_run_id, jobs.created_at, jobs.updated_at, " +
				"events.status AS event_status, events.title AS event_title, events.target AS event_target, events.fingerprint AS event_fingerprint, events.received_at AS event_received_at",
		).
		Joins("JOIN ai_alert_ingest_events AS events ON events.id = jobs.event_id")
}

func isActiveAlertHealStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "pending", "auto_fixing", "waiting_approval":
		return true
	default:
		return false
	}
}
