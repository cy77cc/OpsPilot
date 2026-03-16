// Package jobs 提供定时任务管理服务的路由注册。
//
// 本文件注册任务相关的 HTTP 路由，包括：
//   - 任务 CRUD
//   - 任务启停控制
//   - 执行记录查询
//   - 日志查看
package jobs

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	jobshandler "github.com/cy77cc/OpsPilot/internal/service/jobs/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterJobsHandlers 注册任务服务路由到 v1 组。
func RegisterJobsHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := jobshandler.NewHandler(svcCtx)
	g := v1.Group("/jobs", middleware.JWTAuth())
	{
		g.GET("", h.ListJobs)
		g.POST("", h.CreateJob)
		g.GET("/:id", h.GetJob)
		g.PUT("/:id", h.UpdateJob)
		g.DELETE("/:id", h.DeleteJob)
		g.POST("/:id/start", h.StartJob)
		g.POST("/:id/stop", h.StopJob)
		g.GET("/:id/executions", h.GetJobExecutions)
		g.GET("/:id/logs", h.GetJobLogs)
	}
}
