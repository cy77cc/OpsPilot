package model

import "time"

const (
	DisposalModeAuto        = "auto"
	DisposalModeManual      = "manual"
	DisposalModeSuggestOnly = "suggest_only"
)

const (
	SecuritySeverityCritical = "critical"
	SecuritySeverityHigh     = "high"
	SecuritySeverityMedium   = "medium"
	SecuritySeverityLow      = "low"
)

const (
	SecurityEventSourceFalco    = "falco"
	SecurityEventSourceTetragon = "tetragon"
)

type AdmissionPolicy struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`
	ClusterID uint      `gorm:"column:cluster_id;not null;index" json:"cluster_id"`
	PolicyName string   `gorm:"column:policy_name;type:varchar(191);not null;index" json:"policy_name"`
	Version   string    `gorm:"column:version;type:varchar(64);not null" json:"version"`
	Status    string    `gorm:"column:status;type:varchar(32);not null;default:'draft';index" json:"status"`
	ContentJSON string  `gorm:"column:content_json;type:longtext;not null;default:''" json:"content_json"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AdmissionPolicy) TableName() string { return "admission_policies" }

type AdmissionExemption struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`
	ClusterID  uint      `gorm:"column:cluster_id;not null;index" json:"cluster_id"`
	ScopeType  string    `gorm:"column:scope_type;type:varchar(32);not null;index" json:"scope_type"`
	ScopeRef   string    `gorm:"column:scope_ref;type:varchar(255);not null;index" json:"scope_ref"`
	Reason     string    `gorm:"column:reason;type:text;not null" json:"reason"`
	ApprovalID uint      `gorm:"column:approval_id;not null;default:0;index" json:"approval_id"`
	Status     string    `gorm:"column:status;type:varchar(32);not null;default:'pending';index" json:"status"`
	ExpiresAt  time.Time `gorm:"column:expires_at;not null;index" json:"expires_at"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AdmissionExemption) TableName() string { return "admission_exemptions" }

type RuntimeSecurityEvent struct {
	ID             uint      `gorm:"primaryKey;column:id" json:"id"`
	ClusterID      uint      `gorm:"column:cluster_id;not null;index" json:"cluster_id"`
	Namespace      string    `gorm:"column:namespace;type:varchar(191);not null;default:'';index" json:"namespace"`
	Workload       string    `gorm:"column:workload;type:varchar(191);not null;default:'';index" json:"workload"`
	RuleID         string    `gorm:"column:rule_id;type:varchar(191);not null;default:'';index" json:"rule_id"`
	Severity       string    `gorm:"column:severity;type:varchar(32);not null;index" json:"severity"`
	Source         string    `gorm:"column:source;type:varchar(32);not null;index" json:"source"`
	RawPayloadJSON string    `gorm:"column:raw_payload_json;type:longtext;not null;default:''" json:"raw_payload_json"`
	DisposeStatus  string    `gorm:"column:dispose_status;type:varchar(32);not null;default:'pending';index" json:"dispose_status"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RuntimeSecurityEvent) TableName() string { return "runtime_security_events" }
