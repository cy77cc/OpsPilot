package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

type AIRunContentDAO struct {
	db *gorm.DB
}

func NewAIRunContentDAO(db *gorm.DB) *AIRunContentDAO {
	return &AIRunContentDAO{db: db}
}

func (d *AIRunContentDAO) Create(ctx context.Context, content *model.AIRunContent) error {
	return d.db.WithContext(ctx).Create(content).Error
}

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
