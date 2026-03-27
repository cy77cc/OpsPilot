// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现 AI 运行投影的数据访问对象。
package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AIRunProjectionDAO 提供 AI 运行投影的数据访问功能。
type AIRunProjectionDAO struct {
	db *gorm.DB
}

// NewAIRunProjectionDAO 创建 AI 运行投影 DAO 实例。
func NewAIRunProjectionDAO(db *gorm.DB) *AIRunProjectionDAO {
	return &AIRunProjectionDAO{db: db}
}

// Upsert 创建或更新投影记录。
//
// 使用 UPSERT 语义：如果 run_id 已存在则更新，否则插入。
//
// 参数:
//   - ctx: 上下文
//   - projection: 投影记录
//
// 返回: 错误信息
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

// GetByRunID 根据运行 ID 获取投影记录。
//
// 参数:
//   - ctx: 上下文
//   - runID: 运行 ID
//
// 返回: 投影记录或 nil（不存在时）
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
