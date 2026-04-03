// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件实现审批票据的作用域绑定、单次消费和审计持久化。
package cluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// ApprovalTokenReplayedCode 是审批票据重复消费的稳定错误码。
	ApprovalTokenReplayedCode = "approval_token_replayed"
	approvalTokenInvalidCode  = "approval_token_invalid"
	approvalTokenExpiredCode  = "approval_token_expired"
	approvalTokenScopeCode    = "approval_token_scope_mismatch"
	approvalTokenPendingCode  = "approval_token_not_approved"
)

// ApprovalScope 描述审批票据绑定的作用域。
type ApprovalScope struct {
	ClusterID  uint
	Namespace  string
	Action     string
	Resource   string
	ResourceID string
}

// ApprovalError 表示审批票据校验/消费过程中的业务错误。
type ApprovalError struct {
	Code    string
	Message string
}

// Error 实现 error 接口。
func (e *ApprovalError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	return e.Code
}

// IsApprovalError 判断是否为审批业务错误。
func IsApprovalError(err error) (*ApprovalError, bool) {
	if err == nil {
		return nil, false
	}
	approvalErr, ok := err.(*ApprovalError)
	return approvalErr, ok
}

// NormalizeApprovalScope 规范化审批作用域，便于比较与持久化。
func NormalizeApprovalScope(scope ApprovalScope) ApprovalScope {
	scope.Namespace = strings.TrimSpace(scope.Namespace)
	scope.Action = strings.ToLower(strings.TrimSpace(scope.Action))
	scope.Resource = strings.ToLower(strings.TrimSpace(scope.Resource))
	scope.ResourceID = strings.TrimSpace(scope.ResourceID)
	if scope.Resource == "" {
		scope.Resource = scope.Action
	}
	if scope.ResourceID == "" {
		scope.ResourceID = scope.Namespace
	}
	return scope
}

// IssueClusterDeployApproval 创建审批票据记录。
func IssueClusterDeployApproval(ctx context.Context, db *gorm.DB, scope ApprovalScope, requestedBy uint, expiresAt time.Time) (*model.ClusterDeployApproval, error) {
	scope = NormalizeApprovalScope(scope)
	if scope.ClusterID == 0 {
		return nil, &ApprovalError{Code: approvalTokenInvalidCode, Message: approvalTokenInvalidCode}
	}
	if strings.TrimSpace(scope.Namespace) == "" || strings.TrimSpace(scope.Action) == "" {
		return nil, &ApprovalError{Code: approvalTokenInvalidCode, Message: approvalTokenInvalidCode}
	}
	if expiresAt.IsZero() {
		expiresAt = time.Now().UTC().Add(30 * time.Minute)
	}
	rec := model.ClusterDeployApproval{
		Ticket:     fmt.Sprintf("k8s-appr-%d", time.Now().UnixNano()),
		ClusterID:  scope.ClusterID,
		Namespace:  scope.Namespace,
		Action:     scope.Action,
		Resource:   scope.Resource,
		ResourceID: scope.ResourceID,
		Status:     "pending",
		RequestBy:  requestedBy,
		ExpiresAt:  expiresAt,
	}
	if err := db.WithContext(ctx).Create(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

// ConsumeClusterDeployApproval 校验并单次消费审批票据。
func ConsumeClusterDeployApproval(ctx context.Context, db *gorm.DB, ticket string, scope ApprovalScope, consumedBy uint, now time.Time) (*model.ClusterDeployApproval, error) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return nil, &ApprovalError{Code: approvalTokenInvalidCode, Message: approvalTokenInvalidCode}
	}
	scope = NormalizeApprovalScope(scope)
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var rec model.ClusterDeployApproval
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("ticket = ?", ticket).
			First(&rec).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return &ApprovalError{Code: approvalTokenInvalidCode, Message: approvalTokenInvalidCode}
			}
			return err
		}
		if rec.ClusterID != scope.ClusterID || !strings.EqualFold(strings.TrimSpace(rec.Namespace), scope.Namespace) || !strings.EqualFold(strings.TrimSpace(rec.Action), scope.Action) || !strings.EqualFold(strings.TrimSpace(rec.Resource), scope.Resource) || strings.TrimSpace(rec.ResourceID) != scope.ResourceID {
			return &ApprovalError{Code: approvalTokenScopeCode, Message: approvalTokenScopeCode}
		}
		if !rec.ExpiresAt.IsZero() && now.After(rec.ExpiresAt) {
			return &ApprovalError{Code: approvalTokenExpiredCode, Message: approvalTokenExpiredCode}
		}
		switch strings.ToLower(strings.TrimSpace(rec.Status)) {
		case "rejected":
			return &ApprovalError{Code: approvalTokenPendingCode, Message: approvalTokenPendingCode}
		case "approved":
			if rec.ConsumedAt != nil {
				replayAt := now
				rec.ReplayCount++
				rec.ReplayAt = &replayAt
				rec.ReplayBy = consumedBy
				rec.ReplayCode = ApprovalTokenReplayedCode
				rec.ReplayMessage = ApprovalTokenReplayedCode
				if err := tx.Save(&rec).Error; err != nil {
					return err
				}
				return &ApprovalError{Code: ApprovalTokenReplayedCode, Message: ApprovalTokenReplayedCode}
			}
			consumedAt := now
			rec.ConsumedAt = &consumedAt
			rec.ConsumedBy = consumedBy
			rec.ReplayCount = 0
			rec.ReplayAt = nil
			rec.ReplayBy = 0
			rec.ReplayCode = ""
			rec.ReplayMessage = ""
			if err := tx.Save(&rec).Error; err != nil {
				return err
			}
			return nil
		default:
			return &ApprovalError{Code: approvalTokenPendingCode, Message: approvalTokenPendingCode}
		}
	}); err != nil {
		if approvalErr, ok := IsApprovalError(err); ok {
			return &rec, approvalErr
		}
		return nil, err
	}
	return &rec, nil
}

// PersistClusterOperationAudit 持久化集群操作审计，并对 message 进行脱敏。
func PersistClusterOperationAudit(ctx context.Context, db *gorm.DB, clusterID uint, namespace, action, resource, resourceID, status string, operatorID uint, message any) (*model.ClusterOperationAudit, error) {
	rec := model.ClusterOperationAudit{
		ClusterID:  clusterID,
		Namespace:  strings.TrimSpace(namespace),
		Action:     strings.TrimSpace(action),
		Resource:   strings.TrimSpace(resource),
		ResourceID: strings.TrimSpace(resourceID),
		Status:     strings.TrimSpace(status),
		Message:    RedactAuditPayload(message),
		OperatorID: operatorID,
	}
	if err := db.WithContext(ctx).Create(&rec).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}
