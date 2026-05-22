package hostplugin

import (
	"github.com/cy77cc/OpsPilot/internal/core/middleware"
	"github.com/cy77cc/OpsPilot/internal/modules/hostplugin/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func RegisterHostPluginHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := handler.NewHandler(svcCtx)
	g := v1.Group("/host-plugins", middleware.JWTAuth())
	g.GET("/catalog", h.ListCatalog)
	g.GET("/hosts/:id/instances", h.ListHostInstances)
	g.POST("/instances/:instance_id/actions", h.RunInstanceAction)
	g.GET("/tasks/:task_id", h.GetTask)
	g.GET("/tasks/:task_id/logs", h.ListTaskLogs)

	// Package management
	g.GET("/packages", h.ListPackages)
	g.POST("/packages/upload", h.UploadPackage)
	g.DELETE("/packages/:id", h.DeletePackage)

	// Install/uninstall on existing hosts (under /hosts group)
	hostsGroup := v1.Group("/hosts", middleware.JWTAuth())
	hostsGroup.POST("/:id/plugins/install", h.InstallPlugin)
	hostsGroup.POST("/:id/plugins/uninstall", h.UninstallPlugin)
}
