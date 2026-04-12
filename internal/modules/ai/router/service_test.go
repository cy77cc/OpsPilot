package router

import "testing"

func TestRouteDecisionValidateAcceptsConversationWithoutRun(t *testing.T) {
	decision := RouteDecision{
		Mode:           ModeConversation,
		TaskAction:     TaskActionNone,
		RunAction:      RunActionNone,
		ExecutionShape: ExecutionShapeSingleAgent,
		Domain:         DomainGeneral,
		ContextPlan: ContextPlan{
			ToolStrategy: ToolStrategyDirect,
		},
	}
	if err := decision.Validate(); err != nil {
		t.Fatalf("validate conversation decision: %v", err)
	}
}

func TestRouteDecisionValidateRejectsConversationWithRunCreation(t *testing.T) {
	decision := RouteDecision{
		Mode:           ModeConversation,
		TaskAction:     TaskActionCreateTask,
		RunAction:      RunActionCreateRun,
		ExecutionShape: ExecutionShapeSingleAgent,
		Domain:         DomainGeneral,
		ContextPlan: ContextPlan{
			ToolStrategy: ToolStrategyDirect,
		},
	}
	if err := decision.Validate(); err == nil {
		t.Fatal("expected invalid conversation decision")
	}
}

func TestRouteDecisionValidateRejectsOutOfRangeConfidence(t *testing.T) {
	decision := RouteDecision{
		Mode:           ModeOperation,
		TaskAction:     TaskActionNone,
		RunAction:      RunActionCreateRun,
		ExecutionShape: ExecutionShapeSingleAgent,
		Domain:         DomainGeneral,
		Confidence:     1.5,
		ContextPlan: ContextPlan{
			ToolStrategy: ToolStrategyDirect,
		},
	}
	if err := decision.Validate(); err == nil {
		t.Fatal("expected invalid confidence")
	}
}

func TestServiceDecideReturnsBootstrapOperationFallback(t *testing.T) {
	decision, err := NewService().Decide(nil, RouteInput{Message: "hello"})
	if err != nil {
		t.Fatalf("decide fallback: %v", err)
	}
	if decision.Mode != ModeOperation {
		t.Fatalf("expected operation fallback, got %s", decision.Mode)
	}
	if decision.RunAction != RunActionCreateRun {
		t.Fatalf("expected create_run fallback, got %s", decision.RunAction)
	}
}
