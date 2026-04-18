// Package approvalhandler 实现 AI 审批的 HTTP Handler。
package approvalhandler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	svc *Service
}

func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func (h *HTTPHandler) SubmitApproval(c *gin.Context) {
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
	task, err := h.svc.GetApprovalGlobal(c.Request.Context(), approvalID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if task == nil {
		httpx.NotFound(c, "approval not found")
		return
	}
	if task.UserID == 0 && !hasGlobalApprovalPermission(h, userID, "ai:approval:write", "ai:approval:*") {
		httpx.Fail(c, xcode.Forbidden, "")
		return
	}

	reqCtx := c.Request.Context()
	if idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key")); idempotencyKey != "" {
		reqCtx = logic.WithApprovalSubmitIdempotencyKey(reqCtx, idempotencyKey)
	}

	result, err := h.svc.SubmitApproval(reqCtx, logic.SubmitApprovalInput{
		ApprovalID:       approvalID,
		Approved:         req.Approved,
		DisapproveReason: req.DisapproveReason,
		Comment:          req.Comment,
		UserID:           userID,
	})
	if err != nil {
		var notFoundErr *logic.ApprovalNotFoundError
		var forbiddenErr *logic.ApprovalForbiddenError
		var conflictErr *logic.ApprovalConflictError
		switch {
		case errors.As(err, &notFoundErr):
			httpx.NotFound(c, notFoundErr.Error())
		case errors.As(err, &forbiddenErr):
			httpx.Fail(c, xcode.Forbidden, forbiddenErr.Error())
		case errors.As(err, &conflictErr):
			httpx.Fail(c, xcode.ParamError, conflictErr.Error())
		default:
			httpx.ServerErr(c, err)
		}
		return
	}

	httpx.OK(c, result)
}

func (h *HTTPHandler) RetryResumeApproval(c *gin.Context) {
	approvalID := c.Param("id")
	if approvalID == "" {
		httpx.BadRequest(c, "approval_id is required")
		return
	}

	var req aiv1.RetryResumeApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if strings.TrimSpace(req.TriggerID) == "" {
		httpx.BadRequest(c, "trigger_id is required")
		return
	}

	userID := httpx.UIDFromCtx(c)
	task, err := h.svc.GetApprovalGlobal(c.Request.Context(), approvalID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if task == nil {
		httpx.NotFound(c, "approval not found")
		return
	}
	if task.UserID == 0 && !hasGlobalApprovalPermission(h, userID, "ai:approval:write", "ai:approval:*") {
		httpx.Fail(c, xcode.Forbidden, "")
		return
	}

	result, err := h.svc.RetryResumeApproval(c.Request.Context(), logic.RetryResumeApprovalInput{
		ApprovalID: approvalID,
		TriggerID:  req.TriggerID,
		UserID:     userID,
	})
	if err != nil {
		var notFoundErr *logic.ApprovalNotFoundError
		var forbiddenErr *logic.ApprovalForbiddenError
		var conflictErr *logic.ApprovalConflictError
		switch {
		case errors.As(err, &notFoundErr):
			httpx.NotFound(c, notFoundErr.Error())
		case errors.As(err, &forbiddenErr):
			httpx.Fail(c, xcode.Forbidden, forbiddenErr.Error())
		case errors.As(err, &conflictErr):
			httpx.Fail(c, xcode.ParamError, conflictErr.Error())
		default:
			httpx.ServerErr(c, err)
		}
		return
	}

	httpx.OK(c, result)
}

func (h *HTTPHandler) GetApproval(c *gin.Context) {
	approvalID := c.Param("id")
	if approvalID == "" {
		httpx.BadRequest(c, "approval_id is required")
		return
	}

	userID := httpx.UIDFromCtx(c)
	var (
		result *ai.AIApprovalTask
		err    error
	)
	if hasGlobalApprovalPermission(h, userID, "ai:approval:read", "ai:approval:*") {
		result, err = h.svc.GetApprovalGlobal(c.Request.Context(), approvalID)
	} else {
		result, err = h.svc.GetApproval(c.Request.Context(), approvalID, userID)
	}
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

func (h *HTTPHandler) ListPendingApprovals(c *gin.Context) {
	userID := httpx.UIDFromCtx(c)
	result, err := h.svc.ListPendingApprovals(c.Request.Context(), userID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, result)
}

func (h *HTTPHandler) ListPendingApprovalsGlobal(c *gin.Context) {
	if h == nil || h.svc == nil || h.svc.DB() == nil {
		httpx.ServerErr(c, errors.New("approval service not initialized"))
		return
	}
	if !httpx.Authorize(c, h.svc.DB(), "ai:approval:read", "ai:approval:*") {
		return
	}

	page, err := parsePositiveQueryInt(c.Query("page"), 1)
	if err != nil {
		httpx.BadRequest(c, "page must be a positive integer")
		return
	}
	pageSize, err := parsePositiveQueryInt(c.Query("page_size"), 50)
	if err != nil {
		httpx.BadRequest(c, "page_size must be a positive integer")
		return
	}

	result, total, err := h.svc.ListPendingApprovalsGlobal(c.Request.Context(), page, pageSize)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": result, "total": total})
}

func (h *HTTPHandler) HealthCheck() error {
	if h == nil || h.svc == nil {
		return fmt.Errorf("approval service not initialized")
	}
	return nil
}

func parsePositiveQueryInt(raw string, fallback int) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid positive integer: %q", raw)
	}
	return parsed, nil
}

func hasGlobalApprovalPermission(h *HTTPHandler, userID uint64, codes ...string) bool {
	if h == nil || h.svc == nil || h.svc.DB() == nil || userID == 0 {
		return false
	}
	return httpx.HasAnyPermission(h.svc.DB(), userID, codes...)
}
