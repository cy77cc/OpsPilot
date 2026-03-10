package ai

import (
	"github.com/cy77cc/k8s-manage/internal/middleware"
	aiHandler "github.com/cy77cc/k8s-manage/internal/service/ai/handler"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

func RegisterAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	g := v1.Group("/ai", middleware.JWTAuth())
	registerHandlers(g, svcCtx)
}

func registerHandlers(g *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := aiHandler.NewHTTPHandler(svcCtx)
	g.POST("/chat", h.Chat)
	g.POST("/chat/respond", h.ChatRespond)
	g.POST("/feedback", h.SubmitFeedback)
	g.POST("/approval/respond", h.HandleApprovalResponse)
	g.POST("/adk/resume", h.ResumeADKApproval)
	g.GET("/sessions", h.ListSessions)
	g.GET("/sessions/current", h.CurrentSession)
	g.GET("/sessions/:id", h.GetSession)
	g.POST("/sessions/:id/branch", h.BranchSession)
	g.PATCH("/sessions/:id", h.UpdateSessionTitle)
	g.DELETE("/sessions/:id", h.DeleteSession)
}
