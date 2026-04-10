package runtime

import (
	"fmt"
	"strings"
)

// Kernel is the minimal service-level runtime boundary used by Chat/Approval
// orchestration while the full runtime graph is still being introduced.
type Kernel struct{}

func NewKernel() *Kernel {
	return &Kernel{}
}

func (k *Kernel) ShouldCreateRun(mode string) bool {
	return strings.EqualFold(strings.TrimSpace(mode), "operation")
}

func (k *Kernel) ResumeTransition(state RunState) (RunState, error) {
	switch normalizeRunState(state) {
	case RunStateWaitingApproval:
		return RunStateResuming, nil
	default:
		return "", fmt.Errorf("cannot resume from state %q", strings.TrimSpace(string(state)))
	}
}
