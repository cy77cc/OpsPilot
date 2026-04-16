package runtime

import "testing"

func TestProjectNormalizedEvent_HandoffDelegationEmitsStateTransitions(t *testing.T) {
	state := &ProjectionState{Persisted: &PersistedRuntime{}}
	events := projectNormalizedEvent(NormalizedEvent{
		Kind: NormalizedKindHandoff,
		Handoff: &NormalizedHandoff{
			From: "supervisor",
			To:   "monitor",
		},
	}, state)

	if len(events) != 3 {
		t.Fatalf("expected 3 events for delegation handoff, got %d: %#v", len(events), events)
	}
	if events[0].Event != "run_state" || events[1].Event != "agent_handoff" || events[2].Event != "run_state" {
		t.Fatalf("unexpected event sequence: %#v", events)
	}

	firstData, _ := events[0].Data.(map[string]any)
	lastData, _ := events[2].Data.(map[string]any)
	if firstData["status"] != string(RunStateDelegating) {
		t.Fatalf("expected first run_state delegating, got %#v", firstData)
	}
	if lastData["status"] != string(RunStateWaitingSubagent) {
		t.Fatalf("expected second run_state waiting_subagent, got %#v", lastData)
	}

	if state.RunPhase != string(RunStateWaitingSubagent) {
		t.Fatalf("expected run phase waiting_subagent, got %q", state.RunPhase)
	}
	if state.Persisted.Phase != "executing" || state.Persisted.PhaseLabel != "等待专家摘要" {
		t.Fatalf("unexpected persisted phase: %#v", state.Persisted)
	}
	if state.Persisted.Status == nil || state.Persisted.Status.Kind != string(RunStateWaitingSubagent) {
		t.Fatalf("expected waiting_subagent persisted status, got %#v", state.Persisted.Status)
	}
}

func TestProjectNormalizedEvent_HandoffNonDelegationSkipsDelegationStates(t *testing.T) {
	state := &ProjectionState{Persisted: &PersistedRuntime{}}
	events := projectNormalizedEvent(NormalizedEvent{
		Kind: NormalizedKindHandoff,
		Handoff: &NormalizedHandoff{
			From: "supervisor",
			To:   "diagnosis",
		},
	}, state)

	if len(events) != 1 {
		t.Fatalf("expected one handoff event, got %d: %#v", len(events), events)
	}
	if events[0].Event != "agent_handoff" {
		t.Fatalf("expected only agent_handoff event, got %#v", events[0])
	}
}
