// Package ai 提供 AI 编排服务的 HTTP 接口。
//
// 本文件注册 AI 相关的路由，包括：
//   - 对话接口（聊天、响应、反馈）
//   - 流程恢复接口（步骤恢复、审批响应）
//   - 会话管理接口（列表、查询、分支、删除）
package ai

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterAIHandlers 注册 AI 服务路由到 v1 组。
func RegisterAIHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	g := v1.Group("/ai", middleware.JWTAuth())
	registerHandlers(g, svcCtx)
}

// registerHandlers 注册具体的路由处理器。
func registerHandlers(g *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := NewHTTPHandler(svcCtx)
	g.Use(h.SceneContextMiddleware())
	// 对话接口
	g.POST("/chat", h.Chat)
	g.POST("/chat/respond", h.Chat)
	g.POST("/feedback", h.SubmitFeedback)
	g.GET("/capabilities", h.Capabilities)
	g.GET("/tools/:name/params/hints", h.ToolParamHints)
	g.POST("/tools/preview", h.PreviewTool)
	g.POST("/tools/execute", h.ExecuteTool)
	g.GET("/executions/:id", h.ExecutionStatus)
	g.POST("/chains/:chain_id/approvals/:node_id/decision", h.DecideChainApproval)
	g.POST("/approvals", h.CreateApproval)
	g.GET("/approvals", h.ListApprovals)
	g.GET("/approvals/:id", h.GetApproval)
	g.POST("/approvals/:id/approve", h.ApproveApproval)
	g.POST("/approvals/:id/reject", h.RejectApproval)
	g.GET("/scene/:scene/tools", h.SceneTools)
	g.GET("/scene/:scene/prompts", h.ScenePrompts)
	g.GET("/scene/configs", h.ListSceneConfigs)
	g.GET("/scene/configs/:scene", h.GetSceneConfig)
	g.PUT("/scene/configs/:scene", h.UpdateSceneConfig)
	g.DELETE("/scene/configs/:scene", h.DeleteSceneConfig)
	// 会话管理接口
	g.GET("/sessions", h.ListSessions)
	g.GET("/sessions/current", h.CurrentSession)
	g.GET("/sessions/:id", h.GetSession)
	g.POST("/sessions/:id/branch", h.BranchSession)
	g.PATCH("/sessions/:id", h.UpdateSessionTitle)
	g.DELETE("/sessions/:id", h.DeleteSession)
}
