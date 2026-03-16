// Package dashboard 提供仪表盘概览服务的路由注册。
//
// 本文件注册仪表盘相关的 HTTP 路由，提供系统整体概览数据。
package dashboard

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	dashboardhandler "github.com/cy77cc/OpsPilot/internal/service/dashboard/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterDashboardHandlers 注册仪表盘服务路由到 v1 组。
func RegisterDashboardHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := dashboardhandler.NewHandler(svcCtx)
	g := v1.Group("", middleware.JWTAuth())
	{
		g.GET("/dashboard/overview", h.GetOverview)
		g.GET("/dashboard/overview/v2", h.GetOverviewV2)
	}
}
