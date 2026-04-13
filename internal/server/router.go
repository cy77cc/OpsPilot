package server

import (
	"github.com/cy77cc/OpsPilot/internal/bootstrap"
	"github.com/cy77cc/OpsPilot/internal/core/middleware"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// NewRouter constructs the shared HTTP router with middleware and modules.
func NewRouter(appCtx *svc.ServiceContext) *gin.Engine {
	return buildRouter(appCtx, bootstrap.RegisterModules)
}

func buildRouter(appCtx *svc.ServiceContext, registerModules func(*svc.ServiceContext, *gin.Engine)) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.ContextMiddleware(), middleware.Cors(), middleware.Logger())

	registerModules(appCtx, r)
	RegisterSwaggerRoutes(r)
	RegisterWebStaticRoutes(r)

	return r
}
