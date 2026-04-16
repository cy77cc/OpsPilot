package logic

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/orchestrator"
)

func TestLookupDelegatedSpecialist_MonitoringOnly(t *testing.T) {
	t.Parallel()

	registry := orchestrator.NewDefaultRegistry()

	monitor, ok := lookupDelegatedSpecialist(registry, "monitoring")
	if !ok {
		t.Fatal("expected monitoring to use delegated specialist")
	}
	if monitor.Name != "monitor" {
		t.Fatalf("expected monitor specialist, got %+v", monitor)
	}

	for _, scene := range []string{"host", "kubernetes", "cicd", "unknown"} {
		if spec, delegated := lookupDelegatedSpecialist(registry, scene); delegated {
			t.Fatalf("did not expect %s to delegate, got %+v", scene, spec)
		}
	}
}

func TestSceneRequiresReadOnlyExecution_RegisteredReadOnlyScenes(t *testing.T) {
	t.Parallel()

	for _, scene := range []string{"monitoring", "host", "cicd", "kubernetes"} {
		if !sceneRequiresReadOnlyExecution(scene) {
			t.Fatalf("expected %s to require read-only execution", scene)
		}
	}

	for _, scene := range []string{"service", "deployment", "governance", "unknown"} {
		if sceneRequiresReadOnlyExecution(scene) {
			t.Fatalf("did not expect %s to require read-only execution", scene)
		}
	}
}
