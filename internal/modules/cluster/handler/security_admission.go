package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	clusterintegration "github.com/cy77cc/OpsPilot/internal/modules/cluster/integration"
	"github.com/cy77cc/OpsPilot/internal/modules/governance/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type upsertAdmissionPolicyRequest struct {
	PolicyName    string         `json:"policy_name"`
	Version       string         `json:"version"`
	Status        string         `json:"status"`
	Image         string         `json:"image"`
	Namespace     string         `json:"namespace"`
	Workload      string         `json:"workload"`
	ApprovalToken string         `json:"approval_token"`
	Content       map[string]any `json:"content"`
}

type createAdmissionExemptionRequest struct {
	ScopeType     string    `json:"scope_type"`
	ScopeRef      string    `json:"scope_ref"`
	Reason        string    `json:"reason"`
	ExpiresAt     time.Time `json:"expires_at"`
	ApprovalToken string    `json:"approval_token"`
}

func (h *Handler) UpsertAdmissionPolicy(c *gin.Context) {
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

	var req upsertAdmissionPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if strings.TrimSpace(req.PolicyName) == "" || strings.TrimSpace(req.Version) == "" {
		httpx.BindErr(c, fmt.Errorf("policy_name and version are required"))
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
			Namespace:  strings.TrimSpace(req.Namespace),
			Resource:   "admission",
			ResourceID: strings.TrimSpace(req.PolicyName),
			Action:     "admission.apply",
			Context: map[string]any{
				"phase3_domain": "admission",
				"intent":        "policy_upsert",
			},
		},
		RequestSummary: map[string]any{
			"policy_name": strings.TrimSpace(req.PolicyName),
			"version":     strings.TrimSpace(req.Version),
			"image":       strings.TrimSpace(req.Image),
		},
	}

	decision, err := h.phase3Preflight(c.Request.Context(), intent)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !decision.Allowed {
		h.respondPhase3Decision(c, intent, decision, map[string]any{"policy_name": req.PolicyName, "version": req.Version})
		return
	}

	scan := clusterintegration.TrivyScanResult{}
	image := strings.TrimSpace(req.Image)
	if image != "" && h.trivy != nil {
		scan, err = h.trivy.ScanImage(c.Request.Context(), image)
		if err != nil {
			h.respondPhase3Decision(c, intent, governance.Decision{
				Allowed: false,
				State:   governance.StateFailed,
				Code:    OperationCodeFailed,
				Message: err.Error(),
			}, map[string]any{"policy_name": req.PolicyName, "version": req.Version})
			return
		}
		if scan.Summary.Critical > 0 {
			h.respondPhase3Decision(c, intent, governance.Decision{
				Allowed: false,
				State:   governance.StateRejected,
				Code:    "admission_denied",
				Message: "critical vulnerabilities detected",
			}, map[string]any{
				"policy_name": req.PolicyName,
				"version":     req.Version,
				"scan":        scan.Summary,
			})
			return
		}
	}

	contentJSON := "{}"
	if len(req.Content) > 0 {
		buf, merr := json.Marshal(req.Content)
		if merr != nil {
			httpx.ServerErr(c, merr)
			return
		}
		contentJSON = string(buf)
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = "active"
	}
	policy, err := h.repo.Phase3.UpsertAdmissionPolicy(c.Request.Context(), clustermodel.AdmissionPolicy{
		ClusterID:   clusterID,
		PolicyName:  strings.TrimSpace(req.PolicyName),
		Version:     strings.TrimSpace(req.Version),
		Status:      status,
		ContentJSON: contentJSON,
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	h.respondPhase3Decision(c, intent, governance.Decision{Allowed: true, State: governance.StateCompleted, Code: governance.CodeSuccess}, map[string]any{
		"policy": policy,
		"scan":   scan.Summary,
	})
}

func (h *Handler) ListAdmissionResults(c *gin.Context) {
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

	type row struct {
		ID              uint      `json:"id"`
		ImageDigest     string    `json:"image_digest"`
		Scanner         string    `json:"scanner"`
		SeveritySummary string    `json:"severity_summary_json"`
		PolicyDecision  string    `json:"policy_decision"`
		CreatedAt       time.Time `json:"created_at"`
	}
	var rows []row
	err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Table("image_scan_reports").
		Select("id, image_digest, scanner, severity_summary_json, policy_decision, created_at").
		Where("cluster_id = ?", clusterID).
		Order("id DESC").
		Limit(50).
		Find(&rows).Error
	if err != nil {
		// 兼容尚未完成迁移或灰度阶段，返回空结果避免阻断读取链路。
		httpx.OK(c, gin.H{"list": []any{}, "total": 0})
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}

func (h *Handler) CreateAdmissionExemption(c *gin.Context) {
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

	var req createAdmissionExemptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if strings.TrimSpace(req.ScopeType) == "" || strings.TrimSpace(req.ScopeRef) == "" || strings.TrimSpace(req.Reason) == "" {
		httpx.BindErr(c, fmt.Errorf("scope_type, scope_ref and reason are required"))
		return
	}
	if req.ExpiresAt.IsZero() {
		httpx.BindErr(c, fmt.Errorf("expires_at is required"))
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
			Resource:   "admission",
			ResourceID: strings.TrimSpace(req.ScopeRef),
			Action:     "admission.apply",
			Context: map[string]any{
				"phase3_domain": "admission",
				"intent":        "exemption_create",
			},
		},
		RequestSummary: map[string]any{
			"scope_type": strings.TrimSpace(req.ScopeType),
			"scope_ref":  strings.TrimSpace(req.ScopeRef),
			"reason":     strings.TrimSpace(req.Reason),
		},
	}
	decision, err := h.phase3Preflight(c.Request.Context(), intent)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !decision.Allowed {
		h.respondPhase3Decision(c, intent, decision, map[string]any{"scope_type": req.ScopeType, "scope_ref": req.ScopeRef})
		return
	}

	rec, err := h.repo.Phase3.CreateAdmissionExemption(c.Request.Context(), clustermodel.AdmissionExemption{
		ClusterID: clusterID,
		ScopeType: strings.TrimSpace(req.ScopeType),
		ScopeRef:  strings.TrimSpace(req.ScopeRef),
		Reason:    strings.TrimSpace(req.Reason),
		Status:    "active",
		ExpiresAt: req.ExpiresAt.UTC(),
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	h.respondPhase3Decision(c, intent, governance.Decision{Allowed: true, State: governance.StateCompleted, Code: governance.CodeSuccess}, gin.H{"exemption": rec})
}

func (h *Handler) RevokeAdmissionExemption(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	exemptionID := httpx.UintFromParam(c, "exemption_id")
	if clusterID == 0 || exemptionID == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id or exemption id"))
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
	if _, err := h.repo.Phase3.GetAdmissionExemption(c.Request.Context(), clusterID, exemptionID); err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "admission exemption not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	if err := h.repo.Phase3.UpdateAdmissionExemptionStatus(c.Request.Context(), clusterID, exemptionID, "revoked"); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"exemption_id": exemptionID, "status": "revoked"})
}

func (h *Handler) respondPhase3Decision(c *gin.Context, intent governance.OperationIntent, decision governance.Decision, data any) {
	out, err := h.phase3Finalize(c.Request.Context(), governance.FinalizeInput{
		Intent:        intent,
		Decision:      decision,
		ExecutionCode: strings.TrimSpace(decision.Code),
		ExecutionMsg:  strings.TrimSpace(decision.Message),
		Result:        map[string]any{"data": data},
		StartedAt:     time.Now().UTC(),
		FinishedAt:    time.Now().UTC(),
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	envelope := h.phase3BuildEnvelope(decision, out.AuditID, data)
	httpx.OK(c, OperationResponse{
		State:    string(envelope.State),
		Approval: operationApprovalFromGovernanceEnvelope(envelope.Approval),
		AuditID:  envelope.AuditID,
		Code:     envelope.Code,
		Message:  envelope.Message,
		Data:     envelope.Data,
	})
}

func operationApprovalFromGovernanceEnvelope(info *governance.ApprovalInfo) *OperationApproval {
	if info == nil {
		return nil
	}
	return &OperationApproval{
		Required:  true,
		Ticket:    info.Ticket,
		Reason:    info.Reason,
		ExpiresAt: info.ExpiresAt,
	}
}
