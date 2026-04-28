package orchestrator

import (
	cicdspecialist "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/specialists/cicd"
	hostspecialist "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/specialists/host"
	k8sspecialist "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/specialists/kubernetes"
	monitorspecialist "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/specialists/monitor"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/sceneutil"
)

// SpecialistSpec describes a registered specialist target for a scene.
type SpecialistSpec struct {
	Name        string
	Domain      string
	Description string
	Instruction string
	ReadOnly    bool
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
	{
		name, domain, description, instruction, readOnly := monitorspecialist.Spec()
		registerFromSpecialist(registry, monitorspecialist.Scene(), name, domain, description, instruction, readOnly)
	}
	{
		name, domain, description, instruction, readOnly := k8sspecialist.Spec()
		registerFromSpecialist(registry, k8sspecialist.Scene(), name, domain, description, instruction, readOnly)
	}
	{
		name, domain, description, instruction, readOnly := hostspecialist.Spec()
		registerFromSpecialist(registry, hostspecialist.Scene(), name, domain, description, instruction, readOnly)
	}
	{
		name, domain, description, instruction, readOnly := cicdspecialist.Spec()
		registerFromSpecialist(registry, cicdspecialist.Scene(), name, domain, description, instruction, readOnly)
	}
	return registry
}

func registerFromSpecialist(
	registry *Registry,
	scene string,
	name string,
	domain string,
	description string,
	instruction string,
	readOnly bool,
) {
	registry.Register(scene, SpecialistSpec{
		Name:        name,
		Domain:      domain,
		Description: description,
		Instruction: instruction,
		ReadOnly:    readOnly,
	})
}

func (r *Registry) Register(scene string, spec SpecialistSpec) {
	if r == nil {
		return
	}
	if r.byScene == nil {
		r.byScene = map[string]SpecialistSpec{}
	}
	r.byScene[sceneutil.NormalizeScene(scene)] = spec
}

func (r *Registry) Lookup(scene string) (SpecialistSpec, bool) {
	if r == nil || r.byScene == nil {
		return SpecialistSpec{}, false
	}
	spec, ok := r.byScene[sceneutil.NormalizeScene(scene)]
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
