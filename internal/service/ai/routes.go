package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/middleware"
	aiHandler "github.com/cy77cc/OpsPilot/internal/service/ai/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func RegisterAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := aiHandler.NewAIHandler(svcCtx)
	h.StartApprovalWorker(context.Background())

	g := v1.Group("/ai", middleware.JWTAuth())
	{
		// 对话相关
		g.POST("/chat", h.Chat)
		g.GET("/sessions", h.ListSessions)
		g.POST("/sessions", h.CreateSession)
		g.GET("/sessions/:id", h.GetSession)
		g.DELETE("/sessions/:id", h.DeleteSession)
		g.GET("/runs/:runId", h.GetRun)
		g.GET("/runs/:runId/projection", h.GetRunProjection)
		g.GET("/run-contents/:id", h.GetRunContent)
		g.GET("/diagnosis/:reportId", h.GetDiagnosisReport)

		// 审批相关 (Human-in-the-Loop)
		g.GET("/approvals/pending", h.ListPendingApprovals)
		g.GET("/approvals/:id", h.GetApproval)
		g.POST("/approvals/:id/submit", h.SubmitApproval)
	}
}
