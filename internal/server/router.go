package server

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/bootstrap"
	"github.com/cy77cc/OpsPilot/internal/core/middleware"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// NewRouter constructs the shared HTTP router with middleware and modules.
func NewRouter(ctx context.Context, appCtx *svc.ServiceContext) *gin.Engine {
	return buildRouter(appCtx, func(appCtx *svc.ServiceContext, engine *gin.Engine) {
		bootstrap.RegisterModules(ctx, appCtx, engine)
	})
}

func buildRouter(appCtx *svc.ServiceContext, registerModules func(*svc.ServiceContext, *gin.Engine)) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.ContextMiddleware(), middleware.Cors(), middleware.Logger())

	registerModules(appCtx, r)
	RegisterSwaggerRoutes(r)
	RegisterWebStaticRoutes(r)

	return r
}
