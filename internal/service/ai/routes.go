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
	g.GET("/tools", h.ListTools)
	g.GET("/capabilities", h.ListTools)
	g.GET("/tools/:name/params/hints", h.ToolParamHints)
	g.GET("/scene/:scene/tools", h.SceneTools)
	g.GET("/scene/:scene/prompts", h.ScenePrompts)
	g.POST("/tools/preview", h.PreviewTool)
	g.POST("/tools/execute", h.ExecuteTool)
	g.GET("/executions/:id", h.GetExecution)
	g.POST("/approvals", h.CreateApproval)
	g.POST("/approvals/:id/confirm", h.ConfirmApproval)
	g.POST("/approval/respond", h.HandleApprovalResponse)
	g.POST("/adk/resume", h.ResumeADKApproval)
	g.POST("/confirmations/:id/confirm", h.ConfirmConfirmation)
	g.GET("/sessions", h.ListSessions)
	g.GET("/sessions/current", h.CurrentSession)
	g.GET("/sessions/:id", h.GetSession)
	g.POST("/sessions/:id/branch", h.BranchSession)
	g.PATCH("/sessions/:id", h.UpdateSessionTitle)
	g.DELETE("/sessions/:id", h.DeleteSession)
}
