package orchestrator

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
)

func TestRegistry_DefaultMonitorSpecialist(t *testing.T) {
	registry := NewRegistry()
	registry.Register("monitoring", SpecialistSpec{Name: "monitor"})

	spec, ok := registry.Lookup("monitoring")
	if !ok || spec.Name != "monitor" {
		t.Fatalf("unexpected specialist lookup: %#v %v", spec, ok)
	}
}

func TestSupervisor_BuildDelegationTask(t *testing.T) {
	registry := NewRegistry()
	registry.Register("monitoring", SpecialistSpec{Name: "monitor", Domain: "monitoring", ReadOnly: true})
	registry.Register("kubernetes", SpecialistSpec{Name: "kubernetes", Domain: "kubernetes", ReadOnly: true})
	supervisor := NewSupervisor(registry)

	monitorTask := supervisor.BuildDelegationTask(
		"run-1",
		"monitor",
		"Summarize the p95 spike.",
		contracts.Scope{Service: "api-gateway"},
	)
	if err := monitorTask.Validate(); err != nil {
		t.Fatalf("expected valid task, got %v", err)
	}
	if monitorTask.Intent != contracts.IntentMetricAnomalySummary || monitorTask.ExpectedOutput != contracts.ExpectedMetricAnomalySummary {
		t.Fatalf("unexpected monitor semantics: intent=%q expected_output=%q", monitorTask.Intent, monitorTask.ExpectedOutput)
	}

	kubernetesTask := supervisor.BuildDelegationTask(
		"run-1",
		"kubernetes",
		"Summarize the resource inventory.",
		contracts.Scope{Cluster: "prod"},
	)
	if err := kubernetesTask.Validate(); err != nil {
		t.Fatalf("expected valid task, got %v", err)
	}
	if kubernetesTask.Intent != contracts.IntentResourceInventory || kubernetesTask.ExpectedOutput != contracts.ExpectedResourceInventory {
		t.Fatalf("unexpected kubernetes semantics: intent=%q expected_output=%q", kubernetesTask.Intent, kubernetesTask.ExpectedOutput)
	}
}

func TestSupervisor_BuildDispatchDecision_CaseVariantDelegates(t *testing.T) {
	registry := NewRegistry()
	registry.Register("monitoring", SpecialistSpec{Name: "monitor"})
	supervisor := NewSupervisor(registry)

	decision := supervisor.BuildDispatchDecision("Monitoring")
	if decision.ExecutionShape != airuntime.ExecutionShapeDelegatedSpecialist {
		t.Fatalf("expected delegated_specialist shape, got %q", decision.ExecutionShape)
	}
}

func TestSupervisor_BuildDispatchDecision_NonMonitorSceneStaysSingle(t *testing.T) {
	registry := NewRegistry()
	registry.Register("host", SpecialistSpec{Name: "host"})
	supervisor := NewSupervisor(registry)

	decision := supervisor.BuildDispatchDecision("HOST")
	if decision.ExecutionShape != airuntime.ExecutionShapeSingleAgent {
		t.Fatalf("expected single_agent shape, got %q", decision.ExecutionShape)
	}
}
