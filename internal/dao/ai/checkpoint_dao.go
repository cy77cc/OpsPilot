// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现 AI 检查点的数据访问对象。
package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AICheckpointDAO 提供 AI 检查点的数据访问功能。
type AICheckpointDAO struct {
	db *gorm.DB
}

// NewAICheckpointDAO 创建 AI 检查点 DAO 实例。
func NewAICheckpointDAO(db *gorm.DB) *AICheckpointDAO {
	return &AICheckpointDAO{db: db}
}

// Upsert 创建或更新检查点记录。
//
// 使用 UPSERT 语义：如果 checkpoint_id 已存在则更新，否则插入。
// 用于 Human-in-the-Loop 工作流的状态持久化。
//
// 参数:
//   - ctx: 上下文
//   - checkpoint: 检查点记录
//
// 返回: 错误信息
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

// Get 根据检查点 ID 获取检查点记录。
//
// 参数:
//   - ctx: 上下文
//   - checkpointID: 检查点 ID
//
// 返回: 检查点记录或 nil（不存在时）
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
