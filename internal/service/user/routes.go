// Package user 提供用户认证服务的路由注册。
//
// 本文件注册用户相关的 HTTP 路由，包括：
//   - 用户登录、登出、注册
//   - Token 刷新
//   - 用户信息查询
package user

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/middleware"
	userHandler "github.com/cy77cc/OpsPilot/internal/service/user/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterUserHandlers 注册用户服务路由。
func RegisterUserHandlers(r *gin.RouterGroup, serverCtx *svc.ServiceContext) {
	// 无需认证的组
	authGroup := r.Group("auth")

	userHandler := userHandler.NewUserHandler(serverCtx)

	{
		authGroup.POST("login", userHandler.Login)
		authGroup.POST("logout", userHandler.Logout)
		authGroup.POST("refresh", userHandler.Refresh)
		authGroup.POST("register", userHandler.Register)
		authGroup.GET("me", middleware.JWTAuth(), userHandler.Me)
	}

	userGroup := r.Group("user", middleware.JWTAuth())
	{
		userGroup.POST("/", middleware.CasbinAuth(serverCtx.CasbinEnforcer, "user:view"), func(c *gin.Context) {
			httpx.OK(c, nil)
		})
		userGroup.GET("/:id", userHandler.GetUserInfo)
	}
}
