package contracts

import "time"

const (
	OperationStateCompleted        = "completed"
	OperationStateApprovalRequired = "approval_required"
	OperationStateRejected         = "rejected"
	OperationStateFailed           = "failed"
)

const (
	OperationCodeSuccess          = "success"
	OperationCodeApprovalRequired = "approval_required"
	OperationCodeApprovalRejected = "approval_rejected"
	OperationCodeFailed           = "failed"
)

// OperationApproval 描述写操作相关的审批信息。
type OperationApproval struct {
	Required      bool       `json:"required,omitempty"`
	Ticket        string     `json:"ticket,omitempty"`
	ClusterID     uint       `json:"cluster_id,omitempty"`
	Namespace     string     `json:"namespace,omitempty"`
	Action        string     `json:"action,omitempty"`
	Resource      string     `json:"resource,omitempty"`
	ResourceID    string     `json:"resource_id,omitempty"`
	Reason        string     `json:"reason,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	ConsumedAt    *time.Time `json:"consumed_at,omitempty"`
	ConsumedBy    uint       `json:"consumed_by,omitempty"`
	ReplayCount   int        `json:"replay_count,omitempty"`
	ReplayAt      *time.Time `json:"replay_at,omitempty"`
	ReplayBy      uint       `json:"replay_by,omitempty"`
	ReplayCode    string     `json:"replay_code,omitempty"`
	ReplayMessage string     `json:"replay_message,omitempty"`
	Status        string     `json:"status,omitempty"`
}

// OperationResponse 是集群写操作的统一响应信封。
type OperationResponse struct {
	State    string             `json:"state"`
	Approval *OperationApproval `json:"approval,omitempty"`
	AuditID  uint               `json:"audit_id,omitempty"`
	Code     string             `json:"code,omitempty"`
	Message  string             `json:"message,omitempty"`
	Data     any                `json:"data,omitempty"`
}
