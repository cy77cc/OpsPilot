package chat

import (
	aidaodiagnosis "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/diagnosis"
	"gorm.io/gorm"
)

func NewAIDiagnosisReportDAO(db *gorm.DB) *aidaodiagnosis.AIDiagnosisReportDAO {
	return aidaodiagnosis.NewAIDiagnosisReportDAO(db)
}
