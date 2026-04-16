package logic

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/orchestrator"
)

func TestDefaultRegistryEntries_CoversDeepSpecialists(t *testing.T) {
	t.Parallel()

	registry := orchestrator.NewDefaultRegistry()
	entries := registry.Entries()
	if len(entries) != 4 {
		t.Fatalf("expected 4 deep specialist entries, got %d", len(entries))
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
