package runtime

import "testing"

func TestProjectNormalizedEvent_Handoff(t *testing.T) {
	t.Parallel()

	state := &ProjectionState{}
	got := projectNormalizedEvent(NormalizedEvent{
		Kind:      NormalizedKindHandoff,
		AgentName: "OpsPilotAgent",
		Handoff: &NormalizedHandoff{
			From: "OpsPilotAgent",
			To:   "DiagnosisAgent",
		},
	}, state)

	if len(got) != 1 || got[0].Event != "agent_handoff" {
		t.Fatalf("expected one agent_handoff event, got %#v", got)
	}
}

func TestProjectNormalizedEvent_PlannerSteps(t *testing.T) {
	t.Parallel()

	state := &ProjectionState{}
	got := projectNormalizedEvent(NormalizedEvent{
		Kind:      NormalizedKindMessage,
		AgentName: "planner",
		Message: &NormalizedMessage{
			Role:    "assistant",
			Content: `{"steps":["inspect pods","check events"]}`,
		},
	}, state)

	if len(got) != 1 || got[0].Event != "plan" {
		t.Fatalf("expected one plan event, got %#v", got)
	}
}

func TestProjectNormalizedEvent_ApprovalEmitsToolApprovalAndRunState(t *testing.T) {
	t.Parallel()

	state := &ProjectionState{}
	got := projectNormalizedEvent(NormalizedEvent{
		Kind:      NormalizedKindInterrupt,
		AgentName: "executor",
		Interrupt: &NormalizedInterrupt{
			ApprovalID: "ap-1",
			CallID:     "call-1",
			ToolName:   "restart_workload",
		},
	}, state)

	if len(got) != 2 {
		t.Fatalf("expected two projected events, got %#v", got)
	}
	if got[0].Event != "tool_approval" || got[1].Event != "run_state" {
		t.Fatalf("unexpected projected events: %#v", got)
	}
}

func TestNewRunStateEvent(t *testing.T) {
	t.Parallel()

	event := NewRunStateEvent("planning", map[string]any{
		"agent": "planner",
	})

	if event.Event != "run_state" {
		t.Fatalf("expected run_state event, got %q", event.Event)
	}

	payload, ok := event.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %#v", event.Data)
	}
	if payload["status"] != "planning" || payload["agent"] != "planner" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestDecodeStepsEnvelope(t *testing.T) {
	t.Parallel()

	steps, ok := decodeStepsEnvelope(`{"steps":["inspect pods","check events"]}`)
	if !ok {
		t.Fatalf("expected decode success")
	}
	if len(steps) != 2 || steps[0] != "inspect pods" || steps[1] != "check events" {
		t.Fatalf("unexpected decoded steps: %#v", steps)
	}
}

func TestDecodeResponseEnvelope(t *testing.T) {
	t.Parallel()

	response, ok := decodeResponseEnvelope(`{"response":"root cause found"}`)
	if !ok {
		t.Fatalf("expected decode success")
	}
	if response != "root cause found" {
		t.Fatalf("unexpected decoded response: %q", response)
	}
}
