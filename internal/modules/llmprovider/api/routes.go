// Package api 提供 LLM Provider 的路由注册。
package api

import (
	"github.com/cy77cc/OpsPilot/internal/core/middleware"
	"github.com/cy77cc/OpsPilot/internal/modules/llmprovider/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HTTPHandler 导出 HTTP Handler 供外部使用。
type HTTPHandler = handler.HTTPHandler

// NewHTTPHandler 创建 HTTP Handler 实例。
func NewHTTPHandler(svcCtx *svc.ServiceContext) *HTTPHandler {
	return handler.NewHTTPHandler(svcCtx)
}

// NewHTTPHandlerWithDB 创建带有数据库实例的 HTTP Handler。
func NewHTTPHandlerWithDB(db *gorm.DB) *HTTPHandler {
	return handler.NewHTTPHandlerWithDB(db)
}

// RegisterAdminAIModelRoutes registers admin model management routes under /admin/ai/models.
func RegisterAdminAIModelRoutes(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHTTPHandler(svcCtx)

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
