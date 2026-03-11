package planner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
)

func TestPlanReturnsUnavailableWhenRunnerMissing(t *testing.T) {
	_, err := New(nil).Plan(context.Background(), Input{
		Message: "查看所有主机的状态",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看所有主机的状态",
			OperationMode:  "query",
		},
	})
	if err == nil {
		t.Fatalf("Plan() error = nil, want PlanningError")
	}
	var planningErr *PlanningError
	if !errors.As(err, &planningErr) {
		t.Fatalf("Plan() error = %v, want PlanningError", err)
	}
	if planningErr.Code != "planner_runner_unavailable" {
		t.Fatalf("Code = %q, want planner_runner_unavailable", planningErr.Code)
	}
}

func TestPlanDoesNotShortCircuitWhenRewriteReportsAmbiguity(t *testing.T) {
	called := false
	planner := NewWithFunc(func(_ context.Context, in Input, _ func(string)) (Decision, error) {
		called = true
		if len(in.Rewrite.AmbiguityFlags) != 1 {
			t.Fatalf("ambiguity flags = %#v, want preserved input", in.Rewrite.AmbiguityFlags)
		}
		return Decision{Type: DecisionDirectReply, Message: "继续规划"}, nil
	})

	out, err := planner.Plan(context.Background(), Input{
		Message: "帮我看看状态",
		Rewrite: rewrite.Output{
			AmbiguityFlags: []string{"resource_target_not_explicit"},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if !called {
		t.Fatalf("planner run function was not called")
	}
	if out.Type != DecisionDirectReply {
		t.Fatalf("Type = %s, want %s", out.Type, DecisionDirectReply)
	}
}

func TestNormalizeDecisionDoesNotPanicWhenBaseHasNoPlan(t *testing.T) {
	base := &ExecutionPlan{}
	parsed := Decision{
		Type: DecisionPlan,
		Plan: &ExecutionPlan{
			PlanID: "plan-0",
			Goal:   "check mysql-0",
			Steps: []PlanStep{{
				StepID: "step-1",
				Title:  "检查 Pod 状态",
				Expert: "k8s",
				Task:   "check mysql-0",
				Mode:   "readonly",
				Risk:   "low",
				Input: map[string]any{
					"cluster_id": 1,
					"pod":        "mysql-0",
				},
			}},
		},
	}

	out, err := normalizeDecision(base, parsed)
	if err != nil {
		t.Fatalf("normalizeDecision() error = %v", err)
	}
	if out.Plan == nil {
		t.Fatalf("Plan is nil")
	}
	if out.Plan.Goal != "check mysql-0" {
		t.Fatalf("Goal = %q", out.Plan.Goal)
	}
}

func TestParseDecisionAcceptsNumericStepIDsAndNormalizesPlan(t *testing.T) {
	raw := `{"narrative":"Target pod identified","plan":{"plan_id":"plan_pod_log_analysis_001","goal":"Fetch logs and analyze health","resolved":{"cluster_id":1,"pod_name":"mysql-0","namespace":"default"},"narrative":"Execute log retrieval and health analysis.","steps":[{"step_id":1,"title":"Retrieve Pod Logs","expert":"k8s","intent":"Fetch logs","task":"Retrieve logs","depends_on":[],"mode":"query","risk":"low","narrative":"Get logs"},{"step_id":2,"title":"Analyze Pod Health","expert":"observability","intent":"Assess health","task":"Analyze logs","depends_on":[1],"mode":"analysis","risk":"low","narrative":"Interpret logs"}]},"type":"plan"}`

	parsed, err := ParseDecision(raw)
	if err != nil {
		t.Fatalf("ParseDecision() error = %v", err)
	}
	base := buildBasePlanContext(Input{
		Message: "查看 mysql-0 pod 最近 100 条日志并分析运行状况",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看 mysql-0 pod 最近 100 条日志并分析运行状况",
			OperationMode:  "query",
		},
	})
	out, err := normalizeDecision(base, parsed)
	if err != nil {
		t.Fatalf("normalizeDecision() error = %v", err)
	}
	if out.Plan == nil {
		t.Fatalf("normalized plan is nil")
	}
	if got := primaryID(out.Plan.Resolved.Clusters); got != 1 {
		t.Fatalf("cluster id = %d, want %d", got, 1)
	}
	if got := out.Plan.Steps[0].StepID; got != "1" {
		t.Fatalf("step 1 id = %q, want %q", got, "1")
	}
	if got := out.Plan.Steps[0].Mode; got != "readonly" {
		t.Fatalf("step 1 mode = %q, want readonly", got)
	}
	if got := out.Plan.Steps[1].Expert; got != "observability" {
		t.Fatalf("step 2 expert = %q, want observability preserved", got)
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

func TestNormalizeDecisionReturnsPlanningInvalidWhenK8sPlanMissesClusterContext(t *testing.T) {
	base := buildBasePlanContext(Input{
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

	_, err := normalizeDecision(base, parsed)
	if err == nil {
		t.Fatalf("normalizeDecision() error = nil, want PlanningError")
	}
	var planningErr *PlanningError
	if !errors.As(err, &planningErr) {
		t.Fatalf("normalizeDecision() error = %v, want PlanningError", err)
	}
	if planningErr.Code != "planning_invalid" {
		t.Fatalf("Code = %q, want planning_invalid", planningErr.Code)
	}
}

func TestNormalizeDecisionPropagatesSelectedResourceIDsIntoStepInput(t *testing.T) {
	base := buildBasePlanContext(Input{
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

	out, err := normalizeDecision(base, parsed)
	if err != nil {
		t.Fatalf("normalizeDecision() error = %v", err)
	}
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
	base := buildBasePlanContext(Input{
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

	out, err := normalizeDecision(base, parsed)
	if err != nil {
		t.Fatalf("normalizeDecision() error = %v", err)
	}
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

func TestNormalizeDecisionPreservesFleetHostStepSemantics(t *testing.T) {
	base := buildBasePlanContext(Input{
		Message: "查看所有主机状态",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看所有主机状态",
			OperationMode:  "query",
			NormalizedRequest: rewrite.NormalizedRequest{
				Targets: []rewrite.RequestTarget{
					{Type: "host", Name: "all"},
				},
			},
		},
	})
	parsed := Decision{
		Type: DecisionPlan,
		Plan: &ExecutionPlan{
			PlanID: "plan-host-fleet",
			Goal:   "查看所有主机状态",
			Resolved: ResolvedResources{
				Scope: &ResourceScope{Kind: "all", ResourceType: "host"},
			},
			Steps: []PlanStep{{
				StepID: "step-1",
				Title:  "查询所有主机状态",
				Expert: "hostops",
				Intent: "inventory_scan",
				Task:   "query all hosts",
				Mode:   "readonly",
				Risk:   "low",
				Input: map[string]any{
					"scope": map[string]any{"kind": "all", "resource_type": "host"},
				},
			}},
		},
	}

	out, err := normalizeDecision(base, parsed)
	if err != nil {
		t.Fatalf("normalizeDecision() error = %v", err)
	}
	if out.Type != DecisionPlan || out.Plan == nil {
		t.Fatalf("unexpected decision: %#v", out)
	}
	step := out.Plan.Steps[0]
	if got := step.Intent; got != "inventory_scan" {
		t.Fatalf("step intent = %q, want inventory_scan preserved", got)
	}
	if strings.TrimSpace(step.Task) != "query all hosts" {
		t.Fatalf("step task = %q, want model task preserved", step.Task)
	}
}

func TestBuildBaseDecisionCarriesPodTargetIntoResolvedResources(t *testing.T) {
	out := buildBasePlanContext(Input{
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
	if out == nil {
		t.Fatalf("base plan is nil")
	}
	if got := out.Resolved.PodName; got != "cilium-87f2m" {
		t.Fatalf("pod name = %q, want cilium-87f2m", got)
	}
	if len(out.Resolved.Pods) != 1 || out.Resolved.Pods[0].Name != "cilium-87f2m" {
		t.Fatalf("pods = %#v, want cilium-87f2m", out.Resolved.Pods)
	}
}

func TestValidatePlanPrerequisitesUsesStructuredTargetTypeInsteadOfKeyword(t *testing.T) {
	plan := &ExecutionPlan{
		PlanID: "plan-4",
		Goal:   "读取最近 100 条日志",
		Resolved: ResolvedResources{
			ClusterID: 3,
		},
		Steps: []PlanStep{{
			StepID: "step-1",
			Expert: "k8s",
			Title:  "读取最近 100 条日志",
			Task:   "get the latest 100 lines and analyze health",
			Mode:   "readonly",
			Risk:   "low",
			Input: map[string]any{
				"normalized_request": map[string]any{
					"targets": []any{
						map[string]any{"type": "pod", "name": "cilium-87f2m"},
					},
				},
			},
		}},
	}

	if err := validatePlanPrerequisites(plan); err != nil {
		t.Fatalf("validatePlanPrerequisites() error = %v, want nil", err)
	}
}

func TestParseDecisionAcceptsResolvedScopeForFleetTargets(t *testing.T) {
	raw := `{"type":"plan","narrative":"check all hosts","plan":{"plan_id":"plan-hosts","goal":"查看所有主机状态","resolved":{"scope":{"kind":"all","resource_type":"host"},"hosts":[]},"steps":[{"step_id":"step-1","title":"查询所有主机状态","expert":"hostops","task":"query all hosts","mode":"readonly","risk":"low","input":{"scope":{"kind":"all","resource_type":"host"}}}]}}`

	parsed, err := ParseDecision(raw)
	if err != nil {
		t.Fatalf("ParseDecision() error = %v", err)
	}
	out, err := normalizeDecision(buildBasePlanContext(Input{
		Message: "查看所有主机状态",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看所有主机状态",
			OperationMode:  "query",
			NormalizedRequest: rewrite.NormalizedRequest{
				Targets: []rewrite.RequestTarget{{Type: "host", Name: "all"}},
			},
		},
	}), parsed)
	if err != nil {
		t.Fatalf("normalizeDecision() error = %v", err)
	}
	if out.Type != DecisionPlan || out.Plan == nil {
		t.Fatalf("unexpected decision: %#v", out)
	}
	if out.Plan.Resolved.Scope == nil || out.Plan.Resolved.Scope.ResourceType != "host" {
		t.Fatalf("scope = %#v, want host scope", out.Plan.Resolved.Scope)
	}
	if err := validatePlanPrerequisites(out.Plan); err != nil {
		t.Fatalf("unexpected planning validation error: %v", err)
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

func TestNormalizeDecisionReturnsPlanningInvalidForUnsupportedMode(t *testing.T) {
	base := buildBasePlanContext(Input{
		Message: "查看 payment-api 状态",
		Rewrite: rewrite.Output{
			NormalizedGoal: "查看 payment-api 状态",
		},
	})
	parsed := Decision{
		Type: DecisionPlan,
		Plan: &ExecutionPlan{
			PlanID: "plan-invalid-mode",
			Goal:   "查看 payment-api 状态",
			Steps: []PlanStep{{
				StepID: "step-1",
				Title:  "检查服务状态",
				Expert: "service",
				Task:   "inspect payment-api",
				Mode:   "custom_mode",
				Risk:   "low",
			}},
		},
	}

	_, err := normalizeDecision(base, parsed)
	if err == nil {
		t.Fatalf("normalizeDecision() error = nil, want PlanningError")
	}
	var planningErr *PlanningError
	if !errors.As(err, &planningErr) {
		t.Fatalf("normalizeDecision() error = %v, want PlanningError", err)
	}
	if planningErr.Code != "planning_invalid" {
		t.Fatalf("Code = %q, want planning_invalid", planningErr.Code)
	}
}
