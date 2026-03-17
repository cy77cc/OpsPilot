package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AICheckpointDAO struct {
	db *gorm.DB
}

func NewAICheckpointDAO(db *gorm.DB) *AICheckpointDAO {
	return &AICheckpointDAO{db: db}
}

func (d *AICheckpointDAO) Upsert(ctx context.Context, checkpoint *model.AICheckpoint) error {
	return d.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "checkpoint_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"session_id",
				"run_id",
				"user_id",
				"scene",
				"payload",
				"expires_at",
				"updated_at",
			}),
		}).
		Create(checkpoint).Error
}

func (d *AICheckpointDAO) Get(ctx context.Context, checkpointID string) (*model.AICheckpoint, error) {
	var checkpoint model.AICheckpoint
	err := d.db.WithContext(ctx).
		Where("checkpoint_id = ?", checkpointID).
		First(&checkpoint).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &checkpoint, nil
}
