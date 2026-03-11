// Package model 提供数据库模型定义。
//
// 本文件定义 AI 检查点相关的数据模型，用于中断/恢复场景的状态持久化。
package model

import "time"

// AICheckPoint 是 AI 检查点表模型，存储 eino 框架的检查点数据。
//
// 表名: ai_checkpoints
// 用途: 支持长时间运行任务的断点续传，如复杂运维操作的恢复
//
// 字段说明:
//   - Key: 检查点唯一键，通常为 session_id 或 task_id
//   - Value: 序列化的状态数据，包含执行上下文、中间结果等
type AICheckPoint struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`         // 自增主键
	Key       string    `gorm:"column:key;type:varchar(255);uniqueIndex;not null" json:"key"` // 检查点唯一键
	Value     []byte    `gorm:"column:value;type:mediumblob;not null" json:"value"`   // 序列化状态数据 (最大 16MB)
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`   // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`   // 更新时间
}

// TableName 返回 AI 检查点表名。
func (AICheckPoint) TableName() string { return "ai_checkpoints" }
