// Package rbac 提供基于角色的访问控制服务的路由注册。
//
// 本文件注册 RBAC 相关的 HTTP 路由，包括：
//   - 用户管理
//   - 角色管理
//   - 权限管理
//   - 权限检查
//   - 迁移事件记录
package rbac

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	rbachandler "github.com/cy77cc/OpsPilot/internal/service/rbac/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterRBACHandlers 注册 RBAC 服务路由到 v1 组。
func RegisterRBACHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := rbachandler.NewHandler(svcCtx)
	g := v1.Group("/rbac", middleware.JWTAuth())
	readOnly := middleware.CasbinAuth(svcCtx.CasbinEnforcer, "rbac:read")
	writeOnly := middleware.CasbinAuth(svcCtx.CasbinEnforcer, "rbac:write")
	{
		g.GET("/me/permissions", h.MyPermissions)
		g.POST("/check", readOnly, h.Check)
		g.GET("/users", readOnly, h.ListUsers)
		g.GET("/users/:id", readOnly, h.GetUser)
		g.POST("/users", writeOnly, h.CreateUser)
		g.PUT("/users/:id", writeOnly, h.UpdateUser)
		g.DELETE("/users/:id", writeOnly, h.DeleteUser)
		g.GET("/roles", readOnly, h.ListRoles)
		g.GET("/roles/:id", readOnly, h.GetRole)
		g.POST("/roles", writeOnly, h.CreateRole)
		g.PUT("/roles/:id", writeOnly, h.UpdateRole)
		g.DELETE("/roles/:id", writeOnly, h.DeleteRole)
		g.GET("/permissions", readOnly, h.ListPermissions)
		g.GET("/permissions/:id", readOnly, h.GetPermission)
		g.POST("/migration/events", readOnly, h.RecordMigrationEvent)
	}
}
