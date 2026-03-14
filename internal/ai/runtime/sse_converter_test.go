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
	if len(events) != 3 {
		t.Fatalf("event count = %d", len(events))
	}
	if events[0].Type != EventTurnStarted {
		t.Fatalf("first event type = %s, want %s", events[0].Type, EventTurnStarted)
	}
	if string(events[1].Type) != "phase_started" {
		t.Fatalf("second event type = %s, want phase_started", events[1].Type)
	}
	if got := events[1].Data["phase"]; got != "planning" {
		t.Fatalf("phase = %#v, want planning", got)
	}
	if events[2].Type != EventTurnState {
		t.Fatalf("third event type = %s, want %s", events[2].Type, EventTurnState)
	}
	if got := events[2].Data["status"]; got != "running" {
		t.Fatalf("status = %#v, want running", got)
	}
}

func TestSSEConverterExecuteCompleteEmitsPhaseCompleteBeforeTurnCompletion(t *testing.T) {
	converter := NewSSEConverter()
	events := converter.OnExecuteComplete()
	if len(events) != 2 {
		t.Fatalf("event count = %d", len(events))
	}
	if string(events[0].Type) != "phase_complete" {
		t.Fatalf("first event type = %s, want phase_complete", events[0].Type)
	}
	if got := events[0].Data["phase"]; got != "executing" {
		t.Fatalf("phase = %#v, want executing", got)
	}
	if events[1].Type != EventTurnState {
		t.Fatalf("second event type = %s, want %s", events[1].Type, EventTurnState)
	}
	if got := events[1].Data["status"]; got != "completed" {
		t.Fatalf("status = %#v, want completed", got)
	}
}
