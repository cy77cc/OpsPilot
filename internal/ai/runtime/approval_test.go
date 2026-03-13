package runtime

import (
	"context"
	"testing"
)

func TestApprovalDecisionMakerUsesEnvironmentPolicy(t *testing.T) {
	maker := NewApprovalDecisionMaker(ApprovalDecisionMakerOptions{
		ResolveScene: func(string) ResolvedScene {
			return ResolvedScene{
				SceneKey: "deployment",
				SceneConfig: SceneConfig{
					ApprovalConfig: &SceneApprovalConfig{
						DefaultPolicy: ApprovalPolicy{
							RequireApprovalFor: []string{"high"},
						},
						EnvironmentPolicies: map[string]ApprovalPolicy{
							"production": {
								RequireForAllMutating: true,
							},
						},
					},
				},
			}
		},
		LookupTool: func(string) (ApprovalToolSpec, bool) {
			return ApprovalToolSpec{Name: "service_deploy_apply", Mode: "mutating", Risk: "medium"}, true
		},
	})

	decision, err := maker.Decide(context.Background(), ApprovalCheckRequest{
		ToolName:    "service_deploy_apply",
		Scene:       "deployment",
		Environment: "production",
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if !decision.NeedApproval {
		t.Fatalf("NeedApproval = false, want true")
	}
	if decision.PolicySource != "environment:production" {
		t.Fatalf("PolicySource = %q", decision.PolicySource)
	}
}

func TestApprovalDecisionMakerSkipsOnNamespaceCondition(t *testing.T) {
	maker := NewApprovalDecisionMaker(ApprovalDecisionMakerOptions{
		ResolveScene: func(string) ResolvedScene {
			return ResolvedScene{
				SceneKey: "deployment",
				SceneConfig: SceneConfig{
					ApprovalConfig: &SceneApprovalConfig{
						DefaultPolicy: ApprovalPolicy{
							RequireApprovalFor: []string{"medium", "high"},
							SkipConditions: []SkipCondition{
								{Type: "namespace", Pattern: "dev-*"},
							},
						},
					},
				},
			}
		},
		LookupTool: func(string) (ApprovalToolSpec, bool) {
			return ApprovalToolSpec{Name: "service_deploy_apply", Mode: "mutating", Risk: "medium"}, true
		},
	})

	decision, err := maker.Decide(context.Background(), ApprovalCheckRequest{
		ToolName: "service_deploy_apply",
		Scene:    "deployment",
		Params:   map[string]any{"namespace": "dev-blue"},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decision.NeedApproval {
		t.Fatalf("NeedApproval = true, want false")
	}
	if decision.Reason != "approval skipped by policy condition" {
		t.Fatalf("Reason = %q", decision.Reason)
	}
}

func TestApprovalDecisionMakerInfersEnvironmentFromNamespace(t *testing.T) {
	maker := NewApprovalDecisionMaker(ApprovalDecisionMakerOptions{
		ResolveScene: func(string) ResolvedScene { return ResolvedScene{} },
		LookupTool: func(string) (ApprovalToolSpec, bool) {
			return ApprovalToolSpec{Name: "host_batch_exec_apply", Mode: "mutating", Risk: "high"}, true
		},
	})

	decision, err := maker.Decide(context.Background(), ApprovalCheckRequest{
		ToolName: "host_batch_exec_apply",
		Params:   map[string]any{"namespace": "prod-system"},
	})
	if err != nil {
		t.Fatalf("Decide() error = %v", err)
	}
	if decision.Environment != "production" {
		t.Fatalf("Environment = %q, want %q", decision.Environment, "production")
	}
	if !decision.NeedApproval {
		t.Fatalf("NeedApproval = false, want true")
	}
}
