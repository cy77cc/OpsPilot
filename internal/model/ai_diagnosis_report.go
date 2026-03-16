package model

import "time"

// AIDiagnosisReport stores the structured diagnosis output for a completed run.
type AIDiagnosisReport struct {
	ID                  string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID               string    `gorm:"column:run_id;type:varchar(64);index:idx_ai_diagnosis_reports_run_id" json:"run_id"`
	SessionID           string    `gorm:"column:session_id;type:varchar(64);index:idx_ai_diagnosis_reports_session_id" json:"session_id"`
	Summary             string    `gorm:"column:summary;type:longtext" json:"summary"`
	EvidenceJSON        string    `gorm:"column:evidence_json;type:longtext" json:"evidence_json"`
	RootCausesJSON      string    `gorm:"column:root_causes_json;type:longtext" json:"root_causes_json"`
	RecommendationsJSON string    `gorm:"column:recommendations_json;type:longtext" json:"recommendations_json"`
	GeneratedAt         time.Time `gorm:"column:generated_at" json:"generated_at"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AIDiagnosisReport) TableName() string { return "ai_diagnosis_reports" }
