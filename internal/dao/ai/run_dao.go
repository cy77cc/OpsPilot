package ai 

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

type AIRunDAO struct {
	db *gorm.DB
}

type AIRunStatusUpdate struct {
	Status             string
	AssistantMessageID string
	ProgressSummary    string
	ErrorMessage       string
}

func NewAIRunDAO(db *gorm.DB) *AIRunDAO {
	return &AIRunDAO{db: db}
}

func (d *AIRunDAO) CreateRun(ctx context.Context, run *model.AIRun) error {
	return d.db.WithContext(ctx).Create(run).Error
}

func (d *AIRunDAO) UpdateRunStatus(ctx context.Context, runID string, update AIRunStatusUpdate) error {
	updates := map[string]any{
		"status":              update.Status,
		"assistant_message_id": update.AssistantMessageID,
		"progress_summary":    update.ProgressSummary,
		"error_message":       update.ErrorMessage,
	}
	return d.db.WithContext(ctx).
		Model(&model.AIRun{}).
		Where("id = ?", runID).
		Updates(updates).Error
}

func (d *AIRunDAO) GetRun(ctx context.Context, runID string) (*model.AIRun, error) {
	var run model.AIRun
	err := d.db.WithContext(ctx).Where("id = ?", runID).First(&run).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &run, nil
}
