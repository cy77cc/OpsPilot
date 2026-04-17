package chathandler

import (
	"context"
	"fmt"
)

type RouteMode string

const (
	ModeConversation RouteMode = "conversation"
	ModeTask         RouteMode = "task"
)

type TaskAction string

const (
	TaskActionCreateTask TaskAction = "create_task"
	TaskActionNone       TaskAction = ""
)

type RunAction string

const (
	RunActionCreateRun RunAction = "create_run"
	RunActionNone      RunAction = ""
)

type ExecutionShape string

const (
	ExecutionShapeSingleAgent         ExecutionShape = "single_agent"
	ExecutionShapeDelegatedSpecialist ExecutionShape = "delegated_specialist"
)

type Domain string

const (
	DomainGeneral Domain = "general"
)

type ToolStrategy string

const (
	ToolStrategyDirect ToolStrategy = "direct"
)

type ContextPlan struct {
	ToolStrategy ToolStrategy `json:"tool_strategy"`
}

type RouteDecision struct {
	Mode           RouteMode      `json:"mode"`
	TaskAction     TaskAction     `json:"task_action"`
	RunAction      RunAction      `json:"run_action"`
	ExecutionShape ExecutionShape `json:"execution_shape"`
	Domain         Domain         `json:"domain"`
	ContextPlan    ContextPlan    `json:"context_plan"`
}

func (d RouteDecision) Validate() error {
	if d.Mode == ModeConversation {
		if d.TaskAction != TaskActionNone || d.RunAction != RunActionNone {
			return fmt.Errorf("conversation route cannot schedule task or run actions")
		}
	}
	return nil
}

type RouteInput struct {
	Message string `json:"message"`
}

type RouteService interface {
	Decide(ctx context.Context, input RouteInput) (RouteDecision, error)
}

type routeService struct{}

func NewRouteService() RouteService {
	return &routeService{}
}

func (r *routeService) Decide(_ context.Context, _ RouteInput) (RouteDecision, error) {
	return RouteDecision{
		Mode:           ModeConversation,
		TaskAction:     TaskActionNone,
		RunAction:      RunActionNone,
		ExecutionShape: ExecutionShapeSingleAgent,
		Domain:         DomainGeneral,
		ContextPlan: ContextPlan{
			ToolStrategy: ToolStrategyDirect,
		},
	}, nil
}
