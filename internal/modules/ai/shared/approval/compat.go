package approval

import shared "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"

type ApprovalInfo = shared.ApprovalInfo
type ApprovalResult = shared.ApprovalResult
type ApprovalPreview = shared.ApprovalPreview
type ToolRiskConfig = shared.ToolRiskConfig
type ApprovalEvalMeta = shared.ApprovalEvalMeta
type ApprovalDecision = shared.ApprovalDecision
type ApprovalEvaluator = shared.ApprovalEvaluator

const (
	RiskLevelLow           = shared.RiskLevelLow
	RiskLevelMedium        = shared.RiskLevelMedium
	RiskLevelHigh          = shared.RiskLevelHigh
	RiskLevelCritical      = shared.RiskLevelCritical
	DefaultApprovalTimeout = shared.DefaultApprovalTimeout
)
