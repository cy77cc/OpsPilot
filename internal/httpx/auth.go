package httpx

import (
	"strings"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// IsAdmin reports whether the given user is an administrator.
// A user is considered admin if their username equals "admin" (case-insensitive)
// or if they hold a role with code "admin" (case-insensitive).
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

// HasAnyPermission reports whether the user holds at least one of the given permission codes.
// Admin users always return true. Supports "*:*" wildcard and "<domain>:*" domain wildcard.
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

// Authorize checks that the current user has at least one of the given permission codes.
// If the check fails, it writes a Forbidden response and returns false.
// Usage: if !httpx.Authorize(c, db, "k8s:read") { return }
func Authorize(c *gin.Context, db *gorm.DB, codes ...string) bool {
	uid := UIDFromCtx(c)
	if HasAnyPermission(db, uid, codes...) {
		return true
	}
	Fail(c, xcode.Forbidden, "")
	return false
}
