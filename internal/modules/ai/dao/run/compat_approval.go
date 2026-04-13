package run

import (
	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	riskpolicy "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/risk_policy"
	"gorm.io/gorm"
)

func NewAIApprovalTaskDAO(db *gorm.DB) *aidaoapproval.AIApprovalTaskDAO {
	return aidaoapproval.NewAIApprovalTaskDAO(db)
}

func NewAIApprovalOutboxDAO(db *gorm.DB) *aidaoapproval.AIApprovalOutboxDAO {
	return aidaoapproval.NewAIApprovalOutboxDAO(db)
}

func NewAIToolRiskPolicyDAO(db *gorm.DB) *riskpolicy.AIToolRiskPolicyDAO {
	return riskpolicy.NewAIToolRiskPolicyDAO(db)
}
