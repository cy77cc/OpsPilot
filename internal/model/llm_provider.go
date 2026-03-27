// Package model 提供数据库模型定义。
//
// 本文件定义 LLM 供应商配置模型，用于从数据库中选择默认聊天模型。
package model

import (
	"time"

	"gorm.io/gorm"
)

// AILLMProvider 存储 LLM 模型配置，支持多供应商管理。
type AILLMProvider struct {
	ID            uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name          string         `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Provider      string         `gorm:"column:provider;type:varchar(32);not null;uniqueIndex:uk_ai_llm_providers_provider_model,priority:1;index:idx_ai_llm_providers_enabled_sort,priority:1" json:"provider"`
	Model         string         `gorm:"column:model;type:varchar(128);not null;uniqueIndex:uk_ai_llm_providers_provider_model,priority:2" json:"model"`
	BaseURL       string         `gorm:"column:base_url;type:varchar(512);not null" json:"base_url"`
	APIKey        string         `gorm:"column:api_key;type:varchar(256);not null" json:"-"`
	APIKeyVersion int            `gorm:"column:api_key_version;not null;default:1" json:"api_key_version"`
	Temperature   float64        `gorm:"column:temperature;type:decimal(3,2);not null;default:0.70" json:"temperature"`
	Thinking      bool           `gorm:"column:thinking;not null;default:false" json:"thinking"`
	IsDefault     bool           `gorm:"column:is_default;not null;default:false" json:"is_default"`
	IsEnabled     bool           `gorm:"column:is_enabled;not null;default:true" json:"is_enabled"`
	SortOrder     int            `gorm:"column:sort_order;not null;default:0;index:idx_ai_llm_providers_enabled_sort,priority:2,sort:desc" json:"sort_order"`
	ConfigVersion int            `gorm:"column:config_version;not null;default:1" json:"config_version"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回模型配置表名。
func (AILLMProvider) TableName() string {
	return "ai_llm_providers"
}
