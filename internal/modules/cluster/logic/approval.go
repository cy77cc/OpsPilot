// Package logic 提供集群服务的规范化业务逻辑层实现。
//
// 本文件实现审批票据的作用域绑定、单次消费和审计持久化。
package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
	"github.com/cy77cc/OpsPilot/internal/modules/governance/model"
	governanceapproval "github.com/cy77cc/OpsPilot/internal/modules/governance/approval"
	governanceaudit "github.com/cy77cc/OpsPilot/internal/modules/governance/audit"
	"gorm.io/gorm"
)

const (
	// ApprovalTokenReplayedCode 是审批票据重复消费的稳定错误码。
	ApprovalTokenReplayedCode = "approval_token_replayed"
	approvalTokenInvalidCode  = "approval_token_invalid"
	// ApprovalTokenExpiredCode 是审批票据过期的稳定错误码。
	ApprovalTokenExpiredCode = "approval_token_expired"
	approvalTokenScopeCode   = "approval_token_scope_mismatch"

	// PolicyReleaseApprovalResource 是策略发布审批记录绑定的资源类型。
	PolicyReleaseApprovalResource = "policy_release"
	// PolicyReleaseApprovalActionApply 是策略发布应用动作的审批 action。
	PolicyReleaseApprovalActionApply = "policy.apply"
	// PolicyReleaseApprovalActionRollback 是策略发布回滚动作的审批 action。
	PolicyReleaseApprovalActionRollback = "policy.rollback"
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

// PolicyReleaseApprovalScope 构造策略发布审批绑定作用域。
func PolicyReleaseApprovalScope(clusterID uint, namespace string, releaseID uint, action string) ApprovalScope {
	action = strings.TrimSpace(action)
	if action == "" {
		action = PolicyReleaseApprovalActionApply
	}
	return NormalizeApprovalScope(ApprovalScope{
		ClusterID:  clusterID,
		Namespace:  namespace,
		Action:     action,
		Resource:   PolicyReleaseApprovalResource,
		ResourceID: strconv.FormatUint(uint64(releaseID), 10),
	})
}

// IssuePolicyReleaseApproval 创建策略发布审批票据。
func IssuePolicyReleaseApproval(ctx context.Context, db *gorm.DB, clusterID uint, namespace string, releaseID uint, action string, requestedBy uint, expiresAt time.Time) (*clustermodel.ClusterDeployApproval, error) {
	return IssueClusterDeployApproval(ctx, db, PolicyReleaseApprovalScope(clusterID, namespace, releaseID, action), requestedBy, expiresAt)
}

// ConsumePolicyReleaseApproval 校验并消费策略发布审批票据。
func ConsumePolicyReleaseApproval(ctx context.Context, db *gorm.DB, clusterID uint, namespace string, releaseID uint, action, ticket string, consumedBy uint, now time.Time) (*clustermodel.ClusterDeployApproval, error) {
	return ConsumeClusterDeployApproval(ctx, db, ticket, PolicyReleaseApprovalScope(clusterID, namespace, releaseID, action), consumedBy, now)
}

// IssueClusterDeployApproval 创建审批票据记录。
func IssueClusterDeployApproval(ctx context.Context, db *gorm.DB, scope ApprovalScope, requestedBy uint, expiresAt time.Time) (*clustermodel.ClusterDeployApproval, error) {
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
	rec := clustermodel.ClusterDeployApproval{
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
func ConsumeClusterDeployApproval(ctx context.Context, db *gorm.DB, ticket string, scope ApprovalScope, consumedBy uint, now time.Time) (*clustermodel.ClusterDeployApproval, error) {
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
	if err != nil {
		// Backward compatibility: approvals issued before context sentinel
		// rollout used empty scope context. Retry once with legacy scope when
		// the first attempt fails due to scope mismatch.
		if govErr, ok := governance.IsGovError(err); ok && govErr.Code == governance.CodeApprovalScopeMismatch {
			legacyIntent := intent
			legacyIntent.Scope = governanceScopeFromApprovalScopeLegacy(scope)
			err = svc.Consume(ctx, legacyIntent)
		}
	}
	rec := clustermodel.ClusterDeployApproval{
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
func PersistClusterOperationAudit(ctx context.Context, db *gorm.DB, clusterID uint, namespace, action, resource, resourceID, status string, operatorID uint, message any) (*clustermodel.ClusterOperationAudit, error) {
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
	return &clustermodel.ClusterOperationAudit{
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
		// Keep cluster approval scopes comparable through the governance approval
		// service, which currently treats an empty context payload as mismatch on
		// consume. The sentinel is internal-only and stable across issue/consume.
		Context: map[string]any{
			"approval_scope": "cluster",
		},
	}
}

func governanceScopeFromApprovalScopeLegacy(scope ApprovalScope) governance.Scope {
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
		code = ApprovalTokenExpiredCode
	case governance.CodeApprovalScopeMismatch:
		code = approvalTokenScopeCode
	case governance.CodeApprovalNotApproved:
		code = OperationCodeApprovalRequired
	case governance.CodeApprovalRejected:
		code = OperationCodeApprovalRejected
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
		return SanitizeOperationText(value)
	case []byte:
		return SanitizeOperationText(string(value))
	default:
		buf, err := json.Marshal(RedactAuditPayload(value))
		if err != nil {
			return SanitizeOperationText(fmt.Sprint(RedactAuditPayload(value)))
		}
		return SanitizeOperationText(string(buf))
	}
}
