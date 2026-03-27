// Package model 提供数据库模型定义。
//
// 本文件定义用户、角色、权限相关的数据模型，实现 RBAC 权限控制体系。
package model

import (
	"github.com/gookit/validate"
	"gorm.io/gorm"
)

// UserID 是用户 ID 类型别名，便于类型安全和代码可读性。
type UserID int64

// User 是用户表模型，存储系统用户信息。
//
// 表名: users
// 关联:
//   - UserRole (一对多，通过 user_id)
//   - 不使用数据库外键，关联关系由业务层维护
//
// 状态说明:
//   - 1: 正常
//   - 0: 禁用
type User struct {
	ID            UserID `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                     // 用户 ID
	Username      string `gorm:"column:username;type:varchar(64);not null;unique" json:"username" validate:"required|minLen:7"` // 用户名 (唯一，最少7字符)
	PasswordHash  string `gorm:"column:password_hash;type:varchar(255);not null" json:"password_hash"` // 密码哈希 (bcrypt)
	Email         string `gorm:"column:email;type:varchar(128);not null;default:''" json:"email" validate:"email"` // 邮箱地址
	Phone         string `gorm:"column:phone;type:varchar(32);not null;default:''" json:"phone" validate:"phone"` // 手机号码
	Avatar        string `gorm:"column:avatar;type:varchar(255);not null;default:''" json:"avatar"` // 头像 URL
	Status        int8   `gorm:"column:status;not null;default:1" json:"status"`                   // 状态: 1=正常, 0=禁用
	CreateTime    int64  `gorm:"column:create_time;not null;default:0;autoCreateTime" json:"create_time"` // 创建时间 (Unix 时间戳)
	UpdateTime    int64  `gorm:"column:update_time;not null;default:0;autoUpdateTime" json:"update_time"` // 更新时间 (Unix 时间戳)
	LastLoginTime int64  `gorm:"column:last_login_time;not null;default:0" json:"last_login_time"` // 最后登录时间 (Unix 时间戳)
}

// TableName 返回用户表名。
func (User) TableName() string {
	return "users"
}

// BeforeSave 是 GORM 钩子，保存前验证用户数据。
//
// 参数:
//   - tx: GORM 数据库事务
//
// 返回: 验证失败返回错误，成功返回 nil
func (u *User) BeforeSave(tx *gorm.DB) (err error) {
	v := validate.Struct(u)
	if !v.Validate() {
		return v.Errors
	}
	return nil
}

// UserRole 是用户角色关联表模型，实现用户与角色的多对多关系。
//
// 表名: user_roles
// 关联:
//   - User (多对一，通过 user_id)
//   - Role (多对一，通过 role_id)
type UserRole struct {
	ID     UserID `gorm:"column:id;primaryKey;autoIncrement" json:"id"` // 关联 ID
	UserID int64  `gorm:"column:user_id;not null" json:"user_id"`       // 用户 ID
	RoleID int64  `gorm:"column:role_id;not null" json:"role_id"`       // 角色 ID
}

// TableName 返回用户角色关联表名。
func (UserRole) TableName() string {
	return "user_roles"
}

// Role 是角色表模型，定义系统角色。
//
// 表名: roles
// 关联:
//   - UserRole (一对多，通过 role_id)
//   - RolePermission (一对多，通过 role_id)
//
// 状态说明:
//   - 1: 正常
//   - 0: 禁用
type Role struct {
	ID          UserID `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                       // 角色 ID
	Name        string `gorm:"column:name;type:varchar(64);not null;default:''" json:"name"`       // 角色名称 (显示名)
	Code        string `gorm:"column:code;type:varchar(64);not null;unique;default:''" json:"code"` // 角色编码 (唯一，如: admin, operator)
	Description string `gorm:"column:description;type:varchar(255);not null;default:''" json:"description"` // 角色描述
	Status      int8   `gorm:"column:status;not null;default:1" json:"status"`                     // 状态: 1=正常, 0=禁用
	CreateTime  int64  `gorm:"column:create_time;not null;default:0;autoCreateTime" json:"create_time"` // 创建时间 (Unix 时间戳)
	UpdateTime  int64  `gorm:"column:update_time;not null;default:0;autoUpdateTime" json:"update_time"` // 更新时间 (Unix 时间戳)
}

// TableName 返回角色表名。
func (Role) TableName() string {
	return "roles"
}

// RolePermission 是角色权限关联表模型，实现角色与权限的多对多关系。
//
// 表名: role_permissions
// 关联:
//   - Role (多对一，通过 role_id)
//   - Permission (多对一，通过 permission_id)
type RolePermission struct {
	ID           UserID `gorm:"column:id;primaryKey;autoIncrement" json:"id"` // 关联 ID
	RoleID       int64  `gorm:"column:role_id;not null" json:"role_id"`       // 角色 ID
	PermissionID int64  `gorm:"column:permission_id;not null" json:"permission_id"` // 权限 ID
}

// TableName 返回角色权限关联表名。
func (RolePermission) TableName() string {
	return "role_permissions"
}

// Permission 是权限表模型，定义系统权限。
//
// 表名: permissions
// 关联:
//   - RolePermission (一对多，通过 permission_id)
//
// 权限类型:
//   - 0: 菜单权限 (控制菜单显示)
//   - 1: 操作权限 (控制按钮/操作)
//   - 2: 数据权限 (控制数据范围)
//
// 资源格式: 模块:资源 (如: host:list, deployment:create)
type Permission struct {
	ID          UserID `gorm:"column:id;primaryKey;autoIncrement" json:"id"`                       // 权限 ID
	Name        string `gorm:"column:name;type:varchar(64);not null;default:''" json:"name"`       // 权限名称 (显示名)
	Code        string `gorm:"column:code;type:varchar(128);not null;unique;default:''" json:"code"` // 权限编码 (唯一，如: host:list)
	Type        int8   `gorm:"column:type;not null;default:0" json:"type"`                         // 权限类型: 0=菜单, 1=操作, 2=数据
	Resource    string `gorm:"column:resource;type:varchar(255);not null;default:''" json:"resource"` // 资源标识 (如: host, deployment)
	Action      string `gorm:"column:action;type:varchar(32);not null;default:''" json:"action"`   // 操作类型 (如: list, create, update, delete)
	Description string `gorm:"column:description;type:varchar(255);not null;default:''" json:"description"` // 权限描述
	Status      int8   `gorm:"column:status;not null;default:1" json:"status"`                     // 状态: 1=正常, 0=禁用
	CreateTime  int64  `gorm:"column:create_time;not null;default:0;autoCreateTime" json:"create_time"` // 创建时间 (Unix 时间戳)
	UpdateTime  int64  `gorm:"column:update_time;not null;default:0;autoUpdateTime" json:"update_time"` // 更新时间 (Unix 时间戳)
}

// TableName 返回权限表名。
func (Permission) TableName() string {
	return "permissions"
}
