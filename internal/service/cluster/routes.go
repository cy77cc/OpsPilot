package cluster

import (
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers cluster routes
func RegisterRoutes(r *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	handler := NewHandler(svcCtx)

	clusterGroup := r.Group("/clusters")
	{
		clusterGroup.GET("", handler.GetClusters)
		clusterGroup.GET("/:id", handler.GetClusterDetail)
		clusterGroup.GET("/:id/nodes", handler.GetClusterNodes)
	}
}
