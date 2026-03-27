// Package deployment 提供部署管理服务的路由注册。
//
// 本文件注册部署相关的 HTTP 路由，包括：
//   - 部署目标管理
//   - 发布管理和审批
//   - 集群引导
//   - 凭证管理
//   - 审计日志和指标统计
//   - 部署拓扑和策略
package deployment

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterDeploymentHandlers 注册部署服务路由到 v1 组。
func RegisterDeploymentHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHandler(svcCtx)
	auditH := NewAuditHandler(svcCtx)
	metricsH := NewMetricsHandler(svcCtx)
	topologyH := NewTopologyHandler(svcCtx)
	policyH := NewPolicyHandler(svcCtx)
	g := v1.Group("/deploy", middleware.JWTAuth())
	{
		g.GET("/targets", h.ListTargets)
		g.POST("/targets", h.CreateTarget)
		g.GET("/targets/:id", h.GetTarget)
		g.PUT("/targets/:id", h.UpdateTarget)
		g.DELETE("/targets/:id", h.DeleteTarget)
		g.PUT("/targets/:id/nodes", h.PutTargetNodes)

		g.POST("/releases/preview", h.PreviewRelease)
		g.POST("/releases/apply", h.ApplyRelease)
		g.POST("/releases/:id/approve", h.ApproveRelease)
		g.POST("/releases/:id/reject", h.RejectRelease)
		g.POST("/releases/:id/rollback", h.RollbackRelease)
		g.GET("/releases", h.ListReleases)
		g.GET("/releases/:id", h.GetRelease)
		g.GET("/releases/:id/timeline", h.ListReleaseTimeline)

		g.POST("/clusters/bootstrap/preview", h.PreviewClusterBootstrap)
		g.POST("/clusters/bootstrap/apply", h.ApplyClusterBootstrap)
		g.GET("/clusters/bootstrap/:task_id", h.GetClusterBootstrapTask)
		g.POST("/environments/bootstrap", h.StartEnvironmentBootstrap)
		g.GET("/environments/bootstrap/:job_id", h.GetEnvironmentBootstrapJob)
		g.POST("/credentials/platform/register", h.RegisterPlatformCredential)
		g.POST("/credentials/import", h.ImportExternalCredential)
		g.POST("/credentials/:id/test", h.TestCredential)
		g.GET("/credentials", h.ListCredentials)

		// 审计日志
		g.GET("/audit-logs", auditH.ListAuditLogs)

		// 指标统计
		g.GET("/metrics/summary", metricsH.GetMetricsSummary)
		g.GET("/metrics/trends", metricsH.GetMetricsTrends)

		// 部署拓扑
		g.GET("/topology", topologyH.GetTopology)

		// 策略管理
		g.GET("/policies", policyH.ListPolicies)
		g.GET("/policies/:id", policyH.GetPolicy)
		g.POST("/policies", policyH.CreatePolicy)
		g.PUT("/policies/:id", policyH.UpdatePolicy)
		g.DELETE("/policies/:id", policyH.DeletePolicy)
	}

	sg := v1.Group("/services", middleware.JWTAuth())
	{
		sg.GET("/:id/governance", h.GetGovernance)
		sg.PUT("/:id/governance", h.PutGovernance)
	}
}
