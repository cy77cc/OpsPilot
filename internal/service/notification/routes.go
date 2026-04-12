// Package notification 提供通知管理服务的路由注册。
//
// routes.go 定义通知模块的 HTTP 路由，顶层目录仅保留此文件作为路由索引。
package notification

import (
	"github.com/cy77cc/OpsPilot/internal/service/notification/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterNotificationHandlers 注册通知相关路由。
//
// 路由列表:
//   - GET  /notifications           - 获取通知列表
//   - GET  /notifications/unread-count - 获取未读数量
//   - POST /notifications/:id/read   - 标记已读
//   - POST /notifications/:id/dismiss - 忽略通知
//   - POST /notifications/:id/confirm - 确认告警
//   - POST /notifications/read-all   - 全部已读
//
// 参数:
//   - r: Gin 路由组
//   - svcCtx: 服务上下文
func RegisterNotificationHandlers(r *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	svc := handler.NewNotificationService(svcCtx)

	notifications := r.Group("/notifications")
	{
		notifications.GET("", svc.ListNotifications)
		notifications.GET("/unread-count", svc.UnreadCount)
		notifications.POST("/:id/read", svc.MarkAsRead)
		notifications.POST("/:id/dismiss", svc.Dismiss)
		notifications.POST("/:id/confirm", svc.Confirm)
		notifications.POST("/read-all", svc.MarkAllAsRead)
	}
}
