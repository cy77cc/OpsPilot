package api

import (
	"context"
	"errors"
	"strings"

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
	dao := aidaoalertheal.NewDAO(svcCtx.DB)
	handler := aihttp.NewAlertWebhookHandler(alertheal.NewService(dao), &alertHealJobEnqueuer{svcCtx: svcCtx})
	v1.POST("/ai/alerts/webhook", handler.Handle)
}
