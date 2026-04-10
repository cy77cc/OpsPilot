package runtime

import (
	"fmt"
	"strings"
)

const (
	ExecutionShapeSingleAgent         = "single_agent"
	ExecutionShapeDelegatedSpecialist = "delegated_specialist"
)

type DispatchDecision struct {
	ExecutionShape      string
	Domain              string
	SpecialistAvailable bool
}

// Kernel is the minimal service-level runtime boundary used by Chat/Approval
// orchestration while the full runtime graph is still being introduced.
type Kernel struct{}

func NewKernel() *Kernel {
	return &Kernel{}
}

func (k *Kernel) ShouldCreateRun(mode string) bool {
	return strings.EqualFold(strings.TrimSpace(mode), "operation")
}

func (k *Kernel) DefaultExecutionShape() string {
	return ExecutionShapeSingleAgent
}

func (k *Kernel) BuildDispatchDecision(domain string, specialistAvailable bool) DispatchDecision {
	domain = strings.TrimSpace(strings.ToLower(domain))
	decision := DispatchDecision{
		ExecutionShape:      k.DefaultExecutionShape(),
		Domain:              domain,
		SpecialistAvailable: specialistAvailable,
	}
	if domain != "" && specialistAvailable {
		decision.ExecutionShape = ExecutionShapeDelegatedSpecialist
	}
	return decision
}

func (k *Kernel) ResumeTransition(state RunState) (RunState, error) {
	switch normalizeRunState(state) {
	case RunStateWaitingApproval:
		return RunStateResuming, nil
	default:
		return "", fmt.Errorf("cannot resume from state %q", strings.TrimSpace(string(state)))
	}
}
