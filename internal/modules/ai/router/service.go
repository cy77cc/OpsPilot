package router

import "context"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Decide(_ context.Context, _ RouteInput) (RouteDecision, error) {
	// Bootstrap fallback: until semantic routing is wired to the model,
	// default to the existing operational path rather than silently
	// inventing a conversation-only flow that the runtime cannot serve yet.
	decision := RouteDecision{
		Mode:           ModeOperation,
		TaskAction:     TaskActionNone,
		RunAction:      RunActionCreateRun,
		ExecutionShape: ExecutionShapeSingleAgent,
		Domain:         DomainGeneral,
		ContextPlan: ContextPlan{
			ToolStrategy: ToolStrategyDirect,
		},
	}
	return decision, decision.Validate()
}
