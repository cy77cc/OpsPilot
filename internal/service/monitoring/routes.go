// Package monitoring 提供监控和告警服务的路由注册。
//
// 本文件注册监控相关的 HTTP 路由，包括：
//   - 告警管理和规则配置
//   - 指标查询
//   - 告警渠道管理
//   - 告警投递记录
//   - Alertmanager Webhook 接收
package monitoring

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterMonitoringHandlers 注册监控服务路由到 v1 组。
func RegisterMonitoringHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHandler(svcCtx)
	h.StartCollector()
	h.StartRuleSync()

	// Alertmanager webhook endpoint (internal call, no JWT).
	v1.POST("/alerts/receiver", h.ReceiveWebhook)

	g := v1.Group("", middleware.JWTAuth())
	{
		g.GET("/alerts", h.ListAlerts)
		g.GET("/alert-rules", h.ListRules)
		g.POST("/alert-rules", h.CreateRule)
		g.PUT("/alert-rules/:id", h.UpdateRule)
		g.POST("/alert-rules/:id/enable", h.EnableRule)
		g.POST("/alert-rules/:id/disable", h.DisableRule)
		g.POST("/alerts/rules/sync", h.SyncRules)
		g.GET("/metrics", h.GetMetrics)
		g.GET("/alert-channels", h.ListChannels)
		g.POST("/alert-channels", h.CreateChannel)
		g.PUT("/alert-channels/:id", h.UpdateChannel)
		g.GET("/alert-deliveries", h.ListDeliveries)
	}
}
