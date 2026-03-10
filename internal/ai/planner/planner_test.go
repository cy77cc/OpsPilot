package planner

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
)

func TestPlanFallsBackToMinimalPlan(t *testing.T) {
	out, err := New(nil).Plan(context.Background(), Input{
		Message: "查看所有主机的状态",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看所有主机的状态",
			OperationMode:  "query",
			NormalizedRequest: rewrite.NormalizedRequest{
				Intent: "service_health_check",
				Targets: []rewrite.RequestTarget{
					{Type: "host", Name: "all"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if out.Type != DecisionPlan {
		t.Fatalf("Type = %s, want %s", out.Type, DecisionPlan)
	}
	if out.Plan == nil || len(out.Plan.Steps) != 1 {
		t.Fatalf("Plan = %#v", out.Plan)
	}
	if out.Plan.Steps[0].Expert != "hostops" {
		t.Fatalf("Expert = %q, want hostops", out.Plan.Steps[0].Expert)
	}
	if out.Plan.Steps[0].Task != "查看所有主机的状态" {
		t.Fatalf("Task = %q", out.Plan.Steps[0].Task)
	}
}

func TestPlanFallsBackToClarifyWhenRewriteStillAmbiguous(t *testing.T) {
	out, err := New(nil).Plan(context.Background(), Input{
		Message: "帮我看看状态",
		Rewrite: rewrite.Output{
			AmbiguityFlags: []string{"resource_target_not_explicit"},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if out.Type != DecisionClarify {
		t.Fatalf("Type = %s, want %s", out.Type, DecisionClarify)
	}
	if len(out.Candidates) != 1 {
		t.Fatalf("Candidates = %#v", out.Candidates)
	}
}

func TestNormalizeDecisionDoesNotPanicWhenBaseHasNoPlan(t *testing.T) {
	base := Decision{
		Type:      DecisionClarify,
		Message:   "need more info",
		Narrative: "clarify first",
	}
	parsed := Decision{
		Type: DecisionPlan,
		Plan: &ExecutionPlan{
			Goal: "check mysql-0",
		},
	}

	out := normalizeDecision(base, parsed)
	if out.Plan == nil {
		t.Fatalf("Plan is nil")
	}
	if out.Plan.Goal != "check mysql-0" {
		t.Fatalf("Goal = %q", out.Plan.Goal)
	}
}

func TestParseDecisionAcceptsNumericStepIDsAndNormalizesPlan(t *testing.T) {
	raw := `{"narrative":"Target pod identified","plan":{"plan_id":"plan_pod_log_analysis_001","goal":"Fetch logs and analyze health","resolved":{"cluster_id":1,"pod_name":"mysql-0","namespace":"default"},"narrative":"Execute log retrieval and health analysis.","steps":[{"step_id":1,"title":"Retrieve Pod Logs","expert":"k8s","intent":"Fetch logs","task":"Retrieve logs","depends_on":[],"mode":"query","risk":"low","narrative":"Get logs"},{"step_id":2,"title":"Analyze Pod Health","expert":"analysis","intent":"Assess health","task":"Analyze logs","depends_on":[1],"mode":"analysis","risk":"low","narrative":"Interpret logs"}]},"type":"plan"}`

	parsed, err := ParseDecision(raw)
	if err != nil {
		t.Fatalf("ParseDecision() error = %v", err)
	}
	base := buildBaseDecision(Input{
		Message: "查看 mysql-0 pod 最近 100 条日志并分析运行状况",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看 mysql-0 pod 最近 100 条日志并分析运行状况",
			OperationMode:  "query",
		},
	})
	out := normalizeDecision(base, parsed)
	if out.Plan == nil {
		t.Fatalf("normalized plan is nil")
	}
	if got := out.Plan.Resolved.ClusterID; got != 1 {
		t.Fatalf("cluster id = %d, want %d", got, 1)
	}
	if got := out.Plan.Steps[0].StepID; got != "1" {
		t.Fatalf("step 1 id = %q, want %q", got, "1")
	}
	if got := out.Plan.Steps[0].Mode; got != "readonly" {
		t.Fatalf("step 1 mode = %q, want readonly", got)
	}
	if got := out.Plan.Steps[1].Expert; got != "k8s" {
		t.Fatalf("step 2 expert = %q, want k8s", got)
	}
	if len(out.Plan.Steps[1].DependsOn) != 1 || out.Plan.Steps[1].DependsOn[0] != "1" {
		t.Fatalf("step 2 depends_on = %#v", out.Plan.Steps[1].DependsOn)
	}
	if got := out.Plan.Steps[1].Mode; got != "readonly" {
		t.Fatalf("step 2 mode = %q, want readonly", got)
	}
	if got := looseIntValue(out.Plan.Steps[0].Input["cluster_id"]); got != 1 {
		t.Fatalf("step 1 cluster_id = %d, want 1", got)
	}
	if got := looseStringValue(out.Plan.Steps[0].Input["pod"]); got != "mysql-0" {
		t.Fatalf("step 1 pod = %q, want mysql-0", got)
	}
}

func TestNormalizeDecisionClarifiesWhenK8sPlanMissesClusterContext(t *testing.T) {
	base := buildBaseDecision(Input{
		Message: "查看 mysql-0 pod 最近 100 条日志",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看 mysql-0 pod 最近 100 条日志",
			OperationMode:  "query",
		},
	})
	parsed := Decision{
		Type: DecisionPlan,
		Plan: &ExecutionPlan{
			PlanID: "plan-2",
			Goal:   "查看 mysql-0 pod 最近 100 条日志",
			Steps: []PlanStep{{
				StepID: "step-1",
				Title:  "拉取 Pod 日志",
				Expert: "k8s",
				Task:   "读取 mysql-0 pod 日志",
				Mode:   "readonly",
				Risk:   "low",
			}},
		},
	}

	out := normalizeDecision(base, parsed)
	if out.Type != DecisionClarify {
		t.Fatalf("Type = %s, want %s", out.Type, DecisionClarify)
	}
	if out.Message == "" {
		t.Fatalf("clarify message is empty")
	}
}

func TestNormalizeDecisionPropagatesSelectedResourceIDsIntoStepInput(t *testing.T) {
	base := buildBaseDecision(Input{
		Message: "发布 payment-api 到 prod 集群",
		Rewrite: rewrite.Output{
			NormalizedGoal: "发布 payment-api 到 prod 集群",
			OperationMode:  "mutate",
			ResourceHints: rewrite.ResourceHints{
				ServiceName: "payment-api",
				ServiceID:   11,
				ClusterName: "prod",
				ClusterID:   22,
			},
		},
	})
	parsed := Decision{
		Type: DecisionPlan,
		Plan: &ExecutionPlan{
			PlanID: "plan-3",
			Goal:   "发布 payment-api 到 prod 集群",
			Steps: []PlanStep{{
				StepID: "step-1",
				Title:  "执行服务部署",
				Expert: "service",
				Task:   "deploy payment-api",
				Mode:   "mutating",
				Risk:   "high",
			}},
		},
	}

	out := normalizeDecision(base, parsed)
	if out.Type != DecisionPlan || out.Plan == nil {
		t.Fatalf("unexpected decision: %#v", out)
	}
	if got := looseIntValue(out.Plan.Steps[0].Input["service_id"]); got != 11 {
		t.Fatalf("service_id = %d, want 11", got)
	}
	if got := looseIntValue(out.Plan.Steps[0].Input["cluster_id"]); got != 22 {
		t.Fatalf("cluster_id = %d, want 22", got)
	}
}

func TestBuildBaseDecisionCarriesPodTargetIntoResolvedResources(t *testing.T) {
	out := buildBaseDecision(Input{
		Message: "查看 local 集群 kube-system 空间下的 cilium-87f2m 最近 100 条日志",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看 local 集群 kube-system 空间下的 cilium-87f2m 最近 100 条日志",
			OperationMode:  "query",
			ResourceHints: rewrite.ResourceHints{
				ClusterName: "local",
				Namespace:   "kube-system",
			},
			NormalizedRequest: rewrite.NormalizedRequest{
				Targets: []rewrite.RequestTarget{
					{Type: "pod", Name: "cilium-87f2m"},
				},
			},
		},
	})
	if out.Plan == nil {
		t.Fatalf("base plan is nil")
	}
	if got := out.Plan.Resolved.PodName; got != "cilium-87f2m" {
		t.Fatalf("pod name = %q, want cilium-87f2m", got)
	}
}

func TestNormalizeClarifyDecisionDoesNotAskForKnownPodAgain(t *testing.T) {
	base := buildBaseDecision(Input{
		Message: "查看 local 集群 kube-system 空间下的 cilium-87f2m 最近 100 条日志",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看 local 集群 kube-system 空间下的 cilium-87f2m 最近 100 条日志",
			OperationMode:  "query",
			ResourceHints: rewrite.ResourceHints{
				ClusterName: "local",
				Namespace:   "kube-system",
			},
			NormalizedRequest: rewrite.NormalizedRequest{
				Targets: []rewrite.RequestTarget{
					{Type: "pod", Name: "cilium-87f2m"},
				},
			},
		},
	})
	parsed := Decision{
		Type:      DecisionClarify,
		Message:   "需要先明确目标 Pod 名称后才能继续执行。",
		Narrative: "缺少 pod 信息。",
	}

	out := normalizeDecision(base, parsed)
	if out.Type != DecisionClarify {
		t.Fatalf("Type = %s, want %s", out.Type, DecisionClarify)
	}
	if out.Message != "需要先明确目标集群后才能继续执行。" {
		t.Fatalf("clarify message = %q", out.Message)
	}
}

func TestValidatePlanPrerequisitesUsesStructuredTargetTypeInsteadOfKeyword(t *testing.T) {
	plan := &ExecutionPlan{
		PlanID: "plan-4",
		Resolved: ResolvedResources{
			ClusterID: 3,
		},
		Steps: []PlanStep{{
			StepID: "step-1",
			Expert: "k8s",
			Title:  "读取最近 100 条日志",
			Task:   "get the latest 100 lines and analyze health",
			Input: map[string]any{
				"normalized_request": map[string]any{
					"targets": []any{
						map[string]any{"type": "pod", "name": "cilium-87f2m"},
					},
				},
			},
		}},
	}

	out := validatePlanPrerequisites(plan)
	if out.Type != DecisionClarify {
		t.Fatalf("Type = %s, want %s", out.Type, DecisionClarify)
	}
	if out.Message != "需要先明确目标 Pod 名称后才能继续执行。" {
		t.Fatalf("Message = %q", out.Message)
	}
}
