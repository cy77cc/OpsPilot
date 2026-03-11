// Package middleware 提供 HTTP 中间件实现。
//
// 本文件实现 Casbin 权限控制中间件，用于基于角色的访问控制。
// 支持特权角色（admin、super-admin）自动放行。
package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// CasbinAuth 返回 Casbin 权限控制中间件。
//
// 参数:
//   - enforcer: Casbin 执行器
//   - permissionCode: 需要的权限码
//
// 特权角色（admin、super-admin、root）自动放行。
// 拒绝访问时会记录审计日志。
func CasbinAuth(enforcer *casbin.Enforcer, permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if enforcer == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, xcode.NewErrCode(xcode.ServerError))
			return
		}

		uid, exists := c.Get("uid")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, xcode.NewErrCode(xcode.Unauthorized))
			return
		}

		sub := fmt.Sprintf("%v", uid)
		obj := permissionCode

		if isPrivilegedSubject(enforcer, sub) {
			c.Next()
			return
		}

		ok, err := enforcer.Enforce(sub, obj)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, xcode.NewErrCode(xcode.ServerError))
			return
		}

		if !ok {
			auditAccessDenied(c, sub, obj)
			c.AbortWithStatusJSON(http.StatusForbidden, xcode.NewErrCode(xcode.Forbidden))
			return
		}

		c.Next()
	}
}

func auditAccessDenied(c *gin.Context, actor, action string) {
	resource := c.FullPath()
	if resource == "" {
		resource = c.Request.URL.Path
	}
	c.Set("rbac_deny_audit", gin.H{
		"actor":     actor,
		"resource":  resource,
		"action":    action,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
	log.Printf("rbac deny actor=%s resource=%s action=%s timestamp=%s", actor, resource, action, time.Now().UTC().Format(time.RFC3339))
}

func isPrivilegedSubject(enforcer *casbin.Enforcer, subject string) bool {
	roles, err := enforcer.GetRolesForUser(subject)
	if err != nil {
		return false
	}
	for _, role := range roles {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if normalized == "admin" || normalized == "super-admin" || normalized == "super_admin" || normalized == "root" || normalized == "超级管理员" {
			return true
		}
	}
	return false
}
