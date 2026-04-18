package alerthealhandler

import (
	"errors"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	svc *Service
}

func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func (h *HTTPHandler) ListJobsByAlert(c *gin.Context) {
	if h == nil || h.svc == nil || h.svc.DB() == nil {
		httpx.ServerErr(c, errors.New("alert-heal service not initialized"))
		return
	}
	if !httpx.Authorize(c, h.svc.DB(), "monitoring:read", "ai:alert:read", "ai:alert:*") {
		return
	}

	alertID := httpx.UintFromQuery(c, "alert_id")
	if alertID == 0 {
		httpx.BadRequest(c, "alert_id is required")
		return
	}

	rows, total, err := h.svc.ListJobsByAlert(c.Request.Context(), alertID)
	switch {
	case errors.Is(err, ErrAlertNotFound):
		httpx.NotFound(c, err.Error())
	case err != nil:
		httpx.ServerErr(c, err)
	default:
		httpx.OK(c, gin.H{"list": rows, "total": total})
	}
}

func (h *HTTPHandler) GetJob(c *gin.Context) {
	if h == nil || h.svc == nil || h.svc.DB() == nil {
		httpx.ServerErr(c, errors.New("alert-heal service not initialized"))
		return
	}
	if !httpx.Authorize(c, h.svc.DB(), "monitoring:read", "ai:alert:read", "ai:alert:*") {
		return
	}

	jobID := strings.TrimSpace(c.Param("id"))
	if jobID == "" {
		httpx.BadRequest(c, "job_id is required")
		return
	}

	row, err := h.svc.GetJob(c.Request.Context(), jobID)
	switch {
	case errors.Is(err, ErrAlertHealNotFound):
		httpx.NotFound(c, err.Error())
	case err != nil:
		httpx.ServerErr(c, err)
	default:
		httpx.OK(c, row)
	}
}

func (h *HTTPHandler) RetryJob(c *gin.Context) {
	if h == nil || h.svc == nil || h.svc.DB() == nil {
		httpx.ServerErr(c, errors.New("alert-heal service not initialized"))
		return
	}
	if !httpx.Authorize(c, h.svc.DB(), "monitoring:write", "ai:alert:write", "ai:alert:*") {
		return
	}

	jobID := strings.TrimSpace(c.Param("id"))
	if jobID == "" {
		httpx.BadRequest(c, "job_id is required")
		return
	}

	row, err := h.svc.RetryJob(c.Request.Context(), jobID)
	switch {
	case errors.Is(err, ErrAlertHealNotFound):
		httpx.NotFound(c, err.Error())
	case errors.Is(err, ErrRetryNotAllowed):
		httpx.Fail(c, xcode.ParamError, err.Error())
	case err != nil:
		httpx.ServerErr(c, err)
	default:
		httpx.OK(c, row)
	}
}
