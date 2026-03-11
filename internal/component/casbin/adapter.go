// Package adapter 提供 Casbin 权限适配器实现。
//
// 本文件实现基于 GORM 的只读 Casbin 适配器，
// 从数据库加载角色权限策略和用户角色继承关系。
package adapter

import (
	"fmt"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"gorm.io/gorm"
)

// Adapter 是基于 GORM 的 Casbin 适配器。
type Adapter struct {
	db *gorm.DB
}

// NewAdapter 创建 Casbin 适配器实例。
func NewAdapter(db *gorm.DB) *Adapter {
	return &Adapter{db: db}
}

// LoadPolicy 从数据库加载所有策略规则。
//
// 加载两类策略：
//  1. 角色权限策略 (p, role_code, permission_code)
//  2. 用户角色继承 (g, user_id, role_code)
func (a *Adapter) LoadPolicy(model model.Model) error {
	// 1. Load Role Policies (p, role, permission_code)
	// Query: SELECT r.code as role_code, p.code as permission_code FROM roles r JOIN role_permissions rp ON r.id = rp.role_id JOIN permissions p ON p.id = rp.permission_id WHERE r.status = 1 AND p.status = 1
	type PolicyResult struct {
		RoleCode       string
		PermissionCode string
	}
	var policies []PolicyResult
	err := a.db.Table("roles").
		Select("roles.code as role_code, permissions.code as permission_code").
		Joins("JOIN role_permissions ON roles.id = role_permissions.role_id").
		Joins("JOIN permissions ON permissions.id = role_permissions.permission_id").
		Where("roles.status = 1 AND permissions.status = 1").
		Scan(&policies).Error
	if err != nil {
		return err
	}

	for _, policy := range policies {
		persist.LoadPolicyLine(fmt.Sprintf("p, %s, %s", policy.RoleCode, policy.PermissionCode), model)
	}

	// 2. Load User Role Inheritance (g, user_id, role)
	// Query: SELECT u.id, r.code FROM users u JOIN user_roles ur ON u.id = ur.user_id JOIN roles r ON r.id = ur.role_id WHERE u.status = 1 AND r.status = 1
	type GroupResult struct {
		UserID   int64
		RoleCode string
	}
	var groups []GroupResult
	err = a.db.Table("users").
		Select("users.id as user_id, roles.code as role_code").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("users.status = 1 AND roles.status = 1").
		Scan(&groups).Error
	if err != nil {
		return err
	}

	for _, group := range groups {
		persist.LoadPolicyLine(fmt.Sprintf("g, %d, %s", group.UserID, group.RoleCode), model)
	}

	return nil
}

// SavePolicy 保存所有策略规则到存储。
//
// 本适配器为只读模式，不支持此操作。
func (a *Adapter) SavePolicy(model model.Model) error {
	return fmt.Errorf("not implemented: read-only adapter")
}

// AddPolicy 添加策略规则到存储。
//
// 本适配器为只读模式，不支持此操作。
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	return fmt.Errorf("not implemented: read-only adapter")
}

// RemovePolicy 从存储移除策略规则。
//
// 本适配器为只读模式，不支持此操作。
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	return fmt.Errorf("not implemented: read-only adapter")
}

// RemoveFilteredPolicy 移除匹配过滤器的策略规则。
//
// 本适配器为只读模式，不支持此操作。
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	return fmt.Errorf("not implemented: read-only adapter")
}
