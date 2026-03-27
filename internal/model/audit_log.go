package model

import "time"

// AuditLog 审计日志
type AuditLog struct {
	ID           uint                   `gorm:"primaryKey" json:"id"`
	ActionType   string                 `gorm:"type:varchar(64);not null;index" json:"action_type"`
	ResourceType string                 `gorm:"type:varchar(64);not null;index" json:"resource_type"`
	ResourceID   uint                   `gorm:"not null;index" json:"resource_id"`
	ActorID      uint                   `gorm:"not null;index" json:"actor_id"`
	ActorName    string                 `gorm:"type:varchar(255)" json:"actor_name"`
	Detail       map[string]interface{} `gorm:"type:json" json:"detail"`
	CreatedAt    time.Time              `json:"created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// AuditLog action types
const (
	AuditActionReleaseApply    = "release_apply"
	AuditActionReleaseApprove  = "release_approve"
	AuditActionReleaseReject   = "release_reject"
	AuditActionReleaseRollback = "release_rollback"
	AuditActionTargetCreate    = "target_create"
	AuditActionTargetUpdate    = "target_update"
	AuditActionTargetDelete    = "target_delete"
	AuditActionClusterBootstrap = "cluster_bootstrap"
	AuditActionCredentialCreate = "credential_create"
	AuditActionCredentialTest   = "credential_test"
)

// AuditLog resource types
const (
	AuditResourceRelease    = "release"
	AuditResourceTarget     = "target"
	AuditResourceCluster    = "cluster"
	AuditResourceCredential = "credential"
)
