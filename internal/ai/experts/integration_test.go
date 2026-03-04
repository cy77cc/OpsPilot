package experts

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func buildTestTools(t *testing.T) map[string]tool.InvokableTool {
	t.Helper()
	return map[string]tool.InvokableTool{
		"host_list_inventory":      mustTool(t, "host_list_inventory"),
		"host_ssh_exec_readonly":   mustTool(t, "host_ssh_exec_readonly"),
		"k8s_list_resources":       mustTool(t, "k8s_list_resources"),
		"k8s_get_events":           mustTool(t, "k8s_get_events"),
		"service_get_detail":       mustTool(t, "service_get_detail"),
		"service_deploy_preview":   mustTool(t, "service_deploy_preview"),
		"monitor_metric_query":     mustTool(t, "monitor_metric_query"),
		"audit_log_search":         mustTool(t, "audit_log_search"),
		"permission_check":         mustTool(t, "permission_check"),
		"deployment_target_detail": mustTool(t, "deployment_target_detail"),
	}
}

func TestHybridMOEPipelineE2E(t *testing.T) {
	reg, err := NewExpertRegistry(context.Background(), "configs/experts.yaml", buildTestTools(t), nil)
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	router, err := NewHybridRouter(reg, "configs/scene_mappings.yaml")
	if err != nil {
		t.Fatalf("new router: %v", err)
	}
	orch := NewOrchestrator(reg, NewResultAggregator(AggregationTemplate, nil))

	decision := router.Route(context.Background(), &RouteRequest{
		Message: "请排查服务发布失败",
		Scene:   "scene:services:deploy",
	})
	if decision == nil || decision.Source != "scene" || decision.PrimaryExpert == "" {
		t.Fatalf("unexpected route decision: %#v", decision)
	}

	result, err := orch.Execute(context.Background(), &ExecuteRequest{
		Message:  "请排查服务发布失败",
		Decision: decision,
		RuntimeContext: map[string]any{
			"timeout_ms": 3000,
		},
	})
	if err != nil {
		t.Fatalf("orchestrator execute: %v", err)
	}
	if result == nil || !strings.Contains(result.Response, decision.PrimaryExpert) {
		t.Fatalf("unexpected orchestration result: %#v", result)
	}
}

func TestHybridMOEPipelineErrorCase(t *testing.T) {
	reg, err := NewExpertRegistry(context.Background(), "configs/experts.yaml", buildTestTools(t), nil)
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	orch := NewOrchestrator(reg, NewResultAggregator(AggregationTemplate, nil))
	_, err = orch.Execute(context.Background(), &ExecuteRequest{
		Message: "test",
		Decision: &RouteDecision{
			PrimaryExpert: "non_exist_expert",
			Strategy:      StrategySingle,
			Source:        "scene",
		},
	})
	if err == nil {
		t.Fatalf("expected error for unknown expert")
	}
}
