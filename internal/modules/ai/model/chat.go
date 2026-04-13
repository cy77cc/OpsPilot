package model

import (
	"time"

	"gorm.io/gorm"
)

type AIChatSession struct {
	ID        string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	UserID    uint64         `gorm:"column:user_id;not null;index:idx_ai_chat_sessions_user_scene_updated,priority:1;index:idx_ai_chat_sessions_user_id" json:"user_id"`
	Scene     string         `gorm:"column:scene;type:varchar(32);not null;default:'ai';index:idx_ai_chat_sessions_user_scene_updated,priority:2" json:"scene"`
	Title     string         `gorm:"column:title;type:varchar(255);not null;default:''" json:"title"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime;index:idx_ai_chat_sessions_user_scene_updated,priority:3,sort:desc" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIChatSession) TableName() string { return "ai_chat_sessions" }

type AIChatMessage struct {
	ID           string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	SessionID    string         `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_chat_messages_session_created,priority:1;uniqueIndex:uk_ai_chat_messages_session_seq,priority:1;index:idx_ai_chat_messages_session_role,priority:1" json:"session_id"`
	SessionIDNum int            `gorm:"column:session_id_num;not null;default:0;uniqueIndex:uk_ai_chat_messages_session_seq,priority:2" json:"session_id_num"`
	Role         string         `gorm:"column:role;type:varchar(16);not null;default:'assistant';index:idx_ai_chat_messages_session_role,priority:2" json:"role"`
	Content      string         `gorm:"column:content;type:text;not null" json:"content"`
	Status       string         `gorm:"column:status;type:varchar(16);not null;default:'done'" json:"status"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_ai_chat_messages_session_created,priority:2" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIChatMessage) TableName() string { return "ai_chat_messages" }
