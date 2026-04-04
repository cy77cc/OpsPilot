package cluster

import (
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	governanceapproval "github.com/cy77cc/OpsPilot/internal/service/governance/approval"
	"github.com/gin-gonic/gin"
)

// CreateApproval creates a governance-backed approval ticket for active cluster routes.
func (h *Handler) CreateApproval(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:deploy", "k8s:rollback", "kubernetes:write", "cluster:write") {
		return
	}

	clusterID := httpx.UintFromParam(c, "id")
	if clusterID == 0 {
		httpx.BindErr(c, nil)
		return
	}
	if _, err := h.repo.GetClusterModel(c.Request.Context(), clusterID); err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	var req struct {
		Namespace  string `json:"namespace"`
		Action     string `json:"action" binding:"required"`
		Resource   string `json:"resource"`
		ResourceID string `json:"resource_id"`
		ExpiresAt  string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	scope := NormalizeApprovalScope(ApprovalScope{
		ClusterID:  clusterID,
		Namespace:  strings.TrimSpace(req.Namespace),
		Action:     strings.TrimSpace(req.Action),
		Resource:   strings.TrimSpace(req.Resource),
		ResourceID: strings.TrimSpace(req.ResourceID),
	})
	expiresAt := time.Time{}
	if raw := strings.TrimSpace(req.ExpiresAt); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			expiresAt = parsed.UTC()
		}
	}

	rec, err := IssueClusterDeployApproval(c.Request.Context(), h.svcCtx.DB, scope, uint(httpx.UIDFromCtx(c)), expiresAt)
	if err != nil {
		if approvalErr, ok := IsApprovalError(err); ok {
			httpx.BadRequest(c, approvalErr.Code)
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, OperationApprovalFromRecord(rec))
}

// ConfirmApproval confirms or rejects a governance-backed cluster approval ticket.
func (h *Handler) ConfirmApproval(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "k8s:approve", "kubernetes:approve") {
		return
	}

	clusterID := httpx.UintFromParam(c, "id")
	if clusterID == 0 {
		httpx.BindErr(c, nil)
		return
	}
	if _, err := h.repo.GetClusterModel(c.Request.Context(), clusterID); err != nil {
		httpx.NotFound(c, "cluster not found")
		return
	}

	ticket := strings.TrimSpace(c.Param("ticket"))
	if ticket == "" {
		httpx.BindErr(c, nil)
		return
	}

	var req struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	_ = c.ShouldBindJSON(&req)
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = "approved"
	}
	if status != "approved" && status != "rejected" {
		httpx.BadRequest(c, "status must be approved or rejected")
		return
	}

	var approvalRow model.OperationApproval
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("ticket = ? AND domain = ? AND scope_cluster_id = ?", ticket, "cluster", clusterID).
		First(&approvalRow).Error; err != nil {
		httpx.NotFound(c, "approval ticket not found")
		return
	}

	svc := governanceapproval.NewService(h.svcCtx.DB)
	if err := svc.Confirm(c.Request.Context(), ticket, uint(httpx.UIDFromCtx(c)), status == "approved", req.Note); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Where("ticket = ?", ticket).First(&approvalRow).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, operationApprovalFromGovernanceRecord(&approvalRow))
}
