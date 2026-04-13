package server

import (
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/cy77cc/OpsPilot/docs/swagger"
)

// RegisterSwaggerRoutes registers the Swagger UI when enabled.
func RegisterSwaggerRoutes(r *gin.Engine) {
	if !config.SwaggerEnabled() {
		logger.L().Info("Swagger UI disabled")
		return
	}

	swaggerPath := config.SwaggerPath()
	r.GET(swaggerPath, ginSwagger.WrapHandler(swaggerFiles.Handler))
	logger.L().Info("Swagger UI enabled", logger.String("path", swaggerPath))
}
