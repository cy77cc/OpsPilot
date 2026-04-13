package model

import (
	"time"

	"gorm.io/gorm"
)

type AICheckpoint struct {
	CheckpointID string         `gorm:"column:checkpoint_id;type:varchar(64);primaryKey" json:"checkpoint_id"`
	SessionID    string         `gorm:"column:session_id;type:varchar(64);index" json:"session_id"`
	RunID        string         `gorm:"column:run_id;type:varchar(64);index" json:"run_id"`
	UserID       uint64         `gorm:"column:user_id;not null;default:0;index" json:"user_id"`
	Scene        string         `gorm:"column:scene;type:varchar(32);index" json:"scene"`
	Payload      []byte         `gorm:"column:payload;type:bytea;not null" json:"payload"`
	ExpiresAt    *time.Time     `gorm:"column:expires_at;index" json:"expires_at"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AICheckpoint) TableName() string { return "ai_checkpoints" }
