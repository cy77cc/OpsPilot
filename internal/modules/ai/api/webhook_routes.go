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
)

type alertHealJobEnqueuer struct {
	svcCtx *svc.ServiceContext
}

func (e *alertHealJobEnqueuer) Enqueue(ctx context.Context, event model.AIAlertIngestEvent) (string, error) {
	if e == nil || e.svcCtx == nil || e.svcCtx.DB == nil {
		return "", errors.New("service context not initialized")
	}

	eventID := strings.TrimSpace(event.ID)
	if eventID == "" {
		return "", errors.New("empty ingest event id")
	}

	row := &model.AIAlertHealJob{
		ID:      uuid.NewString(),
		EventID: eventID,
		Scene:   "alert_self_heal",
		Status:  "pending",
	}
	if err := e.svcCtx.DB.WithContext(ctx).Create(row).Error; err != nil {
		return "", err
	}
	return row.ID, nil
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
