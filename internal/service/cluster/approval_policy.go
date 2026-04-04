// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件实现审批票据的作用域绑定、单次消费和审计持久化。
package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
	governanceapproval "github.com/cy77cc/OpsPilot/internal/service/governance/approval"
	governanceaudit "github.com/cy77cc/OpsPilot/internal/service/governance/audit"
	"gorm.io/gorm"
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
	govScope := governanceScopeFromApprovalScope(scope)
	if expiresAt.IsZero() {
		expiresAt = time.Now().UTC().Add(30 * time.Minute)
	}
	svc := governanceapproval.NewServiceWithOptions(db, func() time.Time { return time.Now().UTC() }, time.Until(expiresAt))
	info, err := svc.Issue(ctx, governance.OperationIntent{
		OperatorID: requestedBy,
		Scope:      govScope,
	}, "")
	if err != nil {
		return nil, toClusterApprovalError(err)
	}
	rec := model.ClusterDeployApproval{
		Ticket:     info.Ticket,
		ClusterID:  govScope.ClusterID,
		Namespace:  govScope.Namespace,
		Action:     govScope.Action,
		Resource:   govScope.Resource,
		ResourceID: govScope.ResourceID,
		Status:     "pending",
		RequestBy:  requestedBy,
	}
	if info.ExpiresAt != nil {
		rec.ExpiresAt = *info.ExpiresAt
	}
	return &rec, nil
}

// ConsumeClusterDeployApproval 校验并单次消费审批票据。
func ConsumeClusterDeployApproval(ctx context.Context, db *gorm.DB, ticket string, scope ApprovalScope, consumedBy uint, now time.Time) (*model.ClusterDeployApproval, error) {
	svc := governanceapproval.NewServiceWithOptions(db, func() time.Time {
		if now.IsZero() {
			return time.Now().UTC()
		}
		return now.UTC()
	}, 30*time.Minute)
	govScope := governanceScopeFromApprovalScope(scope)
	intent := governance.OperationIntent{
		OperatorID:    consumedBy,
		ApprovalToken: strings.TrimSpace(ticket),
		Scope:         govScope,
	}
	err := svc.Consume(ctx, intent)
	rec := model.ClusterDeployApproval{
		Ticket:     strings.TrimSpace(ticket),
		ClusterID:  govScope.ClusterID,
		Namespace:  govScope.Namespace,
		Action:     govScope.Action,
		Resource:   govScope.Resource,
		ResourceID: govScope.ResourceID,
	}
	if err != nil {
		return &rec, toClusterApprovalError(err)
	}
	return &rec, nil
}

// PersistClusterOperationAudit 持久化集群操作审计，并对 message 进行脱敏。
func PersistClusterOperationAudit(ctx context.Context, db *gorm.DB, clusterID uint, namespace, action, resource, resourceID, status string, operatorID uint, message any) (*model.ClusterOperationAudit, error) {
	govAudit := governanceaudit.NewService(db, nil)
	msg := strings.TrimSpace(stringifyAuditMessage(message))
	finalize := governance.FinalizeInput{
		Intent: governance.OperationIntent{
			OperatorID: operatorID,
			Scope: governance.Scope{
				Domain:     "cluster",
				ClusterID:  clusterID,
				Namespace:  strings.TrimSpace(namespace),
				Resource:   strings.TrimSpace(resource),
				ResourceID: strings.TrimSpace(resourceID),
				Action:     strings.TrimSpace(action),
			},
		},
		Decision: governance.Decision{
			State:   clusterStatusToGovernanceState(status),
			Code:    clusterStatusToGovernanceCode(status),
			Message: msg,
		},
		ExecutionCode: clusterStatusToGovernanceCode(status),
		ExecutionMsg:  msg,
		Diagnostics: map[string]any{
			"message": RedactAuditPayload(message),
		},
	}
	id, err := govAudit.Record(ctx, finalize)
	if err != nil {
		return nil, err
	}
	return &model.ClusterOperationAudit{
		ID:         id,
		ClusterID:  clusterID,
		Namespace:  strings.TrimSpace(namespace),
		Action:     strings.TrimSpace(action),
		Resource:   strings.TrimSpace(resource),
		ResourceID: strings.TrimSpace(resourceID),
		Status:     strings.TrimSpace(status),
		Message:    msg,
		OperatorID: operatorID,
	}, nil
}

func governanceScopeFromApprovalScope(scope ApprovalScope) governance.Scope {
	scope = NormalizeApprovalScope(scope)
	return governance.Scope{
		Domain:     "cluster",
		ClusterID:  scope.ClusterID,
		Namespace:  scope.Namespace,
		Resource:   scope.Resource,
		ResourceID: scope.ResourceID,
		Action:     scope.Action,
	}
}

func toClusterApprovalError(err error) error {
	if err == nil {
		return nil
	}
	govErr, ok := governance.IsGovError(err)
	if !ok {
		return err
	}
	code := govErr.Code
	switch code {
	case governance.CodeApprovalTokenReplay:
		code = ApprovalTokenReplayedCode
	case governance.CodeApprovalTokenInvalid:
		code = approvalTokenInvalidCode
	case governance.CodeApprovalTokenExpired:
		code = approvalTokenExpiredCode
	case governance.CodeApprovalScopeMismatch:
		code = approvalTokenScopeCode
	case governance.CodeApprovalNotApproved, governance.CodeApprovalRejected:
		code = approvalTokenPendingCode
	}
	return &ApprovalError{Code: code, Message: code}
}

func clusterStatusToGovernanceState(status string) governance.OperationState {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return governance.StateApprovalRequired
	case "failed":
		return governance.StateFailed
	case "rejected":
		return governance.StateRejected
	default:
		return governance.StateCompleted
	}
}

func clusterStatusToGovernanceCode(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return governance.CodeApprovalRequired
	case "failed":
		return governance.CodeInternalError
	case "rejected":
		return governance.CodeApprovalRejected
	default:
		return governance.CodeSuccess
	}
}

func stringifyAuditMessage(message any) string {
	if message == nil {
		return ""
	}
	switch value := message.(type) {
	case string:
		return sanitizeOperationText(value)
	case []byte:
		return sanitizeOperationText(string(value))
	default:
		buf, err := json.Marshal(RedactAuditPayload(value))
		if err != nil {
			return sanitizeOperationText(fmt.Sprint(RedactAuditPayload(value)))
		}
		return sanitizeOperationText(string(buf))
	}
}
