// Package userapi 提供用户认证服务的路由注册。
//
// 本文件注册用户相关的 HTTP 路由，包括：
//   - 用户登录、登出、注册
//   - Token 刷新
//   - 用户信息查询
package userapi

import (
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/middleware"
	userHandler "github.com/cy77cc/OpsPilot/internal/modules/user/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterUserHandlers 注册用户服务路由。
func RegisterUserHandlers(r *gin.RouterGroup, serverCtx *svc.ServiceContext) {
	// 无需认证的组
	authGroup := r.Group("auth")

	uHandler := userHandler.NewUserHandler(serverCtx)

	{
		authGroup.POST("login", uHandler.Login)
		authGroup.POST("logout", uHandler.Logout)
		authGroup.POST("refresh", uHandler.Refresh)
		authGroup.POST("register", uHandler.Register)
		authGroup.GET("me", middleware.JWTAuth(), uHandler.Me)
	}

	userGroup := r.Group("user", middleware.JWTAuth())
	{
		userGroup.GET("/list", uHandler.ListUsers)
		userGroup.POST("/", middleware.CasbinAuth(serverCtx.CasbinEnforcer, "user:view"), func(c *gin.Context) {
			httpx.OK(c, nil)
		})
		userGroup.GET("/:id", uHandler.GetUserInfo)
	}

	orgHandler := userHandler.NewOrgHandler(serverCtx)
	orgGroup := r.Group("org", middleware.JWTAuth())
	{
		orgGroup.GET("/departments/tree", orgHandler.GetDepartmentTree)
		orgGroup.POST("/departments", orgHandler.CreateDepartment)
		orgGroup.PUT("/departments/:id", orgHandler.UpdateDepartment)
		orgGroup.DELETE("/departments/:id", orgHandler.DeleteDepartment)
		orgGroup.GET("/departments/:id/members", orgHandler.GetDepartmentMembers)
		orgGroup.GET("/departments/:id/roles", orgHandler.GetDepartmentRoles)
		orgGroup.POST("/departments/:id/roles", orgHandler.UpdateDepartmentRoles)
		orgGroup.POST("/members/transfer", orgHandler.TransferMember)
	}
}
