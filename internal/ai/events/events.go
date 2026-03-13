package events

// Name is the canonical SSE/runtime event name.
type Name string

const (
	Meta             Name = "meta"
	Delta            Name = "delta"
	ThinkingDelta    Name = "thinking_delta"
	ToolCall         Name = "tool_call"
	ToolResult       Name = "tool_result"
	ApprovalRequired Name = "approval_required"
	Done             Name = "done"
	Error            Name = "error"

	RewriteResult   Name = "rewrite_result"
	PlannerState    Name = "planner_state"
	PlanCreated     Name = "plan_created"
	StageDelta      Name = "stage_delta"
	StepUpdate      Name = "step_update"
	ClarifyRequired Name = "clarify_required"
	ReplanStarted   Name = "replan_started"

	TurnStarted Name = "turn_started"
	TurnState   Name = "turn_state"
)
