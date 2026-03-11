// Package model 提供数据库模型定义。
//
// 本文件定义 AI 聊天会话和消息相关的数据模型。
package model

import "time"

// AIChatSession 是 AI 聊天会话表模型，按用户和场景存储对话元数据。
//
// 表名: ai_chat_sessions
// 关联: AIChatMessage (一对多，通过 session_id)
type AIChatSession struct {
	ID        string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                         // 会话唯一标识 (UUID)
	UserID    uint64    `gorm:"column:user_id;index:idx_ai_session_user_scene" json:"user_id"`          // 所属用户 ID
	Scene     string    `gorm:"column:scene;type:varchar(128);index:idx_ai_session_user_scene" json:"scene"` // 场景标识 (如: host, cluster, service, k8s)
	Title     string    `gorm:"column:title;type:varchar(128)" json:"title"`                             // 会话标题 (自动生成或用户修改)
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                      // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                      // 更新时间
}

// TableName 返回 AI 聊天会话表名。
func (AIChatSession) TableName() string { return "ai_chat_sessions" }

// AIChatMessage 是 AI 聊天消息表模型，存储会话中的每条消息。
//
// 表名: ai_chat_messages
// 关联: AIChatSession (多对一，通过 session_id)
//
// 字段说明:
//   - Thinking: AI 思考过程，仅 assistant 角色有效
//   - MetadataJSON: 扩展元数据，存储工具调用、引用等信息
type AIChatMessage struct {
	ID           string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                              // 消息唯一标识 (UUID)
	SessionID    string    `gorm:"column:session_id;type:varchar(64);index:idx_ai_msg_session_created" json:"session_id"` // 所属会话 ID
	Role         string    `gorm:"column:role;type:varchar(32);index" json:"role"`                               // 角色: user/assistant/system
	Content      string    `gorm:"column:content;type:longtext" json:"content"`                                  // 消息内容
	Thinking     string    `gorm:"column:thinking;type:longtext" json:"thinking"`                               // AI 思考过程 (仅 assistant 角色)
	Status       string    `gorm:"column:status;type:varchar(32);default:''" json:"status"`                     // 状态: pending/streaming/done/error
	MetadataJSON string    `gorm:"column:metadata_json;type:longtext" json:"metadata_json"`                     // 扩展元数据 (JSON 格式)
	CreatedAt    time.Time `gorm:"column:created_at;index:idx_ai_msg_session_created;autoCreateTime" json:"created_at"` // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                          // 更新时间
}

// TableName 返回 AI 聊天消息表名。
func (AIChatMessage) TableName() string { return "ai_chat_messages" }
