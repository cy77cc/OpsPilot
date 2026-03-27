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
	monitoringhandler "github.com/cy77cc/OpsPilot/internal/service/monitoring/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterMonitoringHandlers 注册监控服务路由到 v1 组。
//
// 注册以下路由:
//   - POST /alerts/receiver: Alertmanager Webhook 接收 (无需认证)
//   - GET /alerts: 获取告警事件列表
//   - GET /alert-rules: 获取告警规则列表
//   - POST /alert-rules: 创建告警规则
//   - PUT /alert-rules/:id: 更新告警规则
//   - POST /alert-rules/:id/enable: 启用告警规则
//   - POST /alert-rules/:id/disable: 禁用告警规则
//   - POST /alerts/rules/sync: 手动同步规则
//   - GET /metrics: 查询指标数据
//   - GET /alert-channels: 获取通知渠道列表
//   - POST /alert-channels: 创建通知渠道
//   - PUT /alert-channels/:id: 更新通知渠道
//   - GET /alert-deliveries: 获取投递记录列表
//
// 参数:
//   - v1: Gin 路由组，所有路由将注册到此组下
//   - svcCtx: 服务上下文，包含数据库连接、配置等依赖
func RegisterMonitoringHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := monitoringhandler.NewHandler(svcCtx)
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
