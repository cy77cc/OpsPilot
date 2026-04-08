// Package model 提供数据库模型定义。
package model

import "time"

// SSHCredentialTemplate 是 SSH 认证预设模板表模型。
//
// 表名: ssh_credential_templates
// 用途: 为云主机导入提供快速认证配置
//
// 认证类型:
//   - password: 密码认证，密码加密存储
//   - key: 密钥认证，通过 ssh_key_id 引用 SSHKey
type SSHCredentialTemplate struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                            // 预设 ID
	Name        string    `gorm:"column:name;type:varchar(64);not null;uniqueIndex" json:"name"`            // 预设名称（唯一）
	AuthType    string    `gorm:"column:auth_type;type:varchar(16);not null;default:'password'" json:"auth_type"` // 认证类型: password/key
	SSHUser     string    `gorm:"column:ssh_user;type:varchar(32);not null;default:'root'" json:"ssh_user"` // SSH 用户名
	Port        int       `gorm:"column:port;not null;default:22" json:"port"`                              // SSH 端口
	Password    string    `gorm:"column:password;type:text" json:"-"`                                       // SSH 密码（加密存储）
	SSHKeyID    *uint64   `gorm:"column:ssh_key_id" json:"ssh_key_id"`                                      // SSH 密钥 ID（密钥认证）
	Description string    `gorm:"column:description;type:text" json:"description"`                          // 描述
	CreatedBy   uint64    `gorm:"column:created_by;not null" json:"created_by"`                             // 创建人 ID
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                       // 创建时间
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                       // 更新时间
}

// TableName 返回 SSH 认证预设模板表名。
func (SSHCredentialTemplate) TableName() string {
	return "ssh_credential_templates"
}