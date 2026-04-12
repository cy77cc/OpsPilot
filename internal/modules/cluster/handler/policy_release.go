package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	clusterlogic "github.com/cy77cc/OpsPilot/internal/modules/cluster/logic"
	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	governance "github.com/cy77cc/OpsPilot/internal/modules/governance"
	governanceaudit "github.com/cy77cc/OpsPilot/internal/modules/governance/audit"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Import types from logic
type PolicyReleaseRecord = clusterlogic.PolicyReleaseRecord
type PolicyReleaseCreateInput = clusterlogic.PolicyReleaseCreateInput
type PolicyReference = clusterlogic.PolicyReference
type PolicyTargetCluster = clusterlogic.PolicyTargetCluster
type PolicySimulationResult = clusterlogic.PolicySimulationResult
type PolicySimulationStatus = clusterlogic.PolicySimulationStatus
type PolicyReleaseStatus = clusterlogic.PolicyReleaseStatus
type PolicyRiskLevel = clusterlogic.PolicyRiskLevel
type PolicyReleaseState = clusterlogic.PolicyReleaseState
type ClusterCNIInfoRecord = clusterlogic.ClusterCNIInfoRecord

// Re-export functions and constants
var (
	NewPolicyRelease                     = clusterlogic.NewPolicyRelease
	NewCompletedOperationResponse        = clusterlogic.NewCompletedOperationResponse
	NewApprovalRequiredOperationResponse = clusterlogic.NewApprovalRequiredOperationResponse
	NewRejectedOperationResponse         = clusterlogic.NewRejectedOperationResponse
	NewFailedOperationResponse           = clusterlogic.NewFailedOperationResponse
	ConsumePolicyReleaseApproval         = clusterlogic.ConsumePolicyReleaseApproval
	IssuePolicyReleaseApproval           = clusterlogic.IssuePolicyReleaseApproval
	ObservePolicyReleaseDuration         = clusterlogic.ObservePolicyReleaseDuration
	TimePtrOrNil                         = clusterlogic.TimePtrOrNil
)

const (
	PolicyDefinitionAPIVersion          = clusterlogic.PolicyDefinitionAPIVersion
	PolicyDefinitionKind                = clusterlogic.PolicyDefinitionKind
	PolicyReleaseStateSimulationPassed  = clusterlogic.PolicyReleaseStateSimulationPassed
	PolicyReleaseStateApprovalRequired  = clusterlogic.PolicyReleaseStateApprovalRequired
	PolicyReleaseStateApprovalRejected  = clusterlogic.PolicyReleaseStateApprovalRejected
	PolicyReleaseStateApplying          = clusterlogic.PolicyReleaseStateApplying
	PolicyRiskLevelLow                  = clusterlogic.PolicyRiskLevelLow
	PolicyRiskLevelMedium               = clusterlogic.PolicyRiskLevelMedium
	PolicyReleaseApprovalActionApply    = clusterlogic.PolicyReleaseApprovalActionApply
	PolicyReleaseApprovalActionRollback = clusterlogic.PolicyReleaseApprovalActionRollback
	PolicyErrorFlannelNetpolDisabled    = clusterlogic.PolicyErrorFlannelNetpolDisabled
)

const (
	policyReleaseCreateAction = "policy.release.create"
)

type policyReleaseCreateRequest struct {
	Version               string `json:"version"`
	PreviousStableVersion string `json:"previous_stable_version"`
}

type policyReleaseActionRequest struct {
	Version        string `json:"version"`
	ApprovalToken  string `json:"approval_token"`
	RollbackTarget string `json:"rollback_target_version"`
}

type clusterCNIInfoResponse struct {
	ClusterID    uint            `json:"cluster_id"`
	CNIType      string          `json:"cni_type,omitempty"`
	CNIVersion   string          `json:"cni_version,omitempty"`
	Capabilities map[string]bool `json:"capabilities"`
	Constraints  map[string]any  `json:"constraints,omitempty"`
}

// GetCNIInfo 返回集群 CNI 能力信息。
func (h *Handler) GetCNIInfo(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	if clusterID == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id"))
		return
	}

	info, err := h.repo.GetClusterCNIInfo(c.Request.Context(), clusterID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "cluster not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, clusterCNIInfoResponse{
		ClusterID:    clusterID,
		CNIType:      info.CNIType,
		CNIVersion:   info.CNIVersion,
		Capabilities: buildCNICapabilityMatrix(info),
		Constraints:  buildCNIConstraints(info),
	})
}

// CreatePolicyRelease 创建策略发布快照并写入治理审计。
func (h *Handler) CreatePolicyRelease(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	if clusterID == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid cluster id"))
		return
	}

	var req policyReleaseCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	namespace := strings.TrimSpace(c.Param("namespace"))
	name := strings.TrimSpace(c.Param("name"))
	version := strings.TrimSpace(req.Version)
	if namespace == "" || name == "" || version == "" {
		httpx.BindErr(c, fmt.Errorf("namespace, name and version are required"))
		return
	}

	cniInfo, err := h.repo.GetClusterCNIInfo(c.Request.Context(), clusterID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "cluster not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	now := time.Now().UTC()
	releaseID := uint(now.UnixNano())
	release, err := NewPolicyRelease(PolicyReleaseCreateInput{
		ReleaseID:             releaseID,
		Version:               version,
		PreviousStableVersion: strings.TrimSpace(req.PreviousStableVersion),
		Policy: PolicyReference{
			APIVersion: PolicyDefinitionAPIVersion,
			Kind:       PolicyDefinitionKind,
			Name:       name,
			Namespace:  namespace,
		},
		TargetCluster: PolicyTargetCluster{
			ClusterID:  clusterID,
			CNIType:    cniInfo.CNIType,
			CNIVersion: cniInfo.CNIVersion,
		},
		SimulationResult: PolicySimulationResult{
			Passed: true,
			PolicySimulationStatus: PolicySimulationStatus{
				PassedAt: &now,
			},
			PolicyReleaseStatus: PolicyReleaseStatus{
				Phase:     PolicyReleaseStateSimulationPassed,
				RiskScore: 0,
				RiskLevel: PolicyRiskLevelLow,
			},
		},
		CreatedBy: uint(httpx.UIDFromCtx(c)),
		CreatedAt: now,
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	auditID, err := h.recordPolicyReleaseAudit(c.Request.Context(), release, policyReleaseCreateAction, OperationStateCompleted, OperationCodeSuccess, "policy release created", uint(httpx.UIDFromCtx(c)), "")
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, NewCompletedOperationResponse(auditID, gin.H{"release": release}))
}

// GetPolicyRelease 返回策略发布详情。
func (h *Handler) GetPolicyRelease(c *gin.Context) {
	clusterID := httpx.UintFromParam(c, "id")
	releaseID := httpx.UintFromParam(c, "release_id")
	if clusterID == 0 || releaseID == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid cluster or release id"))
		return
	}

	release, err := h.repo.GetPolicyReleaseRecord(c.Request.Context(), clusterID, releaseID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "policy release not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{"release": release})
}

// ApplyPolicyRelease 处理策略发布审批与下发。
func (h *Handler) ApplyPolicyRelease(c *gin.Context) {
	h.handlePolicyReleaseAction(c, PolicyReleaseApprovalActionApply)
}

// RollbackPolicyRelease 处理策略回滚审批与下发。
func (h *Handler) RollbackPolicyRelease(c *gin.Context) {
	h.handlePolicyReleaseAction(c, PolicyReleaseApprovalActionRollback)
}

func (h *Handler) handlePolicyReleaseAction(c *gin.Context, action string) {
	startedAt := time.Now()
	observeReleasePhase := func(phase string) {
		ObservePolicyReleaseDuration(phase, time.Since(startedAt))
	}

	clusterID := httpx.UintFromParam(c, "id")
	releaseID := httpx.UintFromParam(c, "release_id")
	if clusterID == 0 || releaseID == 0 {
		httpx.BindErr(c, fmt.Errorf("invalid cluster or release id"))
		return
	}

	release, err := h.repo.GetPolicyReleaseRecord(c.Request.Context(), clusterID, releaseID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httpx.NotFound(c, "policy release not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	var req policyReleaseActionRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			httpx.BindErr(c, err)
			return
		}
	}
	if err := validatePolicyReleaseActionVersion(release.Version, req.Version); err != nil {
		httpx.BindErr(c, err)
		return
	}

	operatorID := uint(httpx.UIDFromCtx(c))
	now := time.Now().UTC()
	allowed, approval, state, code, message, err := h.gatePolicyReleaseAction(c.Request.Context(), release, action, strings.TrimSpace(req.ApprovalToken), operatorID, now)
	if err != nil {
		observeReleasePhase("failed")
		httpx.ServerErr(c, err)
		return
	}

	if !allowed {
		auditID, auditErr := h.recordPolicyReleaseAudit(c.Request.Context(), release, action, state, code, message, operatorID, approvalTicket(approval))
		if auditErr != nil {
			observeReleasePhase("failed")
			httpx.ServerErr(c, auditErr)
			return
		}
		observeReleasePhase(state)
		switch state {
		case OperationStateApprovalRequired:
			httpx.OK(c, NewApprovalRequiredOperationResponse(approval, auditID, message, gin.H{"release": release}))
		case OperationStateRejected:
			httpx.OK(c, NewRejectedOperationResponse(auditID, message, gin.H{"release": release}))
		default:
			httpx.OK(c, NewFailedOperationResponse(auditID, code, message, gin.H{"release": release}))
		}
		return
	}

	switch action {
	case PolicyReleaseApprovalActionApply:
		if err := release.MarkApplied(now); err != nil {
			observeReleasePhase("failed")
			httpx.ServerErr(c, err)
			return
		}
		if err := release.MarkActive(); err != nil {
			observeReleasePhase("failed")
			httpx.ServerErr(c, err)
			return
		}
		message = "policy release applied"
	case PolicyReleaseApprovalActionRollback:
		target := strings.TrimSpace(req.RollbackTarget)
		if target == "" {
			target = release.PreviousStableVersion
		}
		if err := release.Rollback(target, now); err != nil {
			observeReleasePhase("failed")
			httpx.ServerErr(c, err)
			return
		}
		message = "policy release rolled back"
	}

	auditID, err := h.recordPolicyReleaseAudit(c.Request.Context(), release, action, OperationStateCompleted, OperationCodeSuccess, message, operatorID, strings.TrimSpace(req.ApprovalToken))
	if err != nil {
		observeReleasePhase("failed")
		httpx.ServerErr(c, err)
		return
	}
	observeReleasePhase(string(release.Status.Phase))

	httpx.OK(c, NewCompletedOperationResponse(auditID, gin.H{"release": release}))
}

func validatePolicyReleaseActionVersion(storedVersion, requestedVersion string) error {
	stored := strings.TrimSpace(storedVersion)
	requested := strings.TrimSpace(requestedVersion)
	if requested == "" || requested == stored {
		return nil
	}
	return fmt.Errorf("release version mismatch: stored=%s requested=%s", stored, requested)
}

func (h *Handler) gatePolicyReleaseAction(ctx context.Context, release *PolicyReleaseRecord, action, approvalToken string, operatorID uint, now time.Time) (bool, *OperationApproval, string, string, string, error) {
	token := strings.TrimSpace(approvalToken)
	if token == "" {
		rec, err := h.findOrIssuePolicyReleaseApproval(ctx, release, action, operatorID, now)
		if err != nil {
			return false, nil, OperationStateFailed, OperationCodeFailed, err.Error(), err
		}
		release.Status.Phase = PolicyReleaseStateApprovalRequired
		release.Approval.Required = true
		release.Approval.ApprovalToken = rec.Ticket
		return false, operationApprovalFromGovernanceRecord(rec), OperationStateApprovalRequired, OperationCodeApprovalRequired, OperationCodeApprovalRequired, nil
	}

	rec, err := ConsumePolicyReleaseApproval(ctx, h.svcCtx.DB, release.TargetCluster.ClusterID, release.Policy.Namespace, release.ReleaseID, action, token, operatorID, now)
	if err != nil {
		release.Approval.Required = true
		release.Approval.ApprovalToken = token
		if approvalErr, ok := IsApprovalError(err); ok {
			switch approvalErr.Code {
			case OperationCodeApprovalRejected:
				release.Status.Phase = PolicyReleaseStateApprovalRejected
				return false, operationApprovalFromClusterApprovalRecord(rec), OperationStateRejected, approvalErr.Code, approvalErr.Message, nil
			case OperationCodeApprovalRequired:
				release.Status.Phase = PolicyReleaseStateApprovalRequired
				return false, operationApprovalFromClusterApprovalRecord(rec), OperationStateApprovalRequired, approvalErr.Code, approvalErr.Message, nil
			default:
				return false, operationApprovalFromClusterApprovalRecord(rec), OperationStateFailed, approvalErr.Code, approvalErr.Message, nil
			}
		}
		return false, operationApprovalFromClusterApprovalRecord(rec), OperationStateFailed, OperationCodeFailed, err.Error(), nil
	}

	release.Approval.Required = true
	release.Approval.ApprovalToken = rec.Ticket
	release.Approval.ApprovedAt = &now
	if action == PolicyReleaseApprovalActionApply {
		if err := release.MarkApplying(now); err != nil {
			return false, nil, OperationStateFailed, OperationCodeFailed, err.Error(), err
		}
	}
	return true, operationApprovalFromClusterApprovalRecord(rec), OperationStateCompleted, OperationCodeSuccess, "", nil
}

func (h *Handler) findOrIssuePolicyReleaseApproval(ctx context.Context, release *PolicyReleaseRecord, action string, operatorID uint, now time.Time) (*clustermodel.OperationApproval, error) {
	var rec clustermodel.OperationApproval
	err := h.svcCtx.DB.WithContext(ctx).
		Where("scope_cluster_id = ? AND namespace = ? AND action = ? AND resource = ? AND resource_id = ? AND status = ? AND consumed_at IS NULL AND (expires_at IS NULL OR expires_at > ?)",
			release.TargetCluster.ClusterID,
			release.Policy.Namespace,
			action,
			PolicyReleaseApprovalResource,
			strconv.FormatUint(uint64(release.ReleaseID), 10),
			"pending",
			now.UTC(),
		).
		Order("updated_at DESC, id DESC").
		First(&rec).Error
	if err == nil {
		return &rec, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	issued, err := IssuePolicyReleaseApproval(ctx, h.svcCtx.DB, release.TargetCluster.ClusterID, release.Policy.Namespace, release.ReleaseID, action, operatorID, now.Add(30*time.Minute))
	if err != nil {
		return nil, err
	}

	var approval clustermodel.OperationApproval
	if err := h.svcCtx.DB.WithContext(ctx).Where("ticket = ?", issued.Ticket).First(&approval).Error; err != nil {
		return nil, err
	}
	return &approval, nil
}

func (h *Handler) recordPolicyReleaseAudit(ctx context.Context, release *PolicyReleaseRecord, action, state, code, message string, operatorID uint, approvalToken string) (uint, error) {
	svc := governanceaudit.NewService(h.svcCtx.DB, nil)
	return svc.Record(ctx, governance.FinalizeInput{
		Intent: governance.OperationIntent{
			OperatorID:    operatorID,
			ApprovalToken: strings.TrimSpace(approvalToken),
			Scope: governance.Scope{
				Domain:     "cluster",
				ClusterID:  release.TargetCluster.ClusterID,
				Namespace:  release.Policy.Namespace,
				Resource:   PolicyReleaseApprovalResource,
				ResourceID: strconv.FormatUint(uint64(release.ReleaseID), 10),
				Action:     action,
			},
			RequestSummary: policyReleaseAuditSummary(release),
		},
		Decision: governance.Decision{
			State:   governance.OperationState(state),
			Code:    code,
			Message: message,
		},
		ExecutionCode: code,
		ExecutionMsg:  message,
		Result: map[string]any{
			"release": release,
		},
		Diagnostics: map[string]any{
			"policy_name":     release.Policy.Name,
			"release_phase":   release.Status.Phase,
			"rollback_target": release.RollbackTargetVersion,
		},
	})
}

func policyReleaseAuditSummary(release *PolicyReleaseRecord) map[string]any {
	return map[string]any{
		"release":     release,
		"policy_name": release.Policy.Name,
		"namespace":   release.Policy.Namespace,
		"version":     release.Version,
	}
}

func buildCNICapabilityMatrix(info ClusterCNIInfoRecord) map[string]bool {
	cniType := strings.ToLower(strings.TrimSpace(info.CNIType))
	capabilities := map[string]bool{
		"standard_np":     true,
		"l7":              false,
		"fqdn":            false,
		"order":           false,
		"service_account": false,
		"publishable":     true,
	}
	switch cniType {
	case "cilium":
		capabilities["l7"] = true
		capabilities["fqdn"] = true
	case "calico":
		capabilities["order"] = true
		capabilities["service_account"] = true
	case "flannel":
		capabilities["publishable"] = info.NetpolEnabled
	}
	if cniType == "" {
		capabilities["standard_np"] = false
		capabilities["publishable"] = false
	}
	return capabilities
}

func buildCNIConstraints(info ClusterCNIInfoRecord) map[string]any {
	constraints := map[string]any{}
	if strings.EqualFold(info.CNIType, "flannel") {
		constraints["netpol_enabled"] = info.NetpolEnabled
		if !info.NetpolEnabled {
			constraints["publish_block_reason"] = PolicyErrorFlannelNetpolDisabled
		}
	}
	return constraints
}

func approvalTicket(approval *OperationApproval) string {
	if approval == nil {
		return ""
	}
	return strings.TrimSpace(approval.Ticket)
}

func operationApprovalFromClusterApprovalRecord(rec *clustermodel.ClusterDeployApproval) *OperationApproval {
	if rec == nil {
		return nil
	}
	return &OperationApproval{
		Required:      true,
		Ticket:        rec.Ticket,
		ClusterID:     rec.ClusterID,
		Namespace:     rec.Namespace,
		Action:        rec.Action,
		Resource:      rec.Resource,
		ResourceID:    rec.ResourceID,
		ExpiresAt:     TimePtrOrNil(rec.ExpiresAt),
		ConsumedAt:    rec.ConsumedAt,
		ConsumedBy:    rec.ConsumedBy,
		ReplayCount:   rec.ReplayCount,
		ReplayAt:      rec.ReplayAt,
		ReplayBy:      rec.ReplayBy,
		ReplayCode:    rec.ReplayCode,
		ReplayMessage: rec.ReplayMessage,
		Status:        rec.Status,
	}
}
