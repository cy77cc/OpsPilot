package orchestrator

import "testing"

func TestNewRegistry_RegistersCoreSpecialists(t *testing.T) {
	registry := NewDefaultRegistry()

	tests := []struct {
		scene string
		want  SpecialistSpec
	}{
		{
			scene: "monitoring",
			want:  SpecialistSpec{Name: "monitor", Domain: "monitoring", ReadOnly: true},
		},
		{
			scene: "kubernetes",
			want:  SpecialistSpec{Name: "kubernetes", Domain: "kubernetes", ReadOnly: true},
		},
		{
			scene: "host",
			want:  SpecialistSpec{Name: "host", Domain: "host", ReadOnly: true},
		},
		{
			scene: "cicd",
			want:  SpecialistSpec{Name: "cicd", Domain: "cicd", ReadOnly: true},
		},
	}

	for _, tt := range tests {
		spec, ok := registry.Lookup(tt.scene)
		if !ok {
			t.Fatalf("expected specialist for scene %s", tt.scene)
		}
		if spec.Name != tt.want.Name || spec.Domain != tt.want.Domain || spec.ReadOnly != tt.want.ReadOnly {
			t.Fatalf("scene %s: got spec %+v, want %+v", tt.scene, spec, tt.want)
		}
	}
}
