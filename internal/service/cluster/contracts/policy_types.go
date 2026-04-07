package contracts

import "time"

// PolicyReleaseState 表示策略发布状态机中的阶段。
type PolicyReleaseState string

const (
	PolicyReleaseStateDraft             PolicyReleaseState = "draft"
	PolicyReleaseStateSimulationPending PolicyReleaseState = "simulation_pending"
	PolicyReleaseStateSimulationPassed  PolicyReleaseState = "simulation_passed"
	PolicyReleaseStateSimulationFailed  PolicyReleaseState = "simulation_failed"
	PolicyReleaseStateApprovalRequired  PolicyReleaseState = "approval_required"
	PolicyReleaseStateApplying          PolicyReleaseState = "applying"
	PolicyReleaseStateApprovalRejected  PolicyReleaseState = "approval_rejected"
	PolicyReleaseStateApplied           PolicyReleaseState = "applied"
	PolicyReleaseStateApplyFailed       PolicyReleaseState = "apply_failed"
	PolicyReleaseStateActive            PolicyReleaseState = "active"
	PolicyReleaseStateRollbackApplied   PolicyReleaseState = "rollback_applied"
)

// PolicyRiskLevel 表示仿真或发布链路中的风险等级。
type PolicyRiskLevel string

const (
	PolicyRiskLevelLow      PolicyRiskLevel = "LOW"
	PolicyRiskLevelMedium   PolicyRiskLevel = "MEDIUM"
	PolicyRiskLevelHigh     PolicyRiskLevel = "HIGH"
	PolicyRiskLevelCritical PolicyRiskLevel = "CRITICAL"
)

// PolicyIssueSeverity 表示仿真问题的严重级别。
type PolicyIssueSeverity string

const (
	PolicyIssueSeverityBlocking PolicyIssueSeverity = "BLOCKING"
	PolicyIssueSeverityHigh     PolicyIssueSeverity = "HIGH"
	PolicyIssueSeverityMedium   PolicyIssueSeverity = "MEDIUM"
	PolicyIssueSeverityLow      PolicyIssueSeverity = "LOW"
)

// PolicyReference 表示发布记录中的策略引用。
type PolicyReference struct {
	APIVersion string `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"` // API 版本
	Kind       string `json:"kind,omitempty" yaml:"kind,omitempty"`             // 资源类型
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`             // 策略名称
	Namespace  string `json:"namespace,omitempty" yaml:"namespace,omitempty"`   // 策略命名空间
}

// PolicyTargetCluster 表示目标集群信息。
type PolicyTargetCluster struct {
	ClusterID  uint   `json:"clusterId,omitempty" yaml:"clusterId,omitempty"`   // 集群 ID
	CNIType    string `json:"cniType,omitempty" yaml:"cniType,omitempty"`       // CNI 类型
	CNIVersion string `json:"cniVersion,omitempty" yaml:"cniVersion,omitempty"` // CNI 版本
}

// PolicyReleaseStatus 表示发布状态与风险信息。
type PolicyReleaseStatus struct {
	Phase     PolicyReleaseState `json:"phase,omitempty" yaml:"phase,omitempty"`         // 发布阶段
	RiskScore int                `json:"riskScore,omitempty" yaml:"riskScore,omitempty"` // 风险分
	RiskLevel PolicyRiskLevel    `json:"riskLevel,omitempty" yaml:"riskLevel,omitempty"` // 风险等级
}

// PolicyIssue 表示仿真发现的问题。
type PolicyIssue struct {
	Code       string              `json:"code,omitempty" yaml:"code,omitempty"`             // 问题码
	Message    string              `json:"message,omitempty" yaml:"message,omitempty"`       // 问题说明
	Severity   PolicyIssueSeverity `json:"severity,omitempty" yaml:"severity,omitempty"`     // 严重级别
	Suggestion string              `json:"suggestion,omitempty" yaml:"suggestion,omitempty"` // 修复建议
}

// PolicyWarning 表示非阻断告警。
type PolicyWarning struct {
	Code    string `json:"code,omitempty" yaml:"code,omitempty"`       // 呆警码
	Message string `json:"message,omitempty" yaml:"message,omitempty"` // 告警说明
}

// PolicyImpactSummary 表示仿真的影响面摘要。
type PolicyImpactSummary struct {
	AffectedPods       int      `json:"affectedPods,omitempty" yaml:"affectedPods,omitempty"`             // 受影响 Pod 数
	AffectedNamespaces []string `json:"affectedNamespaces,omitempty" yaml:"affectedNamespaces,omitempty"` // 受影响命名空间
	NewDeniedFlows     []string `json:"newDeniedFlows,omitempty" yaml:"newDeniedFlows,omitempty"`         // 新增阻断流量
}

// PolicySimulationStatus 表示仿真结果摘要。
type PolicySimulationStatus struct {
	JobID          string              `json:"jobId,omitempty" yaml:"jobId,omitempty"`                   // 仿真任务 ID
	PassedAt       *time.Time          `json:"passedAt,omitempty" yaml:"passedAt,omitempty"`             // 通过时间
	BlockingIssues []PolicyIssue       `json:"blockingIssues,omitempty" yaml:"blockingIssues,omitempty"` // 阻断问题
	Warnings       []PolicyWarning     `json:"warnings,omitempty" yaml:"warnings,omitempty"`             // 呗警列表
	ImpactSummary  PolicyImpactSummary `json:"impactSummary,omitempty" yaml:"impactSummary,omitempty"`   // 影响面摘要
}

// PolicyApprovalStatus 表示审批阶段摘要。
type PolicyApprovalStatus struct {
	Required      bool       `json:"required,omitempty" yaml:"required,omitempty"`           // 是否需要审批
	Approvers     []string   `json:"approvers,omitempty" yaml:"approvers,omitempty"`         // 审批人列表
	ApprovedAt    *time.Time `json:"approvedAt,omitempty" yaml:"approvedAt,omitempty"`       // 审批通过时间
	ApprovalToken string     `json:"approvalToken,omitempty" yaml:"approvalToken,omitempty"` // 审批令牌
}

// PolicyAuditStatus 表示发布审计时间线。
type PolicyAuditStatus struct {
	CreatedAt  *time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`   // 创建时间
	CreatedBy  uint       `json:"createdBy,omitempty" yaml:"createdBy,omitempty"`   // 创建人
	AppliedAt  *time.Time `json:"appliedAt,omitempty" yaml:"appliedAt,omitempty"`   // 应用时间
	RollbackAt *time.Time `json:"rollbackAt,omitempty" yaml:"rollbackAt,omitempty"` // 回滚时间
}

// ApplyDefaults 为发布状态补齐稳定默认值。
func (s *PolicyReleaseStatus) ApplyDefaults() {
	if s.Phase == "" {
		s.Phase = PolicyReleaseStateDraft
	}
	if s.RiskLevel == "" {
		s.RiskLevel = PolicyRiskLevelLow
	}
}

// AllPolicyReleaseStates 返回发布状态机中的稳定状态集合。
func AllPolicyReleaseStates() []PolicyReleaseState {
	return []PolicyReleaseState{
		PolicyReleaseStateDraft,
		PolicyReleaseStateSimulationPending,
		PolicyReleaseStateSimulationPassed,
		PolicyReleaseStateSimulationFailed,
		PolicyReleaseStateApprovalRequired,
		PolicyReleaseStateApplying,
		PolicyReleaseStateApprovalRejected,
		PolicyReleaseStateApplied,
		PolicyReleaseStateApplyFailed,
		PolicyReleaseStateActive,
		PolicyReleaseStateRollbackApplied,
	}
}