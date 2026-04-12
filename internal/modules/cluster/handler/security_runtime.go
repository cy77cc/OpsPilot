package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	clustersecurity "github.com/cy77cc/OpsPilot/internal/modules/cluster/domain/security"
	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	"github.com/cy77cc/OpsPilot/internal/modules/governance/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Import types from domain/security
type RuntimeContainResult = clustersecurity.RuntimeContainResult

const ClusterModeExternalManaged = clustersecurity.ClusterModeExternalManaged
const ClusterModePlatformManaged = clustersecurity.ClusterModePlatformManaged

type runtimeIngestRequest struct {
	Source  string          `json:"source"`
	Payload json.RawMessage `json:"payload"`
}

type runtimeContainRequest struct {
	ApprovalToken string `json:"approval_token"`
}

type runtimeDisposalAction struct {
	ID         uint      `gorm:"column:id;primaryKey" json:"id"`
	EventID    uint      `gorm:"column:event_id" json:"event_id"`
	Action     string    `gorm:"column:action" json:"action"`
	Mode       string    `gorm:"column:mode" json:"mode"`
	ApprovalID uint      `gorm:"column:approval_id" json:"approval_id"`
	AuditID    uint      `gorm:"column:audit_id" json:"audit_id"`
	Result     string    `gorm:"column:result" json:"result"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
}

func (runtimeDisposalAction) TableName() string { return "runtime_disposal_actions" }

func (h *Handler) IngestRuntimeEvent(c *gin.Context) {
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

	var req runtimeIngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if len(req.Payload) == 0 {
		httpx.BindErr(c, fmt.Errorf("payload is required"))
		return
	}

	source := strings.ToLower(strings.TrimSpace(req.Source))
	var evt clustersecurity.RuntimeIngestEvent
	var err error
	switch source {
	case clusterclustermodel.SecurityEventSourceFalco:
		evt, err = clustersecurity.ParseFalcoEvent(req.Payload)
	case clusterclustermodel.SecurityEventSourceTetragon:
		evt, err = clustersecurity.ParseTetragonEvent(req.Payload)
	default:
		httpx.BindErr(c, fmt.Errorf("unsupported source: %s", req.Source))
		return
	}
	if err != nil {
		httpx.BindErr(c, err)
		return
	}

	rec := clusterclustermodel.RuntimeSecurityEvent{
		ClusterID:      clusterID,
		Namespace:      strings.TrimSpace(evt.Namespace),
		Workload:       strings.TrimSpace(evt.Workload),
		RuleID:         strings.TrimSpace(evt.RuleID),
		Severity:       strings.TrimSpace(evt.Severity),
		Source:         strings.TrimSpace(evt.Source),
		RawPayloadJSON: string(req.Payload),
		DisposeStatus:  "pending",
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Create(&rec).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"event": rec})
}

func (h *Handler) ListRuntimeAlerts(c *gin.Context) {
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

	severity := strings.ToLower(strings.TrimSpace(c.Query("severity")))
	pageSize := 100
	if raw := strings.TrimSpace(c.DefaultQuery("page_size", "100")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}

	query := h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("cluster_id = ?", clusterID)
	if severity != "" {
		query = query.Where("LOWER(severity) = ?", severity)
	}

	var events []clusterclustermodel.RuntimeSecurityEvent
	if err := query.
		Order("id DESC").
		Limit(pageSize).
		Find(&events).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": events, "total": len(events)})
}

func (h *Handler) GetRuntimeEvent(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	eventID := httpx.UintFromParam(c, "event_id")
	if clusterID == 0 || eventID == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id or event id"))
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

	var event clusterclustermodel.RuntimeSecurityEvent
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Where("cluster_id = ? AND id = ?", clusterID, eventID).First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "runtime event not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"event": event})
}

func (h *Handler) ResolveRuntimeAlert(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	alertID := httpx.UintFromParam(c, "alert_id")
	if clusterID == 0 || alertID == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id or alert id"))
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
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Model(&clusterclustermodel.RuntimeSecurityEvent{}).
		Where("cluster_id = ? AND id = ?", clusterID, alertID).
		Update("dispose_status", "resolved").Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"alert_id": alertID, "status": "resolved"})
}

func (h *Handler) ContainRuntimeAlert(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	alertID := httpx.UintFromParam(c, "alert_id")
	if clusterID == 0 || alertID == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id or alert id"))
		return
	}
	cluster, err := h.repo.GetClusterModel(c.Request.Context(), clusterID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "cluster not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	var event clusterclustermodel.RuntimeSecurityEvent
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Where("cluster_id = ? AND id = ?", clusterID, alertID).First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "runtime alert not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	var req runtimeContainRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.BindErr(c, err)
			return
		}
	}

	now := time.Now().UTC()
	operatorID := uint(httpx.UIDFromCtx(c))
	intent := governance.OperationIntent{
		OperatorID:    operatorID,
		ApprovalToken: strings.TrimSpace(req.ApprovalToken),
		OccurredAt:    now,
		Scope: governance.Scope{
			ClusterID:  clusterID,
			Resource:   "runtime",
			ResourceID: fmt.Sprintf("%d", alertID),
			Action:     "runtime.contain",
			Context: map[string]any{
				"phase3_domain": "runtime",
				"intent":        "contain",
			},
		},
		RequestSummary: map[string]any{"event_id": alertID, "workload": event.Workload},
	}

	mode := clusterclustermodel.DisposalModeAuto
	decision := governance.Decision{Allowed: true, State: governance.StateCompleted, Code: governance.CodeSuccess}
	if strings.TrimSpace(cluster.Source) == ClusterModeExternalManaged {
		mode = clusterclustermodel.DisposalModeSuggestOnly
		decision.Message = "external_managed cluster uses suggest_only containment"
	} else {
		decision, err = h.phase3Preflight(c.Request.Context(), intent)
		if err != nil {
			httpx.ServerErr(c, err)
			return
		}
		if !decision.Allowed {
			h.respondPhase3Decision(c, intent, decision, gin.H{"event_id": alertID, "mode": mode})
			return
		}
	}

	out, err := h.phase3Finalize(c.Request.Context(), governance.FinalizeInput{
		Intent:        intent,
		Decision:      decision,
		ExecutionCode: strings.TrimSpace(decision.Code),
		ExecutionMsg:  strings.TrimSpace(decision.Message),
		Result: map[string]any{
			"event_id": alertID,
			"mode":     mode,
		},
		StartedAt:  now,
		FinishedAt: time.Now().UTC(),
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	disposeStatus := "contained"
	actionResult := "applied"
	if mode == clustermodel.DisposalModeSuggestOnly {
		disposeStatus = "suggested"
		actionResult = "suggested"
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Model(&clustermodel.RuntimeSecurityEvent{}).
		Where("id = ?", alertID).
		Update("dispose_status", disposeStatus).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Create(&runtimeDisposalAction{
		EventID: alertID,
		Action:  "contain",
		Mode:    mode,
		AuditID: out.AuditID,
		Result:  actionResult,
	}).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	result := RuntimeContainResult{
		EventID:   alertID,
		Mode:      mode,
		AuditID:   out.AuditID,
		Status:    disposeStatus,
		UpdatedAt: time.Now().UTC(),
	}
	env := h.phase3BuildEnvelope(decision, out.AuditID, result)
	httpx.OK(c, OperationResponse{
		State:    string(env.State),
		Approval: operationApprovalFromGovernanceEnvelope(env.Approval),
		AuditID:  env.AuditID,
		Code:     env.Code,
		Message:  env.Message,
		Data:     env.Data,
	})
}
