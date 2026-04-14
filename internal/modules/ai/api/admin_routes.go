package api

import (
	"github.com/cy77cc/OpsPilot/internal/core/middleware"
	modelhandler "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/api"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterAdminAIHandlers registers admin model management routes.
func RegisterAdminAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := modelhandler.NewHTTPHandler(svcCtx)

	readOnly := middleware.CasbinAuth(nil, "ai:model:read")
	writeOnly := middleware.CasbinAuth(nil, "ai:model:write")
	if svcCtx != nil {
		readOnly = middleware.CasbinAuth(svcCtx.CasbinEnforcer, "ai:model:read")
		writeOnly = middleware.CasbinAuth(svcCtx.CasbinEnforcer, "ai:model:write")
	}

	g := v1.Group("/admin/ai", middleware.JWTAuth())
	models := g.Group("/models")
	{
		models.GET("", readOnly, h.ListModels)
		models.GET("/:id", readOnly, h.GetModel)
		models.POST("", writeOnly, h.CreateModel)
		models.PUT("/:id", writeOnly, h.UpdateModel)
		models.PUT("/:id/default", writeOnly, h.SetDefaultModel)
		models.DELETE("/:id", writeOnly, h.DeleteModel)
		models.POST("/import/preview", readOnly, h.PreviewImport)
		models.POST("/import", writeOnly, h.ImportModels)
	}
}
