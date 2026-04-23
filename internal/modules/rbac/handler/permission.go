// Package handler 提供 RBAC 服务的 HTTP 处理器。
//
// 本文件实现基于角色的访问控制 (RBAC) 相关的 HTTP 接口，包括：
//   - 用户管理: CRUD 操作
//   - 角色管理: CRUD 操作
//   - 权限管理: 查询操作
//   - 权限检查: 单点权限验证
//   - 迁移事件记录: 审计日志
package handler

import (
	"fmt"
	"strings"

	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

// Handler 是 RBAC 服务的 HTTP 处理器。
//
// 依赖:
//   - svcCtx: 服务上下文，包含数据库、Casbin 执行器等依赖
type Handler struct {
	svcCtx *svc.ServiceContext
}

// NewHandler 创建 RBAC 处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: RBAC 处理器实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler { return &Handler{svcCtx: svcCtx} }

// codeValidationError 是代码验证错误。
//
// 用于在同步角色/权限时报告无效的代码值。
type codeValidationError struct {
	field string   // 字段名 (roles/permissions)
	codes []string // 无效的代码列表
}

// Error 实现错误接口。
func (e *codeValidationError) Error() string {
	return fmt.Sprintf("invalid %s values: %s", e.field, strings.Join(e.codes, ","))
}

// fetchPermissionsByUserID 根据用户 ID 获取权限代码列表。
func (h *Handler) fetchPermissionsByUserID(userID uint64) ([]string, error) {
	type row struct {
		Code string `gorm:"column:code"`
	}
	var rows []row
	err := h.svcCtx.DB.Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Code)
	}
	return out, nil
}

// hasPermission 检查权限列表中是否包含指定权限。
func hasPermission(perms []string, code string, resource string) bool {
	resourceWildcard := resource + ":*"
	for _, p := range perms {
		if p == code || p == resourceWildcard || p == "*:*" {
			return true
		}
	}
	return false
}

// mergePermissions 合并权限列表并去重。
//
// 参数:
//   - base: 基础权限列表
//   - extras: 额外权限列表
//
// 返回: 合并后的权限列表 (已去重)
func mergePermissions(base []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(extras))
	merged := make([]string, 0, len(base)+len(extras))
	for _, p := range base {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		merged = append(merged, p)
	}
	for _, p := range extras {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		merged = append(merged, p)
	}
	return merged
}

// adminPermissionSet 返回管理员权限集合。
//
// 管理员拥有所有资源的完全访问权限。
//
// 返回: 管理员权限代码列表
func adminPermissionSet() []string {
	return []string{
		"*:*",
		"host:read", "host:write", "host:*",
		"task:read", "task:write", "task:*",
		"kubernetes:read", "kubernetes:write", "kubernetes:*",
		"monitoring:read", "monitoring:write", "monitoring:*",
		"config:read", "config:write", "config:*",
		"rbac:read", "rbac:write", "rbac:*",
		"ai:approval:read", "ai:approval:write",
		"ai:alert:read", "ai:alert:write",
		"automation:*",
		"cicd:*",
		"cmdb:*",
	}
}

// getRoleCodesByUserID 根据用户 ID 获取角色代码列表。
//
// 参数:
//   - userID: 用户 ID
//
// 返回:
//   - []string: 角色代码列表 (已去重)
//   - error: 查询失败时返回错误
func (h *Handler) getRoleCodesByUserID(userID uint64) ([]string, error) {
	type row struct {
		Code string `gorm:"column:code"`
	}
	var rows []row
	err := h.svcCtx.DB.Table("roles").
		Select("roles.code").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		code := strings.TrimSpace(r.Code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out, nil
}

// getPermissionCodesByRoleID 根据角色 ID 获取权限代码列表。
//
// 参数:
//   - roleID: 角色 ID
//
// 返回:
//   - []string: 权限代码列表 (已去重)
//   - error: 查询失败时返回错误
func (h *Handler) getPermissionCodesByRoleID(roleID uint64) ([]string, error) {
	type row struct {
		Code string `gorm:"column:code"`
	}
	var rows []row
	err := h.svcCtx.DB.Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Where("role_permissions.role_id = ?", roleID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		code := strings.TrimSpace(r.Code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out, nil
}

// syncUserRolesTx 在事务中同步用户角色。
//
// 参数:
//   - tx: 数据库事务
//   - userID: 用户 ID
//   - roleCodes: 角色代码列表
//
// 返回:
//   - error: 同步失败时返回错误 (包括角色不存在的情况)
//
// 流程:
//  1. 删除用户现有角色关联
//  2. 验证角色代码是否有效
//  3. 创建新的用户角色关联
func (h *Handler) syncUserRolesTx(tx *gorm.DB, userID uint64, roleCodes []string) error {
	if err := tx.Where("user_id = ?", userID).Delete(&usermodel.UserRole{}).Error; err != nil {
		return err
	}
	cleanCodes := make([]string, 0, len(roleCodes))
	seen := make(map[string]struct{}, len(roleCodes))
	for _, code := range roleCodes {
		v := strings.TrimSpace(code)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		cleanCodes = append(cleanCodes, v)
	}
	if len(cleanCodes) == 0 {
		return nil
	}
	var roles []usermodel.Role
	if err := tx.Where("code IN ?", cleanCodes).Find(&roles).Error; err != nil {
		return err
	}
	if len(roles) != len(cleanCodes) {
		found := make(map[string]struct{}, len(roles))
		for _, role := range roles {
			found[strings.TrimSpace(role.Code)] = struct{}{}
		}
		missing := make([]string, 0)
		for _, code := range cleanCodes {
			if _, ok := found[code]; !ok {
				missing = append(missing, code)
			}
		}
		return &codeValidationError{field: "roles", codes: missing}
	}
	for _, role := range roles {
		if err := tx.Create(&usermodel.UserRole{UserID: int64(userID), RoleID: int64(role.ID)}).Error; err != nil {
			return err
		}
	}
	return nil
}

// syncRolePermissionsTx 在事务中同步角色权限。
//
// 参数:
//   - tx: 数据库事务
//   - roleID: 角色 ID
//   - permissionCodes: 权限代码列表
//
// 返回:
//   - error: 同步失败时返回错误 (包括权限不存在的情况)
//
// 流程:
//  1. 删除角色现有权限关联
//  2. 验证权限代码是否有效
//  3. 创建新的角色权限关联
func (h *Handler) syncRolePermissionsTx(tx *gorm.DB, roleID uint64, permissionCodes []string) error {
	if err := tx.Where("role_id = ?", roleID).Delete(&usermodel.RolePermission{}).Error; err != nil {
		return err
	}
	cleanCodes := make([]string, 0, len(permissionCodes))
	seen := make(map[string]struct{}, len(permissionCodes))
	for _, code := range permissionCodes {
		v := strings.TrimSpace(code)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		cleanCodes = append(cleanCodes, v)
	}
	if len(cleanCodes) == 0 {
		return nil
	}
	var perms []usermodel.Permission
	if err := tx.Where("code IN ?", cleanCodes).Find(&perms).Error; err != nil {
		return err
	}
	if len(perms) != len(cleanCodes) {
		found := make(map[string]struct{}, len(perms))
		for _, permission := range perms {
			found[strings.TrimSpace(permission.Code)] = struct{}{}
		}
		missing := make([]string, 0)
		for _, code := range cleanCodes {
			if _, ok := found[code]; !ok {
				missing = append(missing, code)
			}
		}
		return &codeValidationError{field: "permissions", codes: missing}
	}
	for _, perm := range perms {
		if err := tx.Create(&usermodel.RolePermission{RoleID: int64(roleID), PermissionID: int64(perm.ID)}).Error; err != nil {
			return err
		}
	}
	return nil
}

// toStatusText 将状态码转换为文本。
//
// 参数:
//   - status: 状态码 (0=禁用, 1=启用)
//
// 返回: 状态文本 ("disabled"/"active")
func toStatusText(status int8) string {
	if status == 1 {
		return "active"
	}
	return "disabled"
}

// toStatusInt 将状态文本转换为状态码。
//
// 参数:
//   - status: 状态文本 ("disabled"/"active")
//
// 返回: 状态码 (0=禁用, 1=启用)
func toStatusInt(status string) int8 {
	if strings.EqualFold(strings.TrimSpace(status), "disabled") {
		return 0
	}
	return 1
}
