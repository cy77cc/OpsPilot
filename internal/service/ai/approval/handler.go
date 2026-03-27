package approval

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

const workerTick = 2 * time.Second

type HTTPHandler struct {
	svc *Service

	workerMu     sync.Mutex
	workerStart  bool
	workerCancel context.CancelFunc

	expirerMu     sync.Mutex
	expirerStart  bool
	expirerCancel context.CancelFunc
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

	result, err := h.svc.RetryResumeApproval(c.Request.Context(), logic.RetryResumeApprovalInput{
		ApprovalID: approvalID,
		TriggerID:  req.TriggerID,
		UserID:     httpx.UIDFromCtx(c),
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

func (h *HTTPHandler) StartApprovalWorker(ctx context.Context) {
	if h == nil || h.svc == nil {
		return
	}

	h.workerMu.Lock()
	defer h.workerMu.Unlock()
	if h.workerStart {
		return
	}

	workerCtx, cancel := context.WithCancel(ctx)
	h.workerCancel = cancel
	h.workerStart = true
	h.svc.StartWorker(workerCtx)
}

func (h *HTTPHandler) StartApprovalExpirer(ctx context.Context) {
	if h == nil || h.svc == nil {
		return
	}

	h.expirerMu.Lock()
	defer h.expirerMu.Unlock()
	if h.expirerStart {
		return
	}

	expirerCtx, cancel := context.WithCancel(ctx)
	h.expirerCancel = cancel
	h.expirerStart = true
	h.svc.StartExpirer(expirerCtx)
}

func (h *HTTPHandler) GetApproval(c *gin.Context) {
	approvalID := c.Param("id")
	if approvalID == "" {
		httpx.BadRequest(c, "approval_id is required")
		return
	}

	userID := httpx.UIDFromCtx(c)
	result, err := h.svc.GetApproval(c.Request.Context(), approvalID, userID)
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

func (h *HTTPHandler) HealthCheck() error {
	if h == nil || h.svc == nil {
		return fmt.Errorf("approval service not initialized")
	}
	return nil
}
