package cluster

import (
	"github.com/cy77cc/k8s-manage/internal/middleware"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterClusterHandlers registers cluster routes
func RegisterClusterHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHandler(svcCtx)

	clusterGroup := v1.Group("/clusters", middleware.JWTAuth())
	{
		// Cluster CRUD
		clusterGroup.GET("", h.GetClusters)
		clusterGroup.POST("", h.CreateCluster)
		clusterGroup.GET("/:id", h.GetClusterDetail)
		clusterGroup.PUT("/:id", h.UpdateCluster)
		clusterGroup.DELETE("/:id", h.DeleteCluster)
		clusterGroup.POST("/:id/test", h.TestCluster)

		// Cluster nodes
		clusterGroup.GET("/:id/nodes", h.GetClusterNodes)
		clusterGroup.POST("/:id/nodes/sync", h.SyncClusterNodesHandler)
		clusterGroup.POST("/:id/nodes", h.AddClusterNodes)
		clusterGroup.GET("/:id/nodes/:name", h.GetNodeDetail)
		clusterGroup.DELETE("/:id/nodes/:name", h.RemoveClusterNode)

		// Namespaces
		clusterGroup.GET("/:id/namespaces", h.GetNamespaces)

		// Workloads
		clusterGroup.GET("/:id/namespaces/:namespace/pods", h.GetPods)
		clusterGroup.GET("/:id/namespaces/:namespace/deployments", h.GetDeployments)
		clusterGroup.GET("/:id/namespaces/:namespace/statefulsets", h.GetStatefulSets)
		clusterGroup.GET("/:id/namespaces/:namespace/daemonsets", h.GetDaemonSets)
		clusterGroup.GET("/:id/namespaces/:namespace/jobs", h.GetJobs)

		// Services and networking
		clusterGroup.GET("/:id/namespaces/:namespace/services", h.GetServices)
		clusterGroup.GET("/:id/namespaces/:namespace/ingresses", h.GetIngresses)

		// Config
		clusterGroup.GET("/:id/namespaces/:namespace/configmaps", h.GetConfigMaps)
		clusterGroup.GET("/:id/namespaces/:namespace/secrets", h.GetSecrets)

		// Storage
		clusterGroup.GET("/:id/pvs", h.GetPVs)
		clusterGroup.GET("/:id/namespaces/:namespace/pvcs", h.GetPVCs)

		// Deployed services
		clusterGroup.GET("/:id/services", h.GetClusterServices)

		// Advanced operations
		clusterGroup.GET("/:id/events", h.GetEvents)
		clusterGroup.GET("/:id/namespaces/:namespace/hpas", h.GetHPAs)
		clusterGroup.GET("/:id/namespaces/:namespace/resourcequotas", h.GetResourceQuotas)
		clusterGroup.GET("/:id/namespaces/:namespace/limitranges", h.GetLimitRanges)
		clusterGroup.GET("/:id/version", h.GetClusterVersion)
		clusterGroup.GET("/:id/certificates", h.GetCertificates)
		clusterGroup.POST("/:id/certificates/renew", h.RenewCertificates)
		clusterGroup.GET("/:id/upgrade-plan", h.GetUpgradePlan)
		clusterGroup.POST("/:id/upgrade", h.UpgradeCluster)

		// Bootstrap (self-hosted cluster creation)
		clusterGroup.GET("/bootstrap/versions", h.GetBootstrapVersions)
		clusterGroup.GET("/bootstrap/profiles", h.ListBootstrapProfiles)
		clusterGroup.POST("/bootstrap/profiles", h.CreateBootstrapProfile)
		clusterGroup.PUT("/bootstrap/profiles/:id", h.UpdateBootstrapProfile)
		clusterGroup.DELETE("/bootstrap/profiles/:id", h.DeleteBootstrapProfile)
		clusterGroup.POST("/bootstrap/preview", h.PreviewBootstrap)
		clusterGroup.POST("/bootstrap/apply", h.ApplyBootstrap)
		clusterGroup.GET("/bootstrap/:task_id", h.GetBootstrapTask)

		// Import external cluster
		clusterGroup.POST("/import", h.ImportExternalCluster)
		clusterGroup.POST("/import/validate", h.ValidateImport)
	}
}
