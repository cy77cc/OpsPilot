package approval

import (
	riskpolicy "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/risk_policy"
	"gorm.io/gorm"
)

func NewAIToolRiskPolicyDAO(db *gorm.DB) *riskpolicy.AIToolRiskPolicyDAO {
	return riskpolicy.NewAIToolRiskPolicyDAO(db)
}
