package model

import "time"

// AISceneConfig stores per-scene AI behavior overrides.
type AISceneConfig struct {
	Scene              string    `gorm:"column:scene;type:varchar(128);primaryKey" json:"scene"`
	Name               string    `gorm:"column:name;type:varchar(128)" json:"name"`
	Description        string    `gorm:"column:description;type:text" json:"description"`
	ConstraintsJSON    string    `gorm:"column:constraints_json;type:longtext" json:"constraints_json"`
	AllowedToolsJSON   string    `gorm:"column:allowed_tools_json;type:longtext" json:"allowed_tools_json"`
	BlockedToolsJSON   string    `gorm:"column:blocked_tools_json;type:longtext" json:"blocked_tools_json"`
	ExamplesJSON       string    `gorm:"column:examples_json;type:longtext" json:"examples_json"`
	ApprovalConfigJSON string    `gorm:"column:approval_config_json;type:longtext" json:"approval_config_json"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AISceneConfig) TableName() string { return "ai_scene_configs" }
