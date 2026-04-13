package model

import (
	"time"

	"gorm.io/gorm"
)

type AIDiagnosisReport struct {
	ID                  string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID               string         `gorm:"column:run_id;type:varchar(64);not null;uniqueIndex" json:"run_id"`
	SessionID           string         `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_diagnosis_reports_session_created,priority:1" json:"session_id"`
	Summary             string         `gorm:"column:summary;type:text;not null" json:"summary"`
	ReportJSON          string         `gorm:"column:report_json;type:text" json:"report_json"`
	EvidenceJSON        string         `gorm:"column:evidence_json;type:text" json:"evidence_json"`
	RootCausesJSON      string         `gorm:"column:root_causes_json;type:text" json:"root_causes_json"`
	RecommendationsJSON string         `gorm:"column:recommendations_json;type:text" json:"recommendations_json"`
	RiskLevel           string         `gorm:"column:risk_level;type:varchar(16)" json:"risk_level"`
	GeneratedAt         time.Time      `gorm:"column:generated_at;autoCreateTime" json:"generated_at"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_ai_diagnosis_reports_session_created,priority:2,sort:desc" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIDiagnosisReport) TableName() string { return "ai_diagnosis_reports" }
