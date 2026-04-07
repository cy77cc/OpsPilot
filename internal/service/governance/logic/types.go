package logic

import (
	"context"
	"strings"
	"time"
)

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

const (
	TargetScopeCluster = "cluster"
	TargetScopeProject = "project"
	TargetScopeTeam    = "team"
	TargetScopeGlobal  = "global"
)

type Scope struct {
	Domain      string         `json:"domain,omitempty"`
	ClusterID   uint           `json:"cluster_id,omitempty"`
	ProjectID   uint           `json:"project_id,omitempty"`
	TeamID      uint           `json:"team_id,omitempty"`
	Namespace   string         `json:"namespace,omitempty"`
	Environment string         `json:"environment,omitempty"`
	TargetScope string         `json:"target_scope,omitempty"`
	Resource    string         `json:"resource,omitempty"`
	ResourceID  string         `json:"resource_id,omitempty"`
	Action      string         `json:"action,omitempty"`
	Context     map[string]any `json:"context,omitempty"`
}

func (s Scope) Normalize() Scope {
	s.Domain = strings.ToLower(strings.TrimSpace(s.Domain))
	s.Namespace = strings.TrimSpace(s.Namespace)
	s.Environment = strings.ToLower(strings.TrimSpace(s.Environment))
	s.TargetScope = strings.ToLower(strings.TrimSpace(s.TargetScope))
	s.Resource = strings.ToLower(strings.TrimSpace(s.Resource))
	s.ResourceID = strings.TrimSpace(s.ResourceID)
	s.Action = strings.ToLower(strings.TrimSpace(s.Action))
	if s.Context == nil {
		s.Context = map[string]any{}
	}
	return s
}

type OperationContext struct {
	TeamID      uint           `json:"team_id,omitempty"`
	Environment string         `json:"environment,omitempty"`
	Values      map[string]any `json:"values,omitempty"`
}

type operationContextKey struct{}

func WithOperationContext(ctx context.Context, meta OperationContext) context.Context {
	return context.WithValue(ctx, operationContextKey{}, meta)
}

func OperationContextFromContext(ctx context.Context) (OperationContext, bool) {
	if ctx == nil {
		return OperationContext{}, false
	}
	meta, ok := ctx.Value(operationContextKey{}).(OperationContext)
	return meta, ok
}

func MergeScopeFromContext(ctx context.Context, scope Scope) Scope {
	scope = scope.Normalize()
	if ctx == nil {
		return scope
	}
	meta, ok := OperationContextFromContext(ctx)
	if !ok {
		return scope
	}
	if scope.TeamID == 0 {
		scope.TeamID = meta.TeamID
	}
	if scope.Environment == "" {
		scope.Environment = strings.ToLower(strings.TrimSpace(meta.Environment))
	}
	if len(meta.Values) == 0 {
		return scope
	}
	if scope.Context == nil {
		scope.Context = make(map[string]any, len(meta.Values))
	}
	for key, value := range meta.Values {
		if _, exists := scope.Context[key]; !exists {
			scope.Context[key] = value
		}
	}
	return scope
}

type OperationIntent struct {
	RequestID      string         `json:"request_id,omitempty"`
	OperatorID     uint           `json:"operator_id,omitempty"`
	ApprovalToken  string         `json:"approval_token,omitempty"`
	Scope          Scope          `json:"scope,omitempty"`
	RequestSummary map[string]any `json:"request_summary,omitempty"`
	OccurredAt     time.Time      `json:"occurred_at,omitempty"`
}

type ApprovalInfo struct {
	Ticket    string     `json:"ticket,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Reason    string     `json:"reason,omitempty"`
}

type Decision struct {
	Allowed  bool           `json:"allowed"`
	State    OperationState `json:"state"`
	Code     string         `json:"code,omitempty"`
	Message  string         `json:"message,omitempty"`
	Approval *ApprovalInfo  `json:"approval,omitempty"`
}

type FinalizeInput struct {
	Intent        OperationIntent `json:"intent"`
	Decision      Decision        `json:"decision"`
	ExecutionCode string          `json:"execution_code,omitempty"`
	ExecutionMsg  string          `json:"execution_msg,omitempty"`
	Result        map[string]any  `json:"result,omitempty"`
	Diagnostics   map[string]any  `json:"diagnostics,omitempty"`
	StartedAt     time.Time       `json:"started_at,omitempty"`
	FinishedAt    time.Time       `json:"finished_at,omitempty"`
}

type FinalizeOutput struct {
	AuditID uint `json:"audit_id"`
}

type Envelope struct {
	State    OperationState `json:"state"`
	Approval *ApprovalInfo  `json:"approval,omitempty"`
	AuditID  uint           `json:"audit_id,omitempty"`
	Code     string         `json:"code,omitempty"`
	Message  string         `json:"message,omitempty"`
	Data     any            `json:"data,omitempty"`
}

type Policy struct {
	RiskLevel           RiskLevel
	ApprovalRequired    bool
	RequiredPermissions []string
	ApprovalReason      string
}
