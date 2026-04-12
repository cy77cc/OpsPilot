package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type registerGitOpsAppRequest struct {
	AppName     string `json:"app_name"`
	Environment string `json:"environment"`
	GitRevision string `json:"git_revision"`
}

type syncGitOpsAppRequest struct {
	Environment     string `json:"environment"`
	DesiredRevision string `json:"desired_revision"`
	ApprovalToken   string `json:"approval_token"`
}

type rollbackGitOpsAppRequest struct {
	Environment   string `json:"environment"`
	RollbackRef   string `json:"rollback_ref"`
	ApprovalToken string `json:"approval_token"`
}

func (h *Handler) RegisterGitOpsApp(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	if clusterID == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id"))
		return
	}
	if _, err := h.repo.GetClusterModel(c.Request.Context(), clusterID); err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "cluster not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	var req registerGitOpsAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if strings.TrimSpace(req.AppName) == "" {
		httpx.BindErr(c, fmt.Errorf("app_name is required"))
		return
	}
	if strings.TrimSpace(req.Environment) == "" {
		req.Environment = "prod"
	}

	release := gitopsReleaseRow{
		ClusterID:   clusterID,
		AppName:     strings.TrimSpace(req.AppName),
		Environment: strings.TrimSpace(req.Environment),
		GitRevision: strings.TrimSpace(req.GitRevision),
		SyncResult:  "registered",
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Create(&release).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"app": release})
}

func (h *Handler) GetGitOpsApp(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	name := strings.TrimSpace(c.Param("name"))
	if clusterID == 0 || name == "" {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id or app name"))
		return
	}
	if _, err := h.repo.GetClusterModel(c.Request.Context(), clusterID); err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "cluster not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	var release gitopsReleaseRow
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("cluster_id = ? AND app_name = ?", clusterID, name).
		Order("id DESC").
		First(&release).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "gitops app not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"app": release})
}

func (h *Handler) SyncGitOpsApp(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	name := strings.TrimSpace(c.Param("name"))
	if clusterID == 0 || name == "" {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id or app name"))
		return
	}
	if _, err := h.repo.GetClusterModel(c.Request.Context(), clusterID); err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "cluster not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	var req syncGitOpsAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if strings.TrimSpace(req.Environment) == "" {
		req.Environment = "prod"
	}

	trip, _, err := ShouldTripGitOpsCircuitBreaker(c.Request.Context(), h.svcCtx.DB, clusterID, name, 3)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	now := time.Now().UTC()
	operatorID := uint(httpx.UIDFromCtx(c))
	intent := governance.OperationIntent{
		OperatorID:    operatorID,
		ApprovalToken: strings.TrimSpace(req.ApprovalToken),
		OccurredAt:    now,
		Scope: governance.Scope{
			ClusterID:  clusterID,
			Resource:   "gitops",
			ResourceID: name,
			Action:     "gitops.sync",
			Context: map[string]any{
				"phase3_domain": "gitops",
				"intent":        "sync",
			},
		},
		RequestSummary: map[string]any{"app_name": name, "environment": req.Environment, "desired_revision": req.DesiredRevision},
	}
	if trip {
		h.respondPhase3Decision(c, intent, governance.Decision{
			Allowed: false,
			State:   governance.StateFailed,
			Code:    "circuit_open",
			Message: "gitops sync is blocked by circuit breaker",
		}, gin.H{"app_name": name})
		return
	}

	decision, err := h.phase3Preflight(c.Request.Context(), intent)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !decision.Allowed {
		h.respondPhase3Decision(c, intent, decision, gin.H{"app_name": name})
		return
	}
	if h.argocd == nil {
		h.respondPhase3Decision(c, intent, governance.Decision{Allowed: false, State: governance.StateFailed, Code: OperationCodeFailed, Message: "argocd client not configured"}, gin.H{"app_name": name})
		return
	}

	syncResult, syncErr := h.argocd.Sync(c.Request.Context(), name)
	if syncErr != nil {
		_ = h.svcCtx.DB.WithContext(c.Request.Context()).Create(&gitopsReleaseRow{
			ClusterID:   clusterID,
			AppName:     name,
			Environment: strings.TrimSpace(req.Environment),
			GitRevision: strings.TrimSpace(req.DesiredRevision),
			SyncResult:  "failed",
		}).Error
		h.respondPhase3Decision(c, intent, governance.Decision{Allowed: false, State: governance.StateFailed, Code: OperationCodeFailed, Message: syncErr.Error()}, gin.H{"app_name": name})
		return
	}

	release := gitopsReleaseRow{
		ClusterID:   clusterID,
		AppName:     name,
		Environment: strings.TrimSpace(req.Environment),
		GitRevision: strings.TrimSpace(syncResult.Revision),
		SyncResult:  strings.TrimSpace(syncResult.Status),
	}
	if release.SyncResult == "" {
		release.SyncResult = "succeeded"
	}
	if release.GitRevision == "" {
		release.GitRevision = strings.TrimSpace(req.DesiredRevision)
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Create(&release).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	drift, derr := EvaluateDrift(c.Request.Context(), h.svcCtx.DB, clusterID, name, req.DesiredRevision)
	if derr != nil {
		httpx.ServerErr(c, derr)
		return
	}
	h.respondPhase3Decision(c, intent, governance.Decision{Allowed: true, State: governance.StateCompleted, Code: governance.CodeSuccess}, gin.H{"release": release, "drift": drift})
}

func (h *Handler) RollbackGitOpsApp(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	name := strings.TrimSpace(c.Param("name"))
	if clusterID == 0 || name == "" {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id or app name"))
		return
	}
	if _, err := h.repo.GetClusterModel(c.Request.Context(), clusterID); err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "cluster not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	var req rollbackGitOpsAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if strings.TrimSpace(req.Environment) == "" {
		req.Environment = "prod"
	}
	if strings.TrimSpace(req.RollbackRef) == "" {
		httpx.BindErr(c, fmt.Errorf("rollback_ref is required"))
		return
	}

	now := time.Now().UTC()
	operatorID := uint(httpx.UIDFromCtx(c))
	intent := governance.OperationIntent{
		OperatorID:    operatorID,
		ApprovalToken: strings.TrimSpace(req.ApprovalToken),
		OccurredAt:    now,
		Scope: governance.Scope{
			ClusterID:  clusterID,
			Resource:   "gitops",
			ResourceID: name,
			Action:     "gitops.sync",
			Context: map[string]any{
				"phase3_domain": "gitops",
				"intent":        "rollback",
			},
		},
		RequestSummary: map[string]any{"app_name": name, "rollback_ref": req.RollbackRef},
	}
	decision, err := h.phase3Preflight(c.Request.Context(), intent)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !decision.Allowed {
		h.respondPhase3Decision(c, intent, decision, gin.H{"app_name": name})
		return
	}

	rec := gitopsReleaseRow{
		ClusterID:   clusterID,
		AppName:     name,
		Environment: strings.TrimSpace(req.Environment),
		GitRevision: strings.TrimSpace(req.RollbackRef),
		RollbackRef: strings.TrimSpace(req.RollbackRef),
		SyncResult:  "rolled_back",
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Create(&rec).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	h.respondPhase3Decision(c, intent, governance.Decision{Allowed: true, State: governance.StateCompleted, Code: governance.CodeSuccess}, gin.H{"release": rec})
}
