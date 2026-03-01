package user

import (
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/middleware"
	userHandler "github.com/cy77cc/k8s-manage/internal/service/user/handler"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

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
