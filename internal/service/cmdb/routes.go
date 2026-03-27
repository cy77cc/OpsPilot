// Package cmdb 提供配置管理数据库服务的路由注册。
//
// 本文件注册 CMDB 相关的 HTTP 路由，包括：
//   - 资产管理
//   - 关系管理
//   - 拓扑查询
//   - 同步任务
//   - 变更和审计记录
package cmdb

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	cmdbhandler "github.com/cy77cc/OpsPilot/internal/service/cmdb/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterCMDBHandlers 注册 CMDB 服务路由到 v1 组。
func RegisterCMDBHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := cmdbhandler.NewHandler(svcCtx)
	g := v1.Group("/cmdb", middleware.JWTAuth())
	{
		g.GET("/assets", h.ListAssets)
		g.POST("/assets", h.CreateAsset)
		g.GET("/assets/:id", h.GetAsset)
		g.PUT("/assets/:id", h.UpdateAsset)
		g.DELETE("/assets/:id", h.DeleteAsset)

		g.GET("/relations", h.ListRelations)
		g.POST("/relations", h.CreateRelation)
		g.DELETE("/relations/:id", h.DeleteRelation)

		g.GET("/topology", h.Topology)

		g.POST("/sync/jobs", h.TriggerSync)
		g.GET("/sync/jobs/:id", h.GetSyncJob)
		g.POST("/sync/jobs/:id/retry", h.RetrySyncJob)

		g.GET("/changes", h.ListChanges)
		g.GET("/audits", h.ListAudits)
	}
}
