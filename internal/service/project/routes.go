// Package project 提供项目管理服务的路由注册。
//
// 本文件注册项目相关的 HTTP 路由，包括：
//   - 项目创建和列表
//   - 项目部署
package project

import (
	"github.com/cy77cc/OpsPilot/internal/service/project/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterProjectHandlers 注册项目服务路由。
func RegisterProjectHandlers(g *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	projectHandler := handler.NewProjectHandler(svcCtx)

	// Projects
	projects := g.Group("/projects")
	{
		projects.POST("", projectHandler.CreateProject)
		projects.GET("", projectHandler.ListProjects)
		projects.POST("/deploy", projectHandler.DeployProject)
	}
}
