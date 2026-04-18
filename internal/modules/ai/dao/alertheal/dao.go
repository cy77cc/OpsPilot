package alertheal

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DAO 提供 AI 告警自愈数据访问。
type DAO struct {
	db *gorm.DB
}

func NewDAO(db *gorm.DB) *DAO {
	return &DAO{db: db}
}

// UpsertIngestEvent 按 dedupe_key 幂等写入并返回落库结果。
func (d *DAO) UpsertIngestEvent(ctx context.Context, row *model.AIAlertIngestEvent) (*model.AIAlertIngestEvent, error) {
	if err := d.db.WithContext(ctx).
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
	if err := d.db.WithContext(ctx).Where("dedupe_key = ?", row.DedupeKey).First(&saved).Error; err != nil {
		return nil, err
	}
	return &saved, nil
}
