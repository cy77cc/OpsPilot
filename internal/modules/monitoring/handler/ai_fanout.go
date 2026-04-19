package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	aidaoalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/alertheal"
	ailogicalertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/logic/alertheal"
	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AlertAIFanout 定义监控告警向 AI 告警自愈链路扇出的接口。
type AlertAIFanout interface {
	HandleAlertmanager(ctx context.Context, payload AlertmanagerWebhook) error
}

type aiAlertHealFanout struct {
	ingestor interface {
		Ingest(ctx context.Context, protocol string, raw []byte) ([]aimodel.AIAlertIngestEvent, error)
	}
	enqueuer interface {
		EnqueueBatch(ctx context.Context, events []aimodel.AIAlertIngestEvent) (string, error)
	}
}

func newAIAlertHealFanout(svcCtx *svc.ServiceContext) AlertAIFanout {
	if svcCtx == nil || svcCtx.DB == nil {
		return nil
	}
	dao := aidaoalertheal.NewDAO(svcCtx.DB)
	return &aiAlertHealFanout{
		ingestor: ailogicalertheal.NewService(dao),
		enqueuer: &aiAlertHealJobEnqueuer{svcCtx: svcCtx},
	}
}

func (f *aiAlertHealFanout) HandleAlertmanager(ctx context.Context, payload AlertmanagerWebhook) error {
	if f == nil || f.ingestor == nil || f.enqueuer == nil {
		return errors.New("ai fanout is not initialized")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	events, err := f.ingestor.Ingest(ctx, "alertmanager", raw)
	if err != nil || len(events) == 0 {
		return err
	}
	_, err = f.enqueuer.EnqueueBatch(ctx, events)
	return err
}

type aiAlertHealJobEnqueuer struct {
	svcCtx *svc.ServiceContext
}

func (e *aiAlertHealJobEnqueuer) EnqueueBatch(ctx context.Context, events []aimodel.AIAlertIngestEvent) (string, error) {
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
			row := &aimodel.AIAlertHealJob{
				ID:      uuid.NewString(),
				EventID: eventID,
				Scene:   "alert_self_heal",
				Status:  "pending",
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "event_id"}},
				DoNothing: true,
			}).Create(row).Error; err != nil {
				return err
			}
			var saved aimodel.AIAlertHealJob
			if err := tx.Where("event_id = ?", eventID).Take(&saved).Error; err != nil {
				return err
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
