// Package topology 提供服务拓扑查询服务的路由注册。
//
// 本文件注册拓扑相关的 HTTP 路由，包括：
//   - 服务拓扑查询
//   - 主机关联服务查询
//   - 集群关联服务查询
//   - 全局拓扑图
package topology

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	topologyhandler "github.com/cy77cc/OpsPilot/internal/service/topology/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterTopologyHandlers 注册拓扑服务路由到 v1 组。
func RegisterTopologyHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := topologyhandler.NewHandler(svcCtx)
	g := v1.Group("/topology", middleware.JWTAuth())
	{
		g.GET("/services/:id", h.ServiceTopology)
		g.GET("/hosts/:id/services", h.HostServices)
		g.GET("/clusters/:id/services", h.ClusterServices)
		g.GET("/graph", h.Graph)
	}
}
