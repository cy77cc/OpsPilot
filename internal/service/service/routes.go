package service

import (
	"github.com/cy77cc/k8s-manage/internal/middleware"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

func RegisterServiceHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHandler(svcCtx)
	g := v1.Group("/services", middleware.JWTAuth())
	{
		g.POST("/render/preview", h.Preview)
		g.POST("/transform", h.Transform)
		g.POST("/variables/extract", h.ExtractVariables)
		g.POST("", h.Create)
		g.GET("", h.List)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
		g.PUT("/:id/visibility", h.UpdateVisibility)
		g.PUT("/:id/grant-teams", h.UpdateGrantedTeams)
		g.GET("/:id/variables/schema", h.GetVariableSchema)
		g.GET("/:id/variables/values", h.GetVariableValues)
		g.PUT("/:id/variables/values", h.UpsertVariableValues)
		g.GET("/:id/revisions", h.ListRevisions)
		g.POST("/:id/revisions", h.CreateRevision)
		g.PUT("/:id/deploy-target", h.UpsertDeployTarget)
		g.GET("/:id/releases", h.ListReleaseRecords)
		g.POST("/:id/deploy/preview", h.DeployPreview)
		g.POST("/:id/deploy", h.Deploy)
		g.POST("/:id/rollback", h.Rollback)
		g.GET("/:id/events", h.Events)
		g.GET("/quota", h.Quota)
		g.POST("/:id/deploy/helm", h.DeployHelm)
		g.POST("/helm/import", h.HelmImport)
		g.POST("/helm/render", h.HelmRender)
	}
}
