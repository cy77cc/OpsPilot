// Package ai 提供 AI 模块的数据访问层。
//
// 本文件实现 AI 运行内容的数据访问对象。
package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// AIRunContentDAO 提供 AI 运行内容的数据访问功能。
type AIRunContentDAO struct {
	db *gorm.DB
}

// NewAIRunContentDAO 创建 AI 运行内容 DAO 实例。
func NewAIRunContentDAO(db *gorm.DB) *AIRunContentDAO {
	return &AIRunContentDAO{db: db}
}

// Create 创建新的内容记录。
func (d *AIRunContentDAO) Create(ctx context.Context, content *model.AIRunContent) error {
	return d.db.WithContext(ctx).Create(content).Error
}

// Get 根据 ID 获取内容记录。
//
// 参数:
//   - ctx: 上下文
//   - id: 内容 ID
//
// 返回: 内容记录或 nil（不存在时）
func (d *AIRunContentDAO) Get(ctx context.Context, id string) (*model.AIRunContent, error) {
	var content model.AIRunContent
	if err := d.db.WithContext(ctx).Where("id = ?", id).First(&content).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &content, nil
}
