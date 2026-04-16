package orchestrator

import "testing"

func TestNewRegistry_RegistersCoreSpecialists(t *testing.T) {
	registry := NewDefaultRegistry()

	for _, scene := range []string{"monitoring", "kubernetes", "host", "cicd"} {
		spec, ok := registry.Lookup(scene)
		if !ok {
			t.Fatalf("expected specialist for scene %s", scene)
		}
		if !spec.ReadOnly {
			t.Fatalf("expected read-only specialist for scene %s", scene)
		}
	}
}
