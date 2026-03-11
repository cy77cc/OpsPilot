// Package httpx 提供 HTTP 请求处理相关的工具函数。
//
// 本文件实现基于数据库的权限检查功能，
// 用于判断用户是否为管理员或是否拥有特定权限。
package httpx

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// IsAdmin 判断指定用户是否为管理员。
//
// 判断条件（满足任一）：
//   - 用户名为 "admin"（不区分大小写）
//   - 用户持有 code 为 "admin" 的角色（不区分大小写）
func IsAdmin(db *gorm.DB, userID uint64) bool {
	if userID == 0 {
		return false
	}
	var u model.User
	if err := db.Select("id", "username").Where("id = ?", userID).First(&u).Error; err == nil {
		if strings.EqualFold(strings.TrimSpace(u.Username), "admin") {
			return true
		}
	}
	var rows []struct {
		Code string `gorm:"column:code"`
	}
	if err := db.Table("roles").
		Select("roles.code").
		Joins("JOIN user_roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ?", userID).
		Scan(&rows).Error; err != nil {
		return false
	}
	for _, row := range rows {
		if strings.EqualFold(strings.TrimSpace(row.Code), "admin") {
			return true
		}
	}
	return false
}

// HasAnyPermission 判断用户是否拥有指定的任一权限。
//
// 权限检查顺序：
//  1. 管理员拥有所有权限
//  2. 检查通配符权限 "*:*"
//  3. 检查精确匹配的权限
//  4. 检查领域通配符 "<domain>:*"
func HasAnyPermission(db *gorm.DB, userID uint64, codes ...string) bool {
	if userID == 0 {
		return false
	}
	if IsAdmin(db, userID) {
		return true
	}
	var rows []struct {
		Code string `gorm:"column:code"`
	}
	err := db.Table("permissions").
		Select("permissions.code").
		Joins("JOIN role_permissions ON role_permissions.permission_id = permissions.id").
		Joins("JOIN user_roles ON user_roles.role_id = role_permissions.role_id").
		Where("user_roles.user_id = ?", userID).
		Scan(&rows).Error
	if err != nil {
		return false
	}
	set := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		set[strings.TrimSpace(r.Code)] = struct{}{}
	}
	if _, ok := set["*:*"]; ok {
		return true
	}
	for _, code := range codes {
		if _, ok := set[code]; ok {
			return true
		}
		parts := strings.SplitN(code, ":", 2)
		if len(parts) == 2 {
			if _, ok := set[parts[0]+":*"]; ok {
				return true
			}
		}
	}
	return false
}

// Authorize 检查当前用户是否拥有指定的任一权限。
//
// 如果检查失败，写入 Forbidden 响应并返回 false。
//
// 用法:
//
//	if !httpx.Authorize(c, db, "k8s:read") { return }
func Authorize(c *gin.Context, db *gorm.DB, codes ...string) bool {
	uid := UIDFromCtx(c)
	if HasAnyPermission(db, uid, codes...) {
		return true
	}
	Fail(c, xcode.Forbidden, "")
	return false
}
