// Package api 实现 AI 模块的 HTTP 路由注册。
//
// 本文件注册 AI 模块的所有 HTTP 路由:
//   - 用户路由: /api/v1/ai/* (需要 JWT 认证)
//   - 管理路由: /api/v1/admin/ai/* (需要 JWT + Casbin 权限)
package api

import (
	"github.com/cy77cc/OpsPilot/internal/core/middleware"
	aialertheal "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/alertheal"
	aiapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/approval"
	aiapprovalhandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/approval"
	aichat "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/chat"
	aihttp "github.com/cy77cc/OpsPilot/internal/modules/ai/interfaces/http"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterAIHandlers 注册用户侧 AI 路由。
//
// 所有路由需要 JWT 认证，包括:
//   - GET /ai/sessions - 列出会话
//   - POST /ai/sessions - 创建会话
//   - GET /ai/sessions/:id - 获取会话详情
//   - DELETE /ai/sessions/:id - 删除会话
//   - GET /ai/runs/:runId - 获取运行状态
//   - GET /ai/runs/:runId/projection - 获取运行投影
//   - GET /ai/run-contents/:id - 获取运行内容
//   - GET /ai/diagnosis/:reportId - 获取诊断报告
//   - GET /ai/approvals/pending - 列出待审批任务
//   - GET /ai/approvals/pending/global - 列出全局待审批任务（需 ai:approval:read）
//   - GET /ai/approvals/:id - 获取审批详情
//   - POST /ai/approvals/:id/submit - 提交审批结果
//   - POST /ai/approvals/:id/retry-resume - 重新入队可重试恢复
func RegisterAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	chatService := aichat.NewService(svcCtx)
	queryHandler := aihttp.NewChatQueryHandler(chatService)
	approvalHandler := aiapprovalhandler.NewHTTPHandler(aiapproval.NewService(svcCtx))
	alertHealHandler := aialertheal.NewHTTPHandler(aialertheal.NewService(svcCtx))

	g := v1.Group("/ai", middleware.JWTAuth())
	{
		g.GET("/sessions", queryHandler.ListSessions)
		g.POST("/sessions", queryHandler.CreateSession)
		g.GET("/sessions/:id", queryHandler.GetSession)
		g.DELETE("/sessions/:id", queryHandler.DeleteSession)
		g.GET("/runs/:runId", queryHandler.GetRun)
		g.GET("/runs/:runId/projection", queryHandler.GetRunProjection)
		g.GET("/run-contents/:id", queryHandler.GetRunContent)
		g.GET("/diagnosis/:reportId", queryHandler.GetDiagnosisReport)

		// 审批相关 (Human-in-the-Loop)
		g.GET("/approvals/pending", approvalHandler.ListPendingApprovals)
		g.GET("/approvals/pending/global", approvalHandler.ListPendingApprovalsGlobal)
		g.GET("/approvals/:id", approvalHandler.GetApproval)
		g.POST("/approvals/:id/submit", approvalHandler.SubmitApproval)
		g.POST("/approvals/:id/retry-resume", approvalHandler.RetryResumeApproval)
		g.GET("/alert-heal/jobs", alertHealHandler.ListJobsByAlert)
		g.GET("/alert-heal/jobs/:id", alertHealHandler.GetJob)
		g.POST("/alert-heal/jobs/:id/retry", alertHealHandler.RetryJob)
	}
}
