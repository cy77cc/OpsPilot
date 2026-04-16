package orchestrator

import (
	"strings"

	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
)

// BuildDispatchDecision computes whether a scene should delegate to a specialist.
func (s *Supervisor) BuildDispatchDecision(scene string) airuntime.DispatchDecision {
	normalized := strings.TrimSpace(scene)
	specialistAvailable := false
	if s != nil && s.registry != nil {
		_, specialistAvailable = s.registry.Lookup(normalized)
	}
	return airuntime.NewKernel().BuildDispatchDecision(normalized, specialistAvailable)
}

// ShouldDelegate reports whether the scene should be handled through delegation.
func (s *Supervisor) ShouldDelegate(scene string) bool {
	return s.BuildDispatchDecision(scene).ExecutionShape == airuntime.ExecutionShapeDelegatedSpecialist
}
