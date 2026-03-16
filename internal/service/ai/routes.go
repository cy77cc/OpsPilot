package ai

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/service/ai/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func RegisterAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	deps := NewDependencies(svcCtx)
	h := handler.New(handler.Dependencies{
		ChatDAO:            deps.ChatDAO,
		RunDAO:             deps.RunDAO,
		DiagnosisReportDAO: deps.DiagnosisReportDAO,
	})

	g := v1.Group("/ai", middleware.JWTAuth())
	{
		g.POST("/chat", h.Chat)
		g.GET("/sessions", h.ListSessions)
		g.POST("/sessions", h.CreateSession)
		g.GET("/sessions/:id", h.GetSession)
		g.DELETE("/sessions/:id", h.DeleteSession)
		g.GET("/runs/:runId", h.GetRun)
		g.GET("/diagnosis/:reportId", h.GetDiagnosisReport)
	}
}
