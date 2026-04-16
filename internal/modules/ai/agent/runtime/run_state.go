package runtime

// RunState 表示运行时状态。
type RunState string

const (
	RunStateCreated         RunState = "created"
	RunStateRouting         RunState = "routing"
	RunStateExecuting       RunState = "executing"
	RunStateDelegating      RunState = "delegating"
	RunStateWaitingSubagent RunState = "waiting_subagent"
	RunStateWaitingApproval RunState = "waiting_approval"
	RunStateResuming        RunState = "resuming"
	RunStateCompleted       RunState = "completed"
	RunStateFailed          RunState = "failed"
	RunStateCancelled       RunState = "cancelled"
)

// Known 返回当前状态是否为受支持状态。
func (s RunState) Known() bool {
	switch s {
	case RunStateCreated,
		RunStateRouting,
		RunStateExecuting,
		RunStateDelegating,
		RunStateWaitingSubagent,
		RunStateWaitingApproval,
		RunStateResuming,
		RunStateCompleted,
		RunStateFailed,
		RunStateCancelled:
		return true
	default:
		return false
	}
}
