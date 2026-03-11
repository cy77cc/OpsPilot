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
	if out.Plan.Resolved.Scope == nil {
		t.Fatalf("Resolved.Scope is nil")
	}
	if out.Plan.Resolved.Scope.Kind != "all" || out.Plan.Resolved.Scope.ResourceType != "host" {
		t.Fatalf("Resolved.Scope = %#v, want all host scope", out.Plan.Resolved.Scope)
	}
	if got := looseStringValue(out.Plan.Steps[0].Input["scope"].(map[string]any)["resource_type"]); got != "host" {
		t.Fatalf("step scope resource_type = %q, want host", got)
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

func TestNormalizeDecisionKeepsValidK8sPlanFromModelOutput(t *testing.T) {
	raw := `{"narrative":"用户请求查看 local 集群 kube-system 命名空间下 cilium-87f2m Pod 的最近 100 条日志并分析运行状况。已解析到集群 ID 为 1。计划分为两步：首先获取 Pod 日志，然后基于日志内容分析运行健康状态。","plan":{"plan_id":"plan-logs-cilium-87f2m","goal":"Retrieve last 100 log lines from pod cilium-87f2m in namespace kube-system on cluster local and evaluate its operational health.","resolved":{"cluster_id":1,"namespace":"kube-system","pod_name":"cilium-87f2m"},"narrative":"Target cluster 'local' resolved to ID 1. Pod name and namespace are explicitly provided. Execution will fetch logs and perform health analysis.","steps":[{"step_id":"step-1","title":"Fetch Pod Logs","expert":"k8s","intent":"Retrieve the last 100 lines of logs from the specified pod.","task":"Fetch logs for pod cilium-87f2m in namespace kube-system on cluster 1, limiting to last 100 entries.","depends_on":[],"mode":"readonly","risk":"low","narrative":"Use k8s interface to pull recent logs from the target pod."},{"step_id":"step-2","title":"Analyze Pod Health","expert":"observability","intent":"Evaluate pod running status based on retrieved logs.","task":"Analyze the fetched logs for error patterns, restart indicators, or readiness issues to determine health status.","depends_on":["step-1"],"mode":"readonly","risk":"low","narrative":"Inspect log content for anomalies to assess operational health."}]},"type":"plan"}`

	parsed, err := ParseDecision(raw)
	if err != nil {
		t.Fatalf("ParseDecision() error = %v", err)
	}
	base := buildBaseDecision(Input{
		Message: "查看 local 集群 kube-system 空间下的 cilium-87f2m 最近 100 条日志，分析运行状况",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看 local 集群 kube-system 空间下的 cilium-87f2m 最近 100 条日志，分析运行状况",
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

	out := normalizeDecision(base, parsed)
	if out.Type != DecisionPlan {
		t.Fatalf("Type = %s, want %s", out.Type, DecisionPlan)
	}
	if out.Plan == nil {
		t.Fatalf("normalized plan is nil")
	}
	if got := looseIntValue(out.Plan.Steps[0].Input["cluster_id"]); got != 1 {
		t.Fatalf("step 1 cluster_id = %d, want 1", got)
	}
	if got := looseStringValue(out.Plan.Steps[0].Input["namespace"]); got != "kube-system" {
		t.Fatalf("step 1 namespace = %q, want kube-system", got)
	}
	if got := looseStringValue(out.Plan.Steps[0].Input["pod"]); got != "cilium-87f2m" {
		t.Fatalf("step 1 pod = %q, want cilium-87f2m", got)
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
	if len(out.Plan.Resolved.Pods) != 1 || out.Plan.Resolved.Pods[0].Name != "cilium-87f2m" {
		t.Fatalf("pods = %#v, want cilium-87f2m", out.Plan.Resolved.Pods)
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
	if out.Type != "" {
		t.Fatalf("Type = %s, want empty decision", out.Type)
	}
}

func TestParseDecisionAcceptsResolvedScopeForFleetTargets(t *testing.T) {
	raw := `{"type":"plan","narrative":"check all hosts","plan":{"plan_id":"plan-hosts","goal":"查看所有主机状态","resolved":{"scope":{"kind":"all","resource_type":"host"},"hosts":[]},"steps":[{"step_id":"step-1","title":"查询所有主机状态","expert":"hostops","task":"query all hosts","mode":"readonly","risk":"low","input":{"scope":{"kind":"all","resource_type":"host"}}}]}}`

	parsed, err := ParseDecision(raw)
	if err != nil {
		t.Fatalf("ParseDecision() error = %v", err)
	}
	out := normalizeDecision(buildBaseDecision(Input{
		Message: "查看所有主机状态",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看所有主机状态",
			OperationMode:  "query",
			NormalizedRequest: rewrite.NormalizedRequest{
				Targets: []rewrite.RequestTarget{{Type: "host", Name: "all"}},
			},
		},
	}), parsed)
	if out.Type != DecisionPlan || out.Plan == nil {
		t.Fatalf("unexpected decision: %#v", out)
	}
	if out.Plan.Resolved.Scope == nil || out.Plan.Resolved.Scope.ResourceType != "host" {
		t.Fatalf("scope = %#v, want host scope", out.Plan.Resolved.Scope)
	}
	if clarify := validatePlanPrerequisites(out.Plan); clarify.Type != "" {
		t.Fatalf("unexpected clarify: %#v", clarify)
	}
}

func TestPopulateStepInputPropagatesMultipleHostsAsHostIDs(t *testing.T) {
	step := PlanStep{Expert: "hostops", Input: map[string]any{}}
	out := populateStepInput(step, ResolvedResources{
		HostIDs: []int{1, 2, 3},
		Hosts:   []ResourceRef{{ID: 1, Name: "n1"}, {ID: 2, Name: "n2"}, {ID: 3, Name: "n3"}},
	})
	got := intSliceValue(out["host_ids"])
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("host_ids = %#v, want [1 2 3]", got)
	}
	if _, ok := out["host_id"]; ok {
		t.Fatalf("host_id should not be set for multi-host input: %#v", out)
	}
}
