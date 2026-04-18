package api

import (
	"context"
	"errors"
	"strings"
	"time"

	aidaoalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/alertheal"
	aihttp "github.com/cy77cc/OpsPilot/internal/modules/ai/interfaces/http"
	alertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/logic/alertheal"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type alertHealJobEnqueuer struct {
	svcCtx *svc.ServiceContext
}

func (e *alertHealJobEnqueuer) EnqueueBatch(ctx context.Context, events []model.AIAlertIngestEvent) (string, error) {
	if e == nil || e.svcCtx == nil || e.svcCtx.DB == nil {
		return "", errors.New("service context not initialized")
	}
	if len(events) == 0 {
		return "", errors.New("empty ingest events")
	}

	var firstJobID string
	err := e.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, event := range events {
			eventID := strings.TrimSpace(event.ID)
			if eventID == "" {
				return errors.New("empty ingest event id")
			}

			row := &model.AIAlertHealJob{
				ID:      uuid.NewString(),
				EventID: eventID,
				Scene:   "alert_self_heal",
				Status:  "pending",
			}
			if cErr := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "event_id"}},
				DoNothing: true,
			}).Create(row).Error; cErr != nil {
				return cErr
			}

			var saved model.AIAlertHealJob
			if qErr := tx.Where("event_id = ?", eventID).Take(&saved).Error; qErr != nil {
				return qErr
			}
			if strings.EqualFold(strings.TrimSpace(event.Status), "resolved") {
				if err := cancelResolvedAlertHealState(tx, event); err != nil {
					return err
				}
			}
			if i == 0 {
				firstJobID = saved.ID
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return firstJobID, nil
}

// RegisterAIWebhookHandlers registers public webhook routes for AI alert self-healing.
func RegisterAIWebhookHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	if v1 == nil || svcCtx == nil || svcCtx.DB == nil {
		return
	}
	if alertheal.NewExecutor(svcCtx) == nil {
		return
	}
	dao := aidaoalertheal.NewDAO(svcCtx.DB)
	handler := aihttp.NewAlertWebhookHandler(alertheal.NewService(dao), &alertHealJobEnqueuer{svcCtx: svcCtx})
	v1.POST("/ai/alerts/webhook", handler.Handle)
}

func cancelResolvedAlertHealState(tx *gorm.DB, event model.AIAlertIngestEvent) error {
	source := strings.TrimSpace(event.Source)
	fingerprint := strings.TrimSpace(event.Fingerprint)
	if source == "" || fingerprint == "" {
		return nil
	}

	now := time.Now().UTC()
	statuses := []string{"pending", "retry_wait", "auto_fixing", "waiting_approval"}
	var jobs []model.AIAlertHealJob
	if err := tx.Table("ai_alert_heal_jobs AS jobs").
		Select("jobs.*").
		Joins("JOIN ai_alert_ingest_events AS events ON events.id = jobs.event_id").
		Where("events.source = ? AND events.fingerprint = ?", source, fingerprint).
		Where("jobs.status IN ?", statuses).
		Order("jobs.created_at ASC").
		Order("jobs.id ASC").
		Find(&jobs).Error; err != nil {
		return err
	}

	for _, job := range jobs {
		if err := tx.Model(&model.AIAlertHealJob{}).
			Where("id = ? AND status IN ?", job.ID, statuses).
			Updates(map[string]any{
				"status":        "canceled_resolved",
				"last_error":    "",
				"next_retry_at": nil,
			}).Error; err != nil {
			return err
		}

		runID := strings.TrimSpace(job.LatestRunID)
		if runID == "" {
			continue
		}
		if err := tx.Model(&model.AIRun{}).
			Where("id = ?", runID).
			Updates(map[string]any{
				"status":        "cancelled",
				"error_message": "alert resolved",
				"finished_at":   now,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AIApprovalTask{}).
			Where("run_id = ? AND status IN ?", runID, []string{"pending", "approved"}).
			Updates(map[string]any{
				"status":     "expired",
				"comment":    "alert resolved",
				"decided_at": now,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
	}

	return nil
}
