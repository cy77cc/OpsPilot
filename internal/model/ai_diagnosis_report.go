package model

import "time"

// AIDiagnosisReport captures the structured diagnosis output generated for successful Phase 1 runs.
type AIDiagnosisReport struct {
	ID                  string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID               string    `gorm:"column:run_id;type:varchar(64);not null;uniqueIndex:uq_ai_diagnosis_reports_run_id" json:"run_id"`
	SessionID           string    `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_diagnosis_reports_session_id" json:"session_id"`
	Summary             string    `gorm:"column:summary;type:longtext;not null" json:"summary"`
	ImpactScope         string    `gorm:"column:impact_scope;type:longtext;not null" json:"impact_scope"`
	SuspectedRootCauses string    `gorm:"column:suspected_root_causes;type:longtext;not null" json:"suspected_root_causes"`
	Evidence            string    `gorm:"column:evidence;type:longtext;not null" json:"evidence"`
	Recommendations     string    `gorm:"column:recommendations;type:longtext;not null" json:"recommendations"`
	RawToolRefs         string    `gorm:"column:raw_tool_refs;type:longtext;not null" json:"raw_tool_refs"`
	Status              string    `gorm:"column:status;type:varchar(32);not null;default:'';index:idx_ai_diagnosis_reports_status" json:"status"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName returns the database table name for AIDiagnosisReport.
func (AIDiagnosisReport) TableName() string {
	return "ai_diagnosis_reports"
}
