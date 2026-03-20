package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AIRunProjectionDAO struct {
	db *gorm.DB
}

func NewAIRunProjectionDAO(db *gorm.DB) *AIRunProjectionDAO {
	return &AIRunProjectionDAO{db: db}
}

func (d *AIRunProjectionDAO) Upsert(ctx context.Context, projection *model.AIRunProjection) error {
	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "run_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"session_id",
				"version",
				"status",
				"projection_json",
				"updated_at",
			}),
		}).
		Create(projection).Error
}

func (d *AIRunProjectionDAO) GetByRunID(ctx context.Context, runID string) (*model.AIRunProjection, error) {
	var projection model.AIRunProjection
	if err := d.db.WithContext(ctx).Where("run_id = ?", runID).First(&projection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &projection, nil
}
