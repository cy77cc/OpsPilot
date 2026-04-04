// Package model provides database model definitions.
//
// This file defines generic project-wide governance persistence models.
package model

import "time"

// OperationApproval stores a generic approval ticket and its lifecycle state.
type OperationApproval struct {
	ID             uint       `gorm:"primaryKey;column:id" json:"id"`
	Ticket         string     `gorm:"column:ticket;type:varchar(96);not null;uniqueIndex" json:"ticket"`
	Domain         string     `gorm:"column:domain;type:varchar(64);not null;default:'';index" json:"domain"`
	ScopeClusterID *uint      `gorm:"column:scope_cluster_id;index" json:"scope_cluster_id,omitempty"`
	ScopeProjectID *uint      `gorm:"column:scope_project_id;index" json:"scope_project_id,omitempty"`
	ScopeTeamID    *uint      `gorm:"column:scope_team_id;index" json:"scope_team_id,omitempty"`
	Namespace      string     `gorm:"column:namespace;type:varchar(128);not null;default:'';index" json:"namespace"`
	Environment    string     `gorm:"column:environment;type:varchar(32);not null;default:'';index" json:"environment"`
	Resource       string     `gorm:"column:resource;type:varchar(64);not null;default:'';index" json:"resource"`
	ResourceID     string     `gorm:"column:resource_id;type:varchar(128);not null;default:'';index" json:"resource_id"`
	Action         string     `gorm:"column:action;type:varchar(64);not null;default:'';index" json:"action"`
	ContextJSON    string     `gorm:"column:context_json;type:longtext;not null;default:''" json:"context_json"`
	Reason         string     `gorm:"column:reason;type:varchar(255);not null;default:''" json:"reason"`
	Status         string     `gorm:"column:status;type:varchar(32);not null;default:'pending';index" json:"status"`
	RequestBy      uint       `gorm:"column:request_by;not null;default:0" json:"request_by"`
	ReviewBy       uint       `gorm:"column:review_by;not null;default:0" json:"review_by"`
	ExpiresAt      *time.Time `gorm:"column:expires_at;index" json:"expires_at,omitempty"`
	ConsumedAt     *time.Time `gorm:"column:consumed_at;index" json:"consumed_at,omitempty"`
	ConsumedBy     uint       `gorm:"column:consumed_by;not null;default:0" json:"consumed_by"`
	ReplayCount    int        `gorm:"column:replay_count;not null;default:0" json:"replay_count"`
	ReplayAt       *time.Time `gorm:"column:replay_at" json:"replay_at,omitempty"`
	ReplayBy       uint       `gorm:"column:replay_by;not null;default:0" json:"replay_by"`
	ReplayCode     string     `gorm:"column:replay_code;type:varchar(64);not null;default:''" json:"replay_code"`
	ReplayMessage  string     `gorm:"column:replay_message;type:varchar(255);not null;default:''" json:"replay_message"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName returns the approval table name.
func (OperationApproval) TableName() string {
	return "operation_approvals"
}

// OperationAudit stores a redacted record of a governance-controlled operation.
type OperationAudit struct {
	ID                 uint      `gorm:"primaryKey;column:id" json:"id"`
	Domain             string    `gorm:"column:domain;type:varchar(64);not null;default:'';index" json:"domain"`
	ScopeClusterID     *uint     `gorm:"column:scope_cluster_id;index" json:"scope_cluster_id,omitempty"`
	ScopeProjectID     *uint     `gorm:"column:scope_project_id;index" json:"scope_project_id,omitempty"`
	ScopeTeamID        *uint     `gorm:"column:scope_team_id;index" json:"scope_team_id,omitempty"`
	Namespace          string    `gorm:"column:namespace;type:varchar(128);not null;default:'';index" json:"namespace"`
	Environment        string    `gorm:"column:environment;type:varchar(32);not null;default:'';index" json:"environment"`
	Resource           string    `gorm:"column:resource;type:varchar(64);not null;default:'';index" json:"resource"`
	ResourceID         string    `gorm:"column:resource_id;type:varchar(128);not null;default:'';index" json:"resource_id"`
	Action             string    `gorm:"column:action;type:varchar(64);not null;default:'';index" json:"action"`
	OperatorID         uint      `gorm:"column:operator_id;not null;default:0;index" json:"operator_id"`
	Status             string    `gorm:"column:status;type:varchar(32);not null;default:'success';index" json:"status"`
	Code               string    `gorm:"column:code;type:varchar(64);not null;default:'';index" json:"code"`
	Message            string    `gorm:"column:message;type:varchar(255);not null;default:''" json:"message"`
	RequestSummaryJSON string    `gorm:"column:request_summary_json;type:longtext;not null;default:''" json:"request_summary_json"`
	ResultSummaryJSON  string    `gorm:"column:result_summary_json;type:longtext;not null;default:''" json:"result_summary_json"`
	DiagnosticsJSON    string    `gorm:"column:diagnostics_json;type:longtext;not null;default:''" json:"diagnostics_json"`
	ApprovalTicket     string    `gorm:"column:approval_ticket;type:varchar(96);not null;default:'';index" json:"approval_ticket"`
	LatencyMS          int64     `gorm:"column:latency_ms;not null;default:0" json:"latency_ms"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName returns the audit table name.
func (OperationAudit) TableName() string {
	return "operation_audits"
}
