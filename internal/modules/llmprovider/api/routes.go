// Package api 提供 LLM Provider 的路由注册。
package api

import (
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

// RegisterRoutes 注册 LLM Provider 相关的 HTTP 路由。
func RegisterRoutes(r *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHTTPHandler(svcCtx)

	// 管理员路由
	admin := r.Group("/admin")
	{
		llmProviders := admin.Group("/llm-providers")
		{
			llmProviders.GET("", h.ListModels)
			llmProviders.POST("", h.CreateModel)
			llmProviders.POST("/preview-import", h.PreviewImport)
			llmProviders.POST("/import", h.ImportModels)
			llmProviders.GET("/:id", h.GetModel)
			llmProviders.PUT("/:id", h.UpdateModel)
			llmProviders.DELETE("/:id", h.DeleteModel)
			llmProviders.PUT("/:id/default", h.SetDefaultModel)
		}
	}
}
