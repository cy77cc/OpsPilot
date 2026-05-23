package model

import "time"

// TunnelSession 是隧道会话表模型，记录活跃隧道信息。
//
// 表名: tunnel_sessions
type TunnelSession struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TunnelID  string    `gorm:"column:tunnel_id;uniqueIndex;size:64" json:"tunnel_id"`
	GatewayID uint64    `gorm:"column:gateway_id;index" json:"gateway_id"`
	HostID    uint64    `gorm:"column:host_id;index" json:"host_id"`
	AgentID   string    `gorm:"column:agent_id;size:64" json:"agent_id"`
	Status    string    `gorm:"column:status;size:16;default:active" json:"status"` // active|closed
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TunnelSession) TableName() string { return "tunnel_sessions" }
