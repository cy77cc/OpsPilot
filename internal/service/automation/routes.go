// Package automation 提供自动化运维服务的路由注册。
//
// 本文件注册自动化相关的 HTTP 路由，包括：
//   - 清单管理（Inventory）
//   - Playbook 管理
//   - 运行预览和执行
//   - 执行日志查询
package automation

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterAutomationHandlers 注册自动化服务路由到 v1 组。
func RegisterAutomationHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHandler(svcCtx)
	g := v1.Group("/automation", middleware.JWTAuth())
	{
		g.GET("/inventories", h.ListInventories)
		g.POST("/inventories", h.CreateInventory)
		g.GET("/playbooks", h.ListPlaybooks)
		g.POST("/playbooks", h.CreatePlaybook)
		g.POST("/runs/preview", h.PreviewRun)
		g.POST("/runs/execute", h.ExecuteRun)
		g.GET("/runs/:id", h.GetRun)
		g.GET("/runs/:id/logs", h.GetRunLogs)
	}
}
