// Package cicd 提供 CI/CD 持续集成部署服务的路由注册。
//
// 本文件注册 CI/CD 相关的 HTTP 路由，包括：
//   - CI 配置管理
//   - CI 运行触发和查询
//   - CD 配置管理
//   - 发布管理和审批
//   - 服务时间线
//   - 审计事件
package cicd

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	cicdhandler "github.com/cy77cc/OpsPilot/internal/service/cicd/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterCICDHandlers 注册 CI/CD 服务路由到 v1 组。
func RegisterCICDHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := cicdhandler.NewHandler(svcCtx)
	g := v1.Group("/cicd", middleware.JWTAuth())
	{
		g.GET("/services/:service_id/ci-config", h.GetServiceCIConfig)
		g.PUT("/services/:service_id/ci-config", h.PutServiceCIConfig)
		g.DELETE("/services/:service_id/ci-config", h.DeleteServiceCIConfig)
		g.POST("/services/:service_id/ci-runs/trigger", h.TriggerCIRun)
		g.GET("/services/:service_id/ci-runs", h.ListCIRuns)

		g.GET("/deployments/:deployment_id/cd-config", h.GetDeploymentCDConfig)
		g.PUT("/deployments/:deployment_id/cd-config", h.PutDeploymentCDConfig)

		g.POST("/releases", h.TriggerRelease)
		g.GET("/releases", h.ListReleases)
		g.POST("/releases/:id/approve", h.ApproveRelease)
		g.POST("/releases/:id/reject", h.RejectRelease)
		g.POST("/releases/:id/rollback", h.RollbackRelease)
		g.GET("/releases/:id/approvals", h.ListApprovals)

		g.GET("/services/:service_id/timeline", h.ServiceTimeline)
		g.GET("/audits", h.ListAuditEvents)
	}
}
