// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件定义集群写操作的标准响应信封，统一承载完成、审批待定、
// 拒绝和失败等状态。
package cluster

import (
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
)

const (
	// OperationStateCompleted 表示操作已完成。
	OperationStateCompleted = "completed"
	// OperationStateApprovalRequired 表示操作需要审批。
	OperationStateApprovalRequired = "approval_required"
	// OperationStateRejected 表示操作被拒绝。
	OperationStateRejected = "rejected"
	// OperationStateFailed 表示操作失败。
	OperationStateFailed = "failed"
)

const (
	// OperationCodeSuccess 是已完成操作的规范业务码。
	OperationCodeSuccess = "success"
	// OperationCodeApprovalRequired 是审批待定操作的规范业务码。
	OperationCodeApprovalRequired = "approval_required"
	// OperationCodeApprovalRejected 是审批拒绝操作的规范业务码。
	OperationCodeApprovalRejected = "approval_rejected"
	// OperationCodeFailed 是失败操作的规范业务码。
	OperationCodeFailed = "failed"
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

// NewOperationResponse 创建标准写响应信封。
func NewOperationResponse(state string, approval *OperationApproval, auditID uint, code, message string, data any) OperationResponse {
	return OperationResponse{
		State:    state,
		Approval: approval,
		AuditID:  auditID,
		Code:     code,
		Message:  message,
		Data:     data,
	}
}

// NewCompletedOperationResponse 构建已完成状态的响应信封。
func NewCompletedOperationResponse(auditID uint, data any) OperationResponse {
	return NewOperationResponse(OperationStateCompleted, nil, auditID, OperationCodeSuccess, "", data)
}

// NewApprovalRequiredOperationResponse 构建审批待定状态的响应信封。
func NewApprovalRequiredOperationResponse(approval *OperationApproval, auditID uint, message string, data any) OperationResponse {
	return NewOperationResponse(OperationStateApprovalRequired, approval, auditID, OperationCodeApprovalRequired, message, data)
}

// NewRejectedOperationResponse 构建已拒绝状态的响应信封。
func NewRejectedOperationResponse(auditID uint, message string, data any) OperationResponse {
	return NewOperationResponse(OperationStateRejected, nil, auditID, OperationCodeApprovalRejected, message, data)
}

// NewFailedOperationResponse 构建失败状态的响应信封。
func NewFailedOperationResponse(auditID uint, code, message string, data any) OperationResponse {
	if code == "" {
		code = OperationCodeFailed
	}
	return NewOperationResponse(OperationStateFailed, nil, auditID, code, message, data)
}

// OperationApprovalFromRecord 将审批记录转换为响应信封中的审批信息。
func OperationApprovalFromRecord(rec *model.ClusterDeployApproval) *OperationApproval {
	if rec == nil {
		return nil
	}
	return &OperationApproval{
		Required:      true,
		Ticket:        rec.Ticket,
		ClusterID:     rec.ClusterID,
		Namespace:     rec.Namespace,
		Action:        rec.Action,
		Resource:      rec.Resource,
		ResourceID:    rec.ResourceID,
		ExpiresAt:     timePtrOrNil(rec.ExpiresAt),
		ConsumedAt:    rec.ConsumedAt,
		ConsumedBy:    rec.ConsumedBy,
		ReplayCount:   rec.ReplayCount,
		ReplayAt:      rec.ReplayAt,
		ReplayBy:      rec.ReplayBy,
		ReplayCode:    rec.ReplayCode,
		ReplayMessage: rec.ReplayMessage,
		Status:        rec.Status,
	}
}

func timePtrOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
