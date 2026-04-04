package policy

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/service/governance"
)

func TestResolver_UsesTeamEnvironmentAndExtensibleContext(t *testing.T) {
	teamID := uint(12)
	resolver := NewResolver(Rule{
		Domain:           "cluster",
		Resource:         "node",
		Action:           "cordon",
		Namespace:        "prod-ns",
		Environment:      "production",
		TeamID:           &teamID,
		Context:          map[string]any{"tier": "critical"},
		RiskLevel:        governance.RiskHigh,
		ApprovalRequired: true,
		ApprovalReason:   "production operations require approval",
	})

	ctx := governance.WithOperationContext(context.Background(), governance.OperationContext{
		TeamID:      teamID,
		Environment: "production",
		Values:      map[string]any{"tier": "critical"},
	})
	policy, err := resolver.Resolve(ctx, governance.Scope{
		Domain:    "cluster",
		Resource:  "node",
		Action:    "cordon",
		Namespace: "prod-ns",
	})
	if err != nil {
		t.Fatalf("resolve policy: %v", err)
	}
	if !policy.ApprovalRequired {
		t.Fatalf("expected approval required policy, got %#v", policy)
	}
	if policy.RiskLevel != governance.RiskHigh {
		t.Fatalf("expected risk high, got %#v", policy)
	}
	if policy.ApprovalReason == "" {
		t.Fatalf("expected approval reason to be preserved")
	}
}

func TestResolver_DoesNotMatchWrongEnvironment(t *testing.T) {
	teamID := uint(12)
	resolver := NewResolver(Rule{
		Domain:      "cluster",
		Resource:    "node",
		Action:      "cordon",
		TeamID:      &teamID,
		Environment: "production",
	})

	ctx := governance.WithOperationContext(context.Background(), governance.OperationContext{
		TeamID:      teamID,
		Environment: "staging",
	})
	_, err := resolver.Resolve(ctx, governance.Scope{
		Domain:   "cluster",
		Resource: "node",
		Action:   "cordon",
	})
	if err == nil {
		t.Fatalf("expected error for unmatched environment")
	}
	if !governance.IsCode(err, governance.CodePolicyNotFound) {
		t.Fatalf("expected policy not found error, got %v", err)
	}
}
