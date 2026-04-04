# 2026-04-03 Governance Module Interface Blueprint

## 1. Package Layout

```text
internal/service/governance/
  types.go
  errors.go
  service.go
  policy/
    resolver.go
  approval/
    service.go
  audit/
    service.go
  envelope/
    mapper.go
  adapter/
    cluster.go
    deployment.go
```

## 2. Core Types (`types.go`)

```go
package governance

import "time"

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type OperationState string

const (
	StateCompleted        OperationState = "completed"
	StateApprovalRequired OperationState = "approval_required"
	StateRejected         OperationState = "rejected"
	StateFailed           OperationState = "failed"
)

type Scope struct {
	Domain     string
	ClusterID  uint
	ProjectID  uint
	Namespace  string
	Resource   string
	ResourceID string
	Action     string
}

type OperationIntent struct {
	RequestID      string
	OperatorID     uint
	ApprovalToken  string
	Scope          Scope
	RequestSummary map[string]any
	OccurredAt     time.Time
}

type ApprovalInfo struct {
	Ticket    string     `json:"ticket,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Reason    string     `json:"reason,omitempty"`
}

type Decision struct {
	Allowed  bool
	State    OperationState
	Code     string
	Message  string
	Approval *ApprovalInfo
}

type FinalizeInput struct {
	Intent         OperationIntent
	Decision       Decision
	ExecutionCode  string
	ExecutionMsg   string
	Result         map[string]any
	Diagnostics    map[string]any
	StartedAt      time.Time
	FinishedAt     time.Time
}

type FinalizeOutput struct {
	AuditID uint
}

type Envelope struct {
	State    OperationState `json:"state"`
	Approval *ApprovalInfo  `json:"approval,omitempty"`
	AuditID  uint           `json:"audit_id,omitempty"`
	Code     string         `json:"code,omitempty"`
	Message  string         `json:"message,omitempty"`
	Data     any            `json:"data,omitempty"`
}
```

## 3. Error Codes (`errors.go`)

```go
package governance

const (
	CodeSuccess               = "success"
	CodeApprovalRequired      = "approval_required"
	CodeApprovalRejected      = "approval_rejected"
	CodeApprovalTokenInvalid  = "approval_token_invalid"
	CodeApprovalTokenExpired  = "approval_token_expired"
	CodeApprovalTokenReplay   = "approval_token_replayed"
	CodeApprovalScopeMismatch = "approval_token_scope_mismatch"
	CodePermissionDenied      = "permission_denied"
	CodePolicyNotFound        = "policy_not_found"
	CodeInternalError         = "internal_error"
)

type GovError struct {
	Code    string
	Message string
}

func (e *GovError) Error() string { return e.Message }
```

## 4. Service Contracts (`service.go`)

```go
package governance

import "context"

type Service interface {
	Preflight(ctx context.Context, intent OperationIntent) (Decision, error)
	Finalize(ctx context.Context, in FinalizeInput) (FinalizeOutput, error)
	BuildEnvelope(decision Decision, auditID uint, data any) Envelope
}

type PolicyResolver interface {
	Resolve(ctx context.Context, scope Scope) (Policy, error)
}

type ApprovalService interface {
	Issue(ctx context.Context, intent OperationIntent, reason string) (*ApprovalInfo, error)
	Consume(ctx context.Context, intent OperationIntent) error
	Confirm(ctx context.Context, ticket string, approverID uint, approved bool, note string) error
}

type AuditService interface {
	Record(ctx context.Context, in FinalizeInput) (uint, error)
}

type Redactor interface {
	Redact(v any) any
}

type Policy struct {
	RiskLevel            RiskLevel
	ApprovalRequired     bool
	RequiredPermissions  []string
	ApprovalReason       string
}
```

## 5. Adapter Pattern (`adapter/*.go`)

Domain handlers should use thin adapters instead of direct policy/approval internals.

```go
package adapter

import (
	"context"
	"github.com/cy77cc/OpsPilot/internal/service/governance"
)

type Runner func(ctx context.Context) (map[string]any, error)

type GovernedExecutor interface {
	Execute(ctx context.Context, intent governance.OperationIntent, run Runner) (governance.Envelope, error)
}
```

Reference cluster adapter behavior:

1. Construct `OperationIntent` from request (`cluster_id`, `resource`, `action`, `approval_token`, `operator_id`)
2. Call `Preflight`
3. If not allowed, return envelope immediately
4. Execute business `Runner`
5. Call `Finalize`
6. Return unified envelope

## 6. Handler Integration Skeleton

```go
func (h *Handler) CordonNode(c *gin.Context) {
	intent := clusteradapter.BuildIntentFromCordon(c)
	env, err := h.governedExecutor.Execute(c.Request.Context(), intent, func(ctx context.Context) (map[string]any, error) {
		return h.cordonNode(ctx, client, nodeName, true)
	})
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, env)
}
```

## 7. Compatibility Hooks

During migration, governance approval service should support:

1. Read/consume from new `operation_approvals`
2. Optional compatibility read from legacy `cluster_deploy_approvals`
3. Stable error codes regardless of storage backend

## 8. Minimal First Implementation Scope

1. Implement `Service`, `PolicyResolver`, `ApprovalService`, `AuditService` interfaces
2. Provide cluster adapter only
3. Migrate cluster high-risk endpoints to adapter executor
4. Keep route signatures unchanged

## 9. Conformance Test Interface

```go
package governance_test

type GovernedEndpointCase struct {
	Name string
	Send func(t *testing.T, approvalToken string) Response
}
```

Each governed endpoint must pass:

1. approval required when no token
2. success with valid approved token
3. replay rejection with `approval_token_replayed`
4. scope mismatch rejection
5. expiry rejection
6. envelope shape consistency
