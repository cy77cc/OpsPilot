// Package model 提供数据库模型定义。
//
// 本文件定义 AI 场景快捷提示词相关的数据模型，用于存储各场景的预设提示词模板。
package model

import "time"

// AIScenePrompt 是场景快捷提示词表模型，存储各场景的预设提示词。
//
// 表名: ai_scene_prompts
// 用途:
//   - 为不同场景提供快捷操作入口
//   - 支持用户快速发起常见运维任务
//   - 可动态配置和启用/禁用
//
// 场景示例:
//   - host: 主机运维场景
//   - cluster: 集群管理场景
//   - service: 服务治理场景
//   - k8s: Kubernetes 运维场景
type AIScenePrompt struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                     // 主键 ID
	Scene        string    `gorm:"column:scene;type:varchar(128);index:idx_scene_prompts" json:"scene"` // 场景标识 (如: host, cluster)
	PromptText   string    `gorm:"column:prompt_text;type:text" json:"prompt_text"`                 // 提示词文本
	PromptType   string    `gorm:"column:prompt_type;type:varchar(32);default:'quick_action'" json:"prompt_type"` // 提示词类型: quick_action/template
	DisplayOrder int       `gorm:"column:display_order;default:0" json:"display_order"`            // 显示顺序 (升序排列)
	IsActive     bool      `gorm:"column:is_active;default:true" json:"is_active"`                  // 是否启用
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`              // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`              // 更新时间
}

// TableName 返回场景快捷提示词表名。
func (AIScenePrompt) TableName() string { return "ai_scene_prompts" }
