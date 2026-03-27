// Package ai 实现 AI 模块的 HTTP 路由注册。
//
// 本文件注册 AI 模块的所有 HTTP 路由:
//   - 用户路由: /api/v1/ai/* (需要 JWT 认证)
//   - 管理路由: /api/v1/admin/ai/* (需要 JWT + Casbin 权限)
package ai

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/service/ai/approval"
	"github.com/cy77cc/OpsPilot/internal/service/ai/chat"
	modelhandler "github.com/cy77cc/OpsPilot/internal/service/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterAIHandlers 注册用户侧 AI 路由。
//
// 所有路由需要 JWT 认证，包括:
//   - POST /ai/chat - SSE 流式对话
//   - GET /ai/sessions - 列出会话
//   - POST /ai/sessions - 创建会话
//   - GET /ai/sessions/:id - 获取会话详情
//   - DELETE /ai/sessions/:id - 删除会话
//   - GET /ai/runs/:runId - 获取运行状态
//   - GET /ai/runs/:runId/projection - 获取运行投影
//   - GET /ai/run-contents/:id - 获取运行内容
//   - GET /ai/diagnosis/:reportId - 获取诊断报告
//   - GET /ai/approvals/pending - 列出待审批任务
//   - GET /ai/approvals/:id - 获取审批详情
//   - POST /ai/approvals/:id/submit - 提交审批结果
//   - POST /ai/approvals/:id/retry-resume - 重新入队可重试恢复
func RegisterAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	chatHandler := chat.NewHTTPHandler(chat.NewService(svcCtx))
	approvalHandler := approval.NewHTTPHandler(approval.NewService(svcCtx))
	approvalHandler.StartApprovalWorker(context.Background())
	approvalHandler.StartApprovalExpirer(context.Background())

	g := v1.Group("/ai", middleware.JWTAuth())
	{
		// 对话相关
		g.POST("/chat", chatHandler.Chat)
		g.GET("/sessions", chatHandler.ListSessions)
		g.POST("/sessions", chatHandler.CreateSession)
		g.GET("/sessions/:id", chatHandler.GetSession)
		g.DELETE("/sessions/:id", chatHandler.DeleteSession)
		g.GET("/runs/:runId", chatHandler.GetRun)
		g.GET("/runs/:runId/projection", chatHandler.GetRunProjection)
		g.GET("/run-contents/:id", chatHandler.GetRunContent)
		g.GET("/diagnosis/:reportId", chatHandler.GetDiagnosisReport)

		// 审批相关 (Human-in-the-Loop)
		g.GET("/approvals/pending", approvalHandler.ListPendingApprovals)
		g.GET("/approvals/:id", approvalHandler.GetApproval)
		g.POST("/approvals/:id/submit", approvalHandler.SubmitApproval)
		g.POST("/approvals/:id/retry-resume", approvalHandler.RetryResumeApproval)
	}
}

// RegisterAdminAIHandlers registers admin model management routes.
func RegisterAdminAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := modelhandler.NewHTTPHandler(svcCtx)

	readOnly := middleware.CasbinAuth(nil, "ai:model:read")
	writeOnly := middleware.CasbinAuth(nil, "ai:model:write")
	if svcCtx != nil {
		readOnly = middleware.CasbinAuth(svcCtx.CasbinEnforcer, "ai:model:read")
		writeOnly = middleware.CasbinAuth(svcCtx.CasbinEnforcer, "ai:model:write")
	}

	g := v1.Group("/admin/ai", middleware.JWTAuth())
	models := g.Group("/models")
	{
		models.GET("", readOnly, h.ListModels)
		models.GET("/:id", readOnly, h.GetModel)
		models.POST("", writeOnly, h.CreateModel)
		models.PUT("/:id", writeOnly, h.UpdateModel)
		models.PUT("/:id/default", writeOnly, h.SetDefaultModel)
		models.DELETE("/:id", writeOnly, h.DeleteModel)
		models.POST("/import/preview", readOnly, h.PreviewImport)
		models.POST("/import", writeOnly, h.ImportModels)
	}
}
