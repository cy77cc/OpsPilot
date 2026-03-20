package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

type AIRunEventDAO struct {
	db *gorm.DB
}

func NewAIRunEventDAO(db *gorm.DB) *AIRunEventDAO {
	return &AIRunEventDAO{db: db}
}

func (d *AIRunEventDAO) Create(ctx context.Context, event *model.AIRunEvent) error {
	return d.db.WithContext(ctx).Create(event).Error
}

func (d *AIRunEventDAO) ListByRun(ctx context.Context, runID string) ([]model.AIRunEvent, error) {
	var events []model.AIRunEvent
	err := d.db.WithContext(ctx).
		Where("run_id = ?", runID).
		Order("seq ASC, created_at ASC, id ASC").
		Find(&events).Error
	return events, err
}
