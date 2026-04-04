package policy

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/service/governance"
)

// Rule describes a policy match entry.
type Rule struct {
	Domain              string
	Resource            string
	Action              string
	Namespace           string
	Environment         string
	TargetScope         string
	ClusterID           *uint
	ProjectID           *uint
	TeamID              *uint
	Context             map[string]any
	RiskLevel           governance.RiskLevel
	ApprovalRequired    bool
	RequiredPermissions []string
	ApprovalReason      string
}

// Resolver evaluates a scope against an ordered set of rules.
type Resolver struct {
	Rules         []Rule
	DefaultPolicy *governance.Policy
	AllowFallback bool
}

func NewResolver(rules ...Rule) *Resolver {
	return &Resolver{Rules: rules}
}

func (r *Resolver) Resolve(ctx context.Context, scope governance.Scope) (governance.Policy, error) {
	scope = governance.MergeScopeFromContext(ctx, scope)
	for _, rule := range r.Rules {
		if rule.matches(scope) {
			return governance.Policy{
				RiskLevel:           rule.RiskLevel,
				ApprovalRequired:    rule.ApprovalRequired,
				RequiredPermissions: append([]string(nil), rule.RequiredPermissions...),
				ApprovalReason:      strings.TrimSpace(rule.ApprovalReason),
			}, nil
		}
	}

	if r.AllowFallback && r.DefaultPolicy != nil {
		return *r.DefaultPolicy, nil
	}
	return governance.Policy{}, governance.NewGovError(governance.CodePolicyNotFound, "policy not found")
}

func (r Rule) matches(scope governance.Scope) bool {
	scope = scope.Normalize()
	if !matchString(r.Domain, scope.Domain) {
		return false
	}
	if !matchString(r.Resource, scope.Resource) {
		return false
	}
	if !matchString(r.Action, scope.Action) {
		return false
	}
	if !matchString(r.Namespace, scope.Namespace) {
		return false
	}
	if !matchString(r.Environment, scope.Environment) {
		return false
	}
	if !matchString(r.TargetScope, scope.TargetScope) {
		return false
	}
	if r.ClusterID != nil && *r.ClusterID != scope.ClusterID {
		return false
	}
	if r.ProjectID != nil && *r.ProjectID != scope.ProjectID {
		return false
	}
	if r.TeamID != nil && *r.TeamID != scope.TeamID {
		return false
	}
	if len(r.Context) > 0 && !contextMatches(r.Context, scope.Context) {
		return false
	}
	return true
}

func matchString(expected, actual string) bool {
	expected = strings.ToLower(strings.TrimSpace(expected))
	actual = strings.ToLower(strings.TrimSpace(actual))
	return expected == "" || expected == actual
}

func contextMatches(expected, actual map[string]any) bool {
	if len(expected) == 0 {
		return true
	}
	for key, value := range expected {
		actualValue, ok := actual[key]
		if !ok {
			return false
		}
		left, err := json.Marshal(actualValue)
		if err != nil {
			return false
		}
		right, err := json.Marshal(value)
		if err != nil {
			return false
		}
		if string(left) != string(right) {
			return false
		}
	}
	return true
}
