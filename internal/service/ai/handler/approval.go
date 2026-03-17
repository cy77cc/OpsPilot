// Package handler 实现 AI 模块的 HTTP 处理器。
//
// 本文件实现审批相关的 HTTP 接口:
//   - POST /ai/approvals/:id/submit - 提交审批结果（批准/拒绝）
//   - POST /ai/approvals/:id/resume  - 恢复执行（SSE 流式）
//   - GET  /ai/approvals/:id         - 获取审批详情
package handler

import (
	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/gin-gonic/gin"
)

// SubmitApproval 提交审批结果。
//
// 用户在前端审批界面点击"批准"或"拒绝"后调用此接口。
// 该接口仅记录审批结果，不恢复执行。
//
// POST /api/v1/ai/approvals/:id/submit
func (h *Handler) SubmitApproval(c *gin.Context) {
	approvalID := c.Param("id")
	if approvalID == "" {
		httpx.BadRequest(c, "approval_id is required")
		return
	}

	var req aiv1.SubmitApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	userID := httpx.UIDFromCtx(c)
	result, err := h.logic.SubmitApproval(c.Request.Context(), logic.SubmitApprovalInput{
		ApprovalID:       approvalID,
		Approved:         req.Approved,
		DisapproveReason: req.DisapproveReason,
		Comment:          req.Comment,
		UserID:           userID,
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, result)
}

// ResumeApproval 恢复审批执行（SSE 流式）。
//
// 用户批准后，前端调用此接口恢复 AI Agent 执行。
// 该接口通过 SSE 流式返回后续执行结果。
//
// POST /api/v1/ai/approvals/:id/resume
func (h *Handler) ResumeApproval(c *gin.Context) {
	approvalID := c.Param("id")
	if approvalID == "" {
		httpx.BadRequest(c, "approval_id is required")
		return
	}

	var req aiv1.ResumeApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	userID := httpx.UIDFromCtx(c)

	// 设置 SSE 响应头
	c.Status(200)
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	writer := NewSSEWriter(c.Writer)

	if err := h.logic.ResumeApproval(c.Request.Context(), logic.ResumeApprovalInput{
		SessionID:  req.SessionID,
		ApprovalID: approvalID,
		Approved:   req.Approved,
		Reason:     req.Reason,
		Comment:    req.Comment,
		UserID:     userID,
	}, func(event string, data any) {
		writeChatEvent(writer, c, event, data)
	}); err != nil {
		httpx.ServerErr(c, err)
		return
	}
}

// GetApproval 获取审批详情。
//
// GET /api/v1/ai/approvals/:id
func (h *Handler) GetApproval(c *gin.Context) {
	approvalID := c.Param("id")
	if approvalID == "" {
		httpx.BadRequest(c, "approval_id is required")
		return
	}

	userID := httpx.UIDFromCtx(c)
	result, err := h.logic.GetApproval(c.Request.Context(), approvalID, userID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	if result == nil {
		httpx.NotFound(c, "approval not found")
		return
	}

	httpx.OK(c, result)
}

// ListPendingApprovals 列出待处理的审批。
//
// GET /api/v1/ai/approvals/pending
func (h *Handler) ListPendingApprovals(c *gin.Context) {
	userID := httpx.UIDFromCtx(c)
	result, err := h.logic.ListPendingApprovals(c.Request.Context(), userID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, result)
}
