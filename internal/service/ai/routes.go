package ai

import (
	"github.com/cy77cc/k8s-manage/internal/middleware"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

func RegisterAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := newHandler(svcCtx)
	g := v1.Group("/ai", middleware.JWTAuth())
	{
		g.POST("/chat", h.chat)
		g.GET("/capabilities", h.capabilities)
		g.GET("/tools/:name/params/hints", h.toolParamHints)
		g.GET("/scene/:scene/tools", h.sceneTools)
		g.POST("/tools/preview", h.previewTool)
		g.POST("/tools/execute", h.executeTool)
		g.GET("/executions/:id", h.getExecution)
		g.POST("/approvals", h.createApproval)
		g.POST("/approvals/:id/confirm", h.confirmApproval)
		g.POST("/approval/respond", h.handleApprovalResponse)
		g.POST("/adk/resume", h.resumeADKApproval)
		g.POST("/confirmations/:id/confirm", h.confirmConfirmation)
		g.GET("/sessions", h.listSessions)
		g.GET("/sessions/current", h.currentSession)
		g.GET("/sessions/:id", h.getSession)
		g.POST("/sessions/:id/branch", h.branchSession)
		g.PATCH("/sessions/:id", h.updateSessionTitle)
		g.DELETE("/sessions/:id", h.deleteSession)
		g.POST("/analyze", h.analyze)
		g.POST("/recommendations", h.recommendations)
		g.POST("/k8s/analyze", h.k8sAnalyze)
		g.POST("/k8s/actions/preview", h.actionPreview)
		g.POST("/k8s/actions/execute", h.actionExecute)
		g.POST("/actions/preview", h.actionPreview)
		g.POST("/actions/execute", h.actionExecute)
		g.GET("/commands/suggestions", h.commandSuggestions)
		g.GET("/commands/aliases", h.listCommandAliases)
		g.POST("/commands/aliases", h.saveCommandAlias)
		g.DELETE("/commands/aliases/:alias", h.deleteCommandAlias)
		g.GET("/commands/templates", h.listCommandTemplates)
		g.POST("/commands/templates", h.saveCommandTemplate)
		g.POST("/commands/preview", h.previewCommand)
		g.POST("/commands/execute", h.executeCommand)
		g.GET("/commands/history", h.listCommandHistory)
		g.GET("/commands/history/:id", h.getCommandHistory)
	}
}
