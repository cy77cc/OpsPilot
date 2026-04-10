package runtime

import "strings"

// RunState models canonical runtime run states for service-level orchestration.
type RunState string

const (
	RunStateCreated         RunState = "created"
	RunStateRouting         RunState = "routing"
	RunStatePlanning        RunState = "planning"
	RunStateExecuting       RunState = "executing"
	RunStateWaitingApproval RunState = "waiting_approval"
	RunStateResuming        RunState = "resuming"
	RunStateCompleted       RunState = "completed"
	RunStateFailed          RunState = "failed"
	RunStateCancelled       RunState = "cancelled"
)

func normalizeRunState(state RunState) RunState {
	return RunState(strings.ToLower(strings.TrimSpace(string(state))))
}
