// Package automation 提供自动化运维服务的路由注册。
//
// 本文件注册自动化相关的 HTTP 路由，包括：
//   - 清单管理（Inventory）：主机组配置
//   - Playbook 管理：自动化脚本定义
//   - 运行预览和执行：任务执行流程
//   - 执行日志查询：运行状态追踪
//
// 所有路由都需要 JWT 认证，权限控制在 Handler 层实现。
package automation

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	automationhandler "github.com/cy77cc/OpsPilot/internal/service/automation/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterAutomationHandlers 注册自动化服务路由到 v1 组。
//
// 参数:
//   - v1: Gin 路由组，用于注册 /api/v1 下的路由
//   - svcCtx: 服务上下文，包含数据库、配置等依赖
//
// 路由列表:
//   - GET    /automation/inventories   - 获取清单列表
//   - POST   /automation/inventories   - 创建清单
//   - GET    /automation/playbooks     - 获取 Playbook 列表
//   - POST   /automation/playbooks     - 创建 Playbook
//   - POST   /automation/runs/preview  - 预览执行
//   - POST   /automation/runs/execute  - 执行任务
//   - GET    /automation/runs/:id      - 获取执行详情
//   - GET    /automation/runs/:id/logs - 获取执行日志
func RegisterAutomationHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := automationhandler.NewHandler(svcCtx)
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
