// Package cluster 提供 Kubernetes 集群管理服务的路由注册。
//
// 本文件注册集群相关的 HTTP 路由，包括：
//   - 集群 CRUD 操作
//   - 节点管理
//   - 工作负载查询（Pod、Deployment、StatefulSet 等）
//   - 服务和网络
//   - 配置和存储
//   - 集群引导和导入
package cluster

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterClusterHandlers 注册集群服务路由到 v1 组。
func RegisterClusterHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHandler(svcCtx)

	// 启动集群指标采集器
	collector := NewCollector(svcCtx)
	collector.Start()

	clusterGroup := v1.Group("/clusters", middleware.JWTAuth())
	{
		// 集群 CRUD
		clusterGroup.GET("", h.GetClusters)
		clusterGroup.POST("", h.CreateCluster)
		clusterGroup.GET("/:id", h.GetClusterDetail)
		clusterGroup.PUT("/:id", h.UpdateCluster)
		clusterGroup.DELETE("/:id", h.DeleteCluster)
		clusterGroup.POST("/:id/test", h.TestCluster)

		// 集群节点
		clusterGroup.GET("/:id/nodes", h.GetClusterNodes)
		clusterGroup.POST("/:id/nodes/sync", h.SyncClusterNodesHandler)
		clusterGroup.POST("/:id/nodes", h.AddClusterNodes)
		clusterGroup.GET("/:id/nodes/:name", h.GetNodeDetail)
		clusterGroup.DELETE("/:id/nodes/:name", h.RemoveClusterNode)

		// 命名空间
		clusterGroup.GET("/:id/namespaces", h.GetNamespaces)

		// 工作负载
		clusterGroup.GET("/:id/namespaces/:namespace/pods", h.GetPods)
		clusterGroup.GET("/:id/namespaces/:namespace/deployments", h.GetDeployments)
		clusterGroup.GET("/:id/namespaces/:namespace/statefulsets", h.GetStatefulSets)
		clusterGroup.GET("/:id/namespaces/:namespace/daemonsets", h.GetDaemonSets)
		clusterGroup.GET("/:id/namespaces/:namespace/jobs", h.GetJobs)

		// 服务和网络
		clusterGroup.GET("/:id/namespaces/:namespace/services", h.GetServices)
		clusterGroup.GET("/:id/namespaces/:namespace/ingresses", h.GetIngresses)

		// 配置
		clusterGroup.GET("/:id/namespaces/:namespace/configmaps", h.GetConfigMaps)
		clusterGroup.GET("/:id/namespaces/:namespace/secrets", h.GetSecrets)

		// 存储
		clusterGroup.GET("/:id/pvs", h.GetPVs)
		clusterGroup.GET("/:id/namespaces/:namespace/pvcs", h.GetPVCs)

		// 已部署服务
		clusterGroup.GET("/:id/services", h.GetClusterServices)

		// 高级操作
		clusterGroup.GET("/:id/events", h.GetEvents)
		clusterGroup.GET("/:id/namespaces/:namespace/hpas", h.GetHPAs)
		clusterGroup.GET("/:id/namespaces/:namespace/resourcequotas", h.GetResourceQuotas)
		clusterGroup.GET("/:id/namespaces/:namespace/limitranges", h.GetLimitRanges)
		clusterGroup.GET("/:id/version", h.GetClusterVersion)
		clusterGroup.GET("/:id/certificates", h.GetCertificates)
		clusterGroup.POST("/:id/certificates/renew", h.RenewCertificates)
		clusterGroup.GET("/:id/upgrade-plan", h.GetUpgradePlan)
		clusterGroup.POST("/:id/upgrade", h.UpgradeCluster)

		// 引导（自托管集群创建）
		clusterGroup.GET("/bootstrap/versions", h.GetBootstrapVersions)
		clusterGroup.GET("/bootstrap/profiles", h.ListBootstrapProfiles)
		clusterGroup.POST("/bootstrap/profiles", h.CreateBootstrapProfile)
		clusterGroup.PUT("/bootstrap/profiles/:id", h.UpdateBootstrapProfile)
		clusterGroup.DELETE("/bootstrap/profiles/:id", h.DeleteBootstrapProfile)
		clusterGroup.POST("/bootstrap/preview", h.PreviewBootstrap)
		clusterGroup.POST("/bootstrap/apply", h.ApplyBootstrap)
		clusterGroup.GET("/bootstrap/:task_id", h.GetBootstrapTask)

		// 导入外部集群
		clusterGroup.POST("/import", h.ImportExternalCluster)
		clusterGroup.POST("/import/validate", h.ValidateImport)
	}
}
