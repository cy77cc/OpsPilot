package orchestrator

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
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
	supervisor := NewSupervisor(NewRegistry())
	task := supervisor.BuildDelegationTask(
		"run-1",
		"monitor",
		"Summarize the p95 spike.",
		contracts.Scope{Service: "api-gateway"},
	)
	if err := task.Validate(); err != nil {
		t.Fatalf("expected valid task, got %v", err)
	}
}
