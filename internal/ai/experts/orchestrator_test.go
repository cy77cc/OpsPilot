package experts

import (
	"context"
	"strings"
	"testing"
)

func TestOrchestratorExecuteSequential(t *testing.T) {
	reg := &fakeRegistry{
		experts: map[string]*Expert{
			"service_expert": {Name: "service_expert"},
			"k8s_expert":     {Name: "k8s_expert"},
		},
	}
	orch := NewOrchestrator(reg, NewResultAggregator(AggregationTemplate, nil))
	out, err := orch.Execute(context.Background(), &ExecuteRequest{
		Message: "service unavailable",
		Decision: &RouteDecision{
			PrimaryExpert: "service_expert",
			HelperExperts: []string{"k8s_expert"},
			Strategy:      StrategySequential,
			Source:        "scene",
		},
	})
	if err != nil {
		t.Fatalf("execute sequential: %v", err)
	}
	if out == nil || !strings.Contains(out.Response, "service_expert") {
		t.Fatalf("unexpected execute result: %#v", out)
	}
	if len(out.Traces) != 2 {
		t.Fatalf("expected 2 traces, got %d", len(out.Traces))
	}
}

func TestOrchestratorExecuteParallel(t *testing.T) {
	reg := &fakeRegistry{
		experts: map[string]*Expert{
			"host_expert":    {Name: "host_expert"},
			"monitor_expert": {Name: "monitor_expert"},
		},
	}
	orch := NewOrchestrator(reg, NewResultAggregator(AggregationTemplate, nil))
	out, err := orch.Execute(context.Background(), &ExecuteRequest{
		Message: "host issue",
		Decision: &RouteDecision{
			PrimaryExpert: "host_expert",
			HelperExperts: []string{"monitor_expert"},
			Strategy:      StrategyParallel,
			Source:        "keyword",
		},
		RuntimeContext: map[string]any{"timeout_ms": 5000},
	})
	if err != nil {
		t.Fatalf("execute parallel: %v", err)
	}
	if out == nil || len(out.Traces) != 2 {
		t.Fatalf("unexpected execute result: %#v", out)
	}
}
