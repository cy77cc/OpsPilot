package model

import "time"

// HostRoute 是主机路由表模型，记录主机的连接路由信息。
//
// 表名: host_routes
type HostRoute struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	HostID    uint64    `gorm:"column:host_id;uniqueIndex" json:"host_id"`
	Direct    bool      `gorm:"column:direct;default:true" json:"direct"`
	GatewayID uint64    `gorm:"column:gateway_id;index" json:"gateway_id"`
	TunnelID  string    `gorm:"column:tunnel_id;size:64" json:"tunnel_id"`
	Mode      string    `gorm:"column:mode;size:16" json:"mode"` // tunnel|proxy
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (HostRoute) TableName() string { return "host_routes" }
