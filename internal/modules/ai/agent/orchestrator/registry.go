package orchestrator

import "strings"

// SpecialistSpec describes a registered specialist target for a scene.
type SpecialistSpec struct {
	Name     string
	Domain   string
	ReadOnly bool
}

// Registry keeps specialist routing metadata by scene.
type Registry struct {
	byScene map[string]SpecialistSpec
}

type RegistryEntry struct {
	Scene string
	Spec  SpecialistSpec
}

func NewRegistry() *Registry {
	return &Registry{byScene: map[string]SpecialistSpec{}}
}

func NewDefaultRegistry() *Registry {
	registry := NewRegistry()
	registry.Register("monitoring", SpecialistSpec{Name: "monitor", Domain: "monitoring", ReadOnly: true})
	registry.Register("kubernetes", SpecialistSpec{Name: "kubernetes", Domain: "kubernetes", ReadOnly: true})
	registry.Register("host", SpecialistSpec{Name: "host", Domain: "host", ReadOnly: true})
	registry.Register("cicd", SpecialistSpec{Name: "cicd", Domain: "cicd", ReadOnly: true})
	return registry
}

func (r *Registry) Register(scene string, spec SpecialistSpec) {
	if r == nil {
		return
	}
	if r.byScene == nil {
		r.byScene = map[string]SpecialistSpec{}
	}
	r.byScene[normalizeScene(scene)] = spec
}

func (r *Registry) Lookup(scene string) (SpecialistSpec, bool) {
	if r == nil || r.byScene == nil {
		return SpecialistSpec{}, false
	}
	spec, ok := r.byScene[normalizeScene(scene)]
	return spec, ok
}

func (r *Registry) Entries() []RegistryEntry {
	if r == nil || r.byScene == nil {
		return nil
	}
	entries := make([]RegistryEntry, 0, len(r.byScene))
	for scene, spec := range r.byScene {
		entries = append(entries, RegistryEntry{Scene: scene, Spec: spec})
	}
	return entries
}

func normalizeScene(scene string) string {
	return strings.ToLower(strings.TrimSpace(scene))
}
