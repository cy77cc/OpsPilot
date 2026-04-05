package cluster

import "time"

const (
	ClusterModePlatformManaged = "platform_managed"
	ClusterModeExternalManaged = "external_managed"
)

type Phase3GateDecision string

const (
	Phase3GateDecisionAllowed          Phase3GateDecision = "allowed"
	Phase3GateDecisionApprovalRequired Phase3GateDecision = "approval_required"
	Phase3GateDecisionRejected         Phase3GateDecision = "rejected"
	Phase3GateDecisionBlocked          Phase3GateDecision = "blocked"
)

type RuntimeContainResult struct {
	EventID     uint      `json:"event_id"`
	Mode        string    `json:"mode"`
	AuditID     uint      `json:"audit_id,omitempty"`
	ApprovalID  uint      `json:"approval_id,omitempty"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updated_at"`
}
