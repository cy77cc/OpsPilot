// Package model 提供数据库模型定义。
//
// 本文件包含主机探测相关的数据库模型。
package model

import "time"

// HostProbeSession 主机探测会话模型。
//
// 存储一次性的主机探测结果，用于后续创建主机。
// 探测令牌有效期为 10 分钟，只能消费一次。
//
// 表名: host_probe_sessions
type HostProbeSession struct {
	// ID 主键 ID
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	// TokenHash 探测令牌的 SHA256 哈希
	TokenHash string `gorm:"column:token_hash;type:varchar(128);uniqueIndex;not null" json:"-"`

	// Name 主机名称
	Name string `gorm:"column:name;type:varchar(128);not null" json:"name"`

	// IP 主机 IP 地址
	IP string `gorm:"column:ip;type:varchar(64);not null" json:"ip"`

	// Port SSH 端口
	Port int `gorm:"column:port;not null;default:22" json:"port"`

	// AuthType 认证类型 (password/key)
	AuthType string `gorm:"column:auth_type;type:varchar(32);not null" json:"auth_type"`

	// Username SSH 用户名
	Username string `gorm:"column:username;type:varchar(128);not null" json:"username"`

	// SSHKeyID 关联的 SSH 密钥 ID
	SSHKeyID *uint64 `gorm:"column:ssh_key_id" json:"ssh_key_id"`

	// PasswordCipher 密码密文（加密存储）
	PasswordCipher string `gorm:"column:password_cipher;type:text" json:"-"`

	// Reachable 是否可达
	Reachable bool `gorm:"column:reachable;not null;default:false" json:"reachable"`

	// LatencyMS 连接延迟（毫秒）
	LatencyMS int64 `gorm:"column:latency_ms;not null;default:0" json:"latency_ms"`

	// FactsJSON 系统信息 JSON（主机名、操作系统、架构等）
	FactsJSON string `gorm:"column:facts_json;type:longtext" json:"facts_json"`

	// WarningsJSON 警告信息 JSON
	WarningsJSON string `gorm:"column:warnings_json;type:longtext" json:"warnings_json"`

	// ExpiresAt 令牌过期时间
	ExpiresAt time.Time `gorm:"column:expires_at;index" json:"expires_at"`

	// ConsumedAt 令牌消费时间
	ConsumedAt *time.Time `gorm:"column:consumed_at" json:"consumed_at"`

	// CreatedBy 创建者用户 ID
	CreatedBy uint64 `gorm:"column:created_by;index" json:"created_by"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 返回表名。
func (HostProbeSession) TableName() string { return "host_probe_sessions" }
