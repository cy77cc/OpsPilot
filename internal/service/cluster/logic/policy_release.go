package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	clustercontracts "github.com/cy77cc/OpsPilot/internal/service/cluster/contracts"
	clusterpolicy "github.com/cy77cc/OpsPilot/internal/service/cluster/domain/policy"
	"gorm.io/gorm"
)

// Import domain types from policy
type PolicySimulationResult = clusterpolicy.PolicySimulationResult

// Re-export constants from contracts
const (
	PolicyReleaseStateDraft             = clustercontracts.PolicyReleaseStateDraft
	PolicyReleaseStateSimulationPending = clustercontracts.PolicyReleaseStateSimulationPending
	PolicyReleaseStateSimulationPassed  = clustercontracts.PolicyReleaseStateSimulationPassed
	PolicyReleaseStateSimulationFailed  = clustercontracts.PolicyReleaseStateSimulationFailed
	PolicyReleaseStateApprovalRequired  = clustercontracts.PolicyReleaseStateApprovalRequired
	PolicyReleaseStateApplying          = clustercontracts.PolicyReleaseStateApplying
	PolicyReleaseStateApprovalRejected  = clustercontracts.PolicyReleaseStateApprovalRejected
	PolicyReleaseStateApplied           = clustercontracts.PolicyReleaseStateApplied
	PolicyReleaseStateApplyFailed       = clustercontracts.PolicyReleaseStateApplyFailed
	PolicyReleaseStateActive            = clustercontracts.PolicyReleaseStateActive
	PolicyReleaseStateRollbackApplied   = clustercontracts.PolicyReleaseStateRollbackApplied
)

// PolicyReleaseCreateInput 描述创建发布记录所需的最小输入。
type PolicyReleaseCreateInput struct {
	ReleaseID             uint
	Version               string
	PreviousStableVersion string
	Policy                PolicyReference
	TargetCluster         PolicyTargetCluster
	SimulationResult      PolicySimulationResult
	CreatedBy             uint
	CreatedAt             time.Time
}

// PolicyReleaseRecord 表示策略发布服务在 Phase 2 中维护的发布状态。
type PolicyReleaseRecord struct {
	ReleaseID             uint                `json:"release_id,omitempty" yaml:"release_id,omitempty"`
	Version               string              `json:"version,omitempty" yaml:"version,omitempty"`
	PreviousStableVersion string              `json:"previous_stable_version,omitempty" yaml:"previous_stable_version,omitempty"`
	RollbackTargetVersion string              `json:"rollback_target_version,omitempty" yaml:"rollback_target_version,omitempty"`
	Policy                PolicyReference     `json:"policy,omitempty" yaml:"policy,omitempty"`
	TargetCluster         PolicyTargetCluster `json:"target_cluster,omitempty" yaml:"target_cluster,omitempty"`
	Status                PolicyReleaseStatus `json:"status,omitempty" yaml:"status,omitempty"`
	Simulation            PolicySimulationStatus
	Approval              PolicyApprovalStatus `json:"approval,omitempty" yaml:"approval,omitempty"`
	Audit                 PolicyAuditStatus    `json:"audit,omitempty" yaml:"audit,omitempty"`
	LastErrorCode         string               `json:"last_error_code,omitempty" yaml:"last_error_code,omitempty"`
	LastErrorMessage      string               `json:"last_error_message,omitempty" yaml:"last_error_message,omitempty"`
}

// NewPolicyRelease 从仿真结果创建可推进的发布记录。
func NewPolicyRelease(input PolicyReleaseCreateInput) (*PolicyReleaseRecord, error) {
	if input.ReleaseID == 0 {
		return nil, fmt.Errorf("release_id is required")
	}

	version := strings.TrimSpace(input.Version)
	if version == "" {
		return nil, fmt.Errorf("version is required")
	}

	status := input.SimulationResult.PolicyReleaseStatus
	status.ApplyDefaults()
	if input.SimulationResult.Passed {
		status.Phase = PolicyReleaseStateSimulationPassed
	} else {
		status.Phase = PolicyReleaseStateSimulationFailed
	}

	createdAt := normalizePolicyReleaseTime(input.CreatedAt)

	return &PolicyReleaseRecord{
		ReleaseID:             input.ReleaseID,
		Version:               version,
		PreviousStableVersion: strings.TrimSpace(input.PreviousStableVersion),
		Policy:                input.Policy,
		TargetCluster:         input.TargetCluster,
		Status:                status,
		Simulation:            input.SimulationResult.PolicySimulationStatus,
		Audit: PolicyAuditStatus{
			CreatedAt: &createdAt,
			CreatedBy: input.CreatedBy,
		},
	}, nil
}

// EnsureApproval 将发布推进到 approval_required、approval_rejected 或 applying。
func (r *PolicyReleaseRecord) EnsureApproval(ctx context.Context, db *gorm.DB, approvalToken string, operatorID uint, now time.Time) error {
	if r == nil {
		return fmt.Errorf("policy release is nil")
	}
	if db == nil {
		return fmt.Errorf("db is required")
	}
	if r.Status.Phase != PolicyReleaseStateSimulationPassed && r.Status.Phase != PolicyReleaseStateApprovalRequired {
		return fmt.Errorf("cannot evaluate approval from phase %q", r.Status.Phase)
	}

	now = normalizePolicyReleaseTime(now)
	token := strings.TrimSpace(approvalToken)
	r.Approval.Required = true

	if token == "" {
		pendingApproval, err := findPendingPolicyReleaseApproval(db, r, now)
		if err != nil {
			return err
		}
		if pendingApproval != nil {
			r.Status.Phase = PolicyReleaseStateApprovalRequired
			r.Approval.ApprovalToken = pendingApproval.Ticket
			r.Approval.ApprovedAt = nil
			r.clearLastError()
			return nil
		}

		rec, err := IssuePolicyReleaseApproval(
			ctx,
			db,
			r.TargetCluster.ClusterID,
			r.Policy.Namespace,
			r.ReleaseID,
			PolicyReleaseApprovalActionApply,
			operatorID,
			now.Add(30*time.Minute),
		)
		if err != nil {
			return err
		}
		r.Status.Phase = PolicyReleaseStateApprovalRequired
		r.Approval.ApprovalToken = rec.Ticket
		r.Approval.ApprovedAt = nil
		r.clearLastError()
		return nil
	}

	rec, err := ConsumePolicyReleaseApproval(
		ctx,
		db,
		r.TargetCluster.ClusterID,
		r.Policy.Namespace,
		r.ReleaseID,
		PolicyReleaseApprovalActionApply,
		token,
		operatorID,
		now,
	)
	if err != nil {
		r.Approval.ApprovalToken = token
		if approvalErr, ok := IsApprovalError(err); ok {
			r.LastErrorCode = approvalErr.Code
			r.LastErrorMessage = approvalErr.Message
			switch approvalErr.Code {
			case OperationCodeApprovalRejected:
				r.Status.Phase = PolicyReleaseStateApprovalRejected
				return nil
			default:
				r.Status.Phase = PolicyReleaseStateApprovalRequired
			}
		}
		return err
	}

	r.Approval.ApprovalToken = rec.Ticket
	r.Approval.ApprovedAt = &now
	r.clearLastError()
	return r.MarkApplying(now)
}

// MarkApplying 将发布推进到 applying。
func (r *PolicyReleaseRecord) MarkApplying(_ time.Time) error {
	if r == nil {
		return fmt.Errorf("policy release is nil")
	}
	switch r.Status.Phase {
	case PolicyReleaseStateSimulationPassed:
		if r.Approval.Required && !r.hasValidatedApproval() {
			return fmt.Errorf("cannot transition from %q to %q without validated approval", r.Status.Phase, PolicyReleaseStateApplying)
		}
	case PolicyReleaseStateApprovalRequired:
		if !r.hasValidatedApproval() {
			return fmt.Errorf("cannot transition from %q to %q without validated approval", r.Status.Phase, PolicyReleaseStateApplying)
		}
	default:
		return fmt.Errorf("cannot transition from %q to %q", r.Status.Phase, PolicyReleaseStateApplying)
	}

	r.Status.Phase = PolicyReleaseStateApplying
	return nil
}

func (r *PolicyReleaseRecord) hasValidatedApproval() bool {
	return strings.TrimSpace(r.Approval.ApprovalToken) != "" && r.Approval.ApprovedAt != nil
}

func findPendingPolicyReleaseApproval(db *gorm.DB, release *PolicyReleaseRecord, now time.Time) (*model.OperationApproval, error) {
	if db == nil || release == nil {
		return nil, nil
	}

	now = normalizePolicyReleaseTime(now)

	var approval model.OperationApproval
	err := db.
		Where("scope_cluster_id = ? AND namespace = ? AND action = ? AND resource = ? AND resource_id = ? AND status = ? AND consumed_at IS NULL AND (expires_at IS NULL OR expires_at > ?)",
			release.TargetCluster.ClusterID,
			release.Policy.Namespace,
			PolicyReleaseApprovalActionApply,
			PolicyReleaseApprovalResource,
			fmt.Sprintf("%d", release.ReleaseID),
			"pending",
			now,
		).
		Order("updated_at DESC, id DESC").
		First(&approval).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &approval, nil
}

// MarkApplied 将发布推进到 applied 并记录应用时间。
func (r *PolicyReleaseRecord) MarkApplied(now time.Time) error {
	if r == nil {
		return fmt.Errorf("policy release is nil")
	}
	if r.Status.Phase != PolicyReleaseStateApplying {
		return fmt.Errorf("cannot transition from %q to %q", r.Status.Phase, PolicyReleaseStateApplied)
	}
	now = normalizePolicyReleaseTime(now)
	r.Status.Phase = PolicyReleaseStateApplied
	r.Audit.AppliedAt = &now
	r.clearLastError()
	return nil
}

// MarkActive 将发布从 applied 推进到 active。
func (r *PolicyReleaseRecord) MarkActive() error {
	if r == nil {
		return fmt.Errorf("policy release is nil")
	}
	if r.Status.Phase != PolicyReleaseStateApplied {
		return fmt.Errorf("cannot transition from %q to %q", r.Status.Phase, PolicyReleaseStateActive)
	}
	r.Status.Phase = PolicyReleaseStateActive
	return nil
}

// MarkApplyFailed 记录 apply_failed 阶段并保存失败原因。
func (r *PolicyReleaseRecord) MarkApplyFailed(code, message string) error {
	if r == nil {
		return fmt.Errorf("policy release is nil")
	}
	if r.Status.Phase != PolicyReleaseStateApplying {
		return fmt.Errorf("cannot transition from %q to %q", r.Status.Phase, PolicyReleaseStateApplyFailed)
	}
	r.Status.Phase = PolicyReleaseStateApplyFailed
	r.LastErrorCode = strings.TrimSpace(code)
	r.LastErrorMessage = strings.TrimSpace(message)
	return nil
}

// Rollback 在 apply_failed/applied/active 后记录 rollback_applied。
func (r *PolicyReleaseRecord) Rollback(targetVersion string, now time.Time) error {
	if r == nil {
		return fmt.Errorf("policy release is nil")
	}
	switch r.Status.Phase {
	case PolicyReleaseStateApplyFailed, PolicyReleaseStateApplied, PolicyReleaseStateActive:
	default:
		return fmt.Errorf("cannot transition from %q to %q", r.Status.Phase, PolicyReleaseStateRollbackApplied)
	}

	targetVersion = strings.TrimSpace(targetVersion)
	if targetVersion == "" {
		targetVersion = strings.TrimSpace(r.PreviousStableVersion)
	}
	if targetVersion == "" {
		return fmt.Errorf("previous_stable_version is required for rollback")
	}

	now = normalizePolicyReleaseTime(now)
	r.RollbackTargetVersion = targetVersion
	r.Status.Phase = PolicyReleaseStateRollbackApplied
	r.Audit.RollbackAt = &now
	return nil
}

func (r *PolicyReleaseRecord) clearLastError() {
	r.LastErrorCode = ""
	r.LastErrorMessage = ""
}

func normalizePolicyReleaseTime(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now().UTC()
	}
	return ts.UTC()
}
