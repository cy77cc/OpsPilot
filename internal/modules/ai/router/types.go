package router

import "fmt"

type Mode string
type TaskAction string
type RunAction string
type ExecutionShape string
type Domain string
type ToolStrategy string

const (
	ModeConversation Mode = "conversation"
	ModeOperation    Mode = "operation"

	TaskActionNone         TaskAction = "none"
	TaskActionCreateTask   TaskAction = "create_task"
	TaskActionContinueTask TaskAction = "continue_task"

	RunActionNone        RunAction = "none"
	RunActionCreateRun   RunAction = "create_run"
	RunActionContinueRun RunAction = "continue_run"

	ExecutionShapeSingleAgent         ExecutionShape = "single_agent"
	ExecutionShapeDelegatedSpecialist ExecutionShape = "delegated_specialist"

	DomainGeneral Domain = "general"

	ToolStrategyDirect      ToolStrategy = "direct"
	ToolStrategySearchFirst ToolStrategy = "search_first"
)

type RouteInput struct {
	Message string `json:"message"`
}

type ContextPlan struct {
	LoadArtifacts []string     `json:"load_artifacts,omitempty"`
	ToolStrategy  ToolStrategy `json:"tool_strategy"`
}

type RouteDecision struct {
	Mode                Mode           `json:"mode"`
	TaskAction          TaskAction     `json:"task_action"`
	RunAction           RunAction      `json:"run_action"`
	ExecutionShape      ExecutionShape `json:"execution_shape"`
	Domain              Domain         `json:"domain"`
	NeedsApprovalReview bool           `json:"needs_approval_review"`
	ContextPlan         ContextPlan    `json:"context_plan"`
	Confidence          float64        `json:"confidence"`
}

func (d RouteDecision) Validate() error {
	switch d.Mode {
	case ModeConversation, ModeOperation:
	default:
		return fmt.Errorf("invalid mode %q", d.Mode)
	}
	switch d.TaskAction {
	case TaskActionNone, TaskActionCreateTask, TaskActionContinueTask:
	default:
		return fmt.Errorf("invalid task action %q", d.TaskAction)
	}
	switch d.RunAction {
	case RunActionNone, RunActionCreateRun, RunActionContinueRun:
	default:
		return fmt.Errorf("invalid run action %q", d.RunAction)
	}
	switch d.ExecutionShape {
	case ExecutionShapeSingleAgent, ExecutionShapeDelegatedSpecialist:
	default:
		return fmt.Errorf("invalid execution shape %q", d.ExecutionShape)
	}
	switch d.Domain {
	case DomainGeneral:
	default:
		return fmt.Errorf("invalid domain %q", d.Domain)
	}
	if d.Mode == ModeConversation && d.RunAction != RunActionNone {
		return fmt.Errorf("conversation mode cannot create or continue runs")
	}
	if d.Mode == ModeConversation && d.TaskAction != TaskActionNone {
		return fmt.Errorf("conversation mode cannot create or continue tasks")
	}
	if d.Confidence < 0 || d.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	if d.ContextPlan.ToolStrategy == "" {
		return fmt.Errorf("tool strategy is required")
	}
	switch d.ContextPlan.ToolStrategy {
	case ToolStrategyDirect, ToolStrategySearchFirst:
	default:
		return fmt.Errorf("invalid tool strategy %q", d.ContextPlan.ToolStrategy)
	}
	return nil
}
