package runtime

import "testing"

func TestSSEConverterApprovalRequiredIncludesCheckpointIdentity(t *testing.T) {
	converter := NewSSEConverter()
	events := converter.OnApprovalRequired(&PendingApproval{
		ID:       "approval-1",
		PlanID:   "plan-1",
		StepID:   "step-1",
		ToolName: "scale_deployment",
		Summary:  "scale nginx",
	}, "cp-1")

	if len(events) != 2 {
		t.Fatalf("event count = %d", len(events))
	}
	if got := events[1].Data["checkpoint_id"]; got != "cp-1" {
		t.Fatalf("checkpoint_id = %#v", got)
	}
}

func TestSSEConverterPlannerStartEmitsTurnLifecycle(t *testing.T) {
	converter := NewSSEConverter()
	events := converter.OnPlannerStart("sess-1", "plan-1", "turn-1")
	if len(events) != 2 {
		t.Fatalf("event count = %d", len(events))
	}
	if events[0].Type != EventTurnStarted {
		t.Fatalf("first event type = %s, want %s", events[0].Type, EventTurnStarted)
	}
	if events[1].Type != EventTurnState {
		t.Fatalf("second event type = %s, want %s", events[1].Type, EventTurnState)
	}
	if got := events[1].Data["status"]; got != "running" {
		t.Fatalf("status = %#v, want running", got)
	}
}
