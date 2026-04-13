package model

import (
	"time"

	"gorm.io/gorm"
)

type AIScenePrompt struct {
	ID           uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Scene        string         `gorm:"column:scene;type:varchar(32);not null;index:idx_ai_scene_prompts_scene_active_order,priority:1" json:"scene"`
	PromptText   string         `gorm:"column:prompt_text;type:text;not null" json:"prompt_text"`
	DisplayOrder int            `gorm:"column:display_order;not null;default:0;index:idx_ai_scene_prompts_scene_active_order,priority:3" json:"display_order"`
	IsActive     bool           `gorm:"column:is_active;not null;default:true;index:idx_ai_scene_prompts_scene_active_order,priority:2" json:"is_active"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIScenePrompt) TableName() string { return "ai_scene_prompts" }

type AISceneConfig struct {
	ID               uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Scene            string         `gorm:"column:scene;type:varchar(32);not null;uniqueIndex" json:"scene"`
	Description      string         `gorm:"column:description;type:text" json:"description"`
	ConstraintsJSON  string         `gorm:"column:constraints_json;type:text" json:"constraints_json"`
	AllowedToolsJSON string         `gorm:"column:allowed_tools_json;type:text" json:"allowed_tools_json"`
	BlockedToolsJSON string         `gorm:"column:blocked_tools_json;type:text" json:"blocked_tools_json"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AISceneConfig) TableName() string { return "ai_scene_configs" }
