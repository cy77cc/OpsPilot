package model

import "time"

// AIScenePrompt 场景快捷提示词
type AIScenePrompt struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Scene        string    `gorm:"column:scene;type:varchar(128);index:idx_scene_prompts" json:"scene"`
	PromptText   string    `gorm:"column:prompt_text;type:text" json:"prompt_text"`
	PromptType   string    `gorm:"column:prompt_type;type:varchar(32);default:'quick_action'" json:"prompt_type"`
	DisplayOrder int       `gorm:"column:display_order;default:0" json:"display_order"`
	IsActive     bool      `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AIScenePrompt) TableName() string { return "ai_scene_prompts" }
