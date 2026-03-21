package runtime

import (
	"testing"
	"unicode/utf8"
)

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

func TestProjectNormalizedEvent_PersistedPlanStoresPlannerFirstStepOnly(t *testing.T) {
	t.Parallel()

	state := &ProjectionState{}
	projectNormalizedEvent(NormalizedEvent{
		Kind:      NormalizedKindMessage,
		AgentName: "planner",
		Message: &NormalizedMessage{
			Role:    "assistant",
			Content: `{"steps":["inspect pods","check events"]}`,
		},
	}, state)

	if state.Persisted == nil || state.Persisted.Plan == nil {
		t.Fatalf("expected persisted plan to be initialized")
	}
	if len(state.Persisted.Plan.Steps) != 1 {
		t.Fatalf("expected persisted plan to keep only first planner step, got %#v", state.Persisted.Plan.Steps)
	}
	if state.Persisted.Plan.Steps[0].Title != "inspect pods" {
		t.Fatalf("expected first planner step title to be stored, got %#v", state.Persisted.Plan.Steps[0].Title)
	}
}

func TestProjectNormalizedEvent_PersistedPlanAppendsReplannerFirstStep(t *testing.T) {
	t.Parallel()

	state := &ProjectionState{}
	projectNormalizedEvent(NormalizedEvent{
		Kind:      NormalizedKindMessage,
		AgentName: "planner",
		Message: &NormalizedMessage{
			Role:    "assistant",
			Content: `{"steps":["inspect pods","check events"]}`,
		},
	}, state)
	projectNormalizedEvent(NormalizedEvent{
		Kind:      NormalizedKindMessage,
		AgentName: "replanner",
		Message: &NormalizedMessage{
			Role:    "assistant",
			Content: `{"steps":["verify node pressure","recheck pod status"]}`,
		},
	}, state)
	projectNormalizedEvent(NormalizedEvent{
		Kind:      NormalizedKindMessage,
		AgentName: "replanner",
		Message: &NormalizedMessage{
			Role:    "assistant",
			Content: `{"response":"done"}`,
		},
	}, state)

	if state.Persisted == nil || state.Persisted.Plan == nil {
		t.Fatalf("expected persisted plan to be initialized")
	}
	if len(state.Persisted.Plan.Steps) != 2 {
		t.Fatalf("expected planner first step + replanner first step, got %#v", state.Persisted.Plan.Steps)
	}
	if state.Persisted.Plan.Steps[0].Title != "inspect pods" {
		t.Fatalf("unexpected planner step title: %#v", state.Persisted.Plan.Steps[0].Title)
	}
	if state.Persisted.Plan.Steps[1].Title != "verify node pressure" {
		t.Fatalf("unexpected replanner appended step title: %#v", state.Persisted.Plan.Steps[1].Title)
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

func TestProjectNormalizedEvent_PartialPlannerChunkIsBuffered(t *testing.T) {
	t.Parallel()

	state := &ProjectionState{}
	got := projectNormalizedEvent(NormalizedEvent{
		Kind:      NormalizedKindMessage,
		AgentName: "planner",
		Message: &NormalizedMessage{
			Role:        "assistant",
			Content:     `{"steps":["inspect pods",`,
			IsStreaming: true,
		},
	}, state)

	if len(got) != 0 {
		t.Fatalf("expected partial planner chunk to stay buffered, got %#v", got)
	}
	if state.PendingPlannerJSON != `{"steps":["inspect pods",` {
		t.Fatalf("expected planner buffer to persist partial chunk, got %q", state.PendingPlannerJSON)
	}
}

func TestProjectNormalizedEvent_ToolCall(t *testing.T) {
	event := NormalizedEvent{
		Kind:      NormalizedKindToolCall,
		AgentName: "executor",
		Tool: &NormalizedTool{
			CallID:    "call-123",
			ToolName:  "k8s_query",
			Arguments: map[string]any{"namespace": "default"},
		},
	}

	got := projectNormalizedEvent(event, &ProjectionState{})

	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Event != "tool_call" {
		t.Fatalf("expected event tool_call, got %s", got[0].Event)
	}

	data, ok := got[0].Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be map[string]any")
	}
	if data["call_id"] != "call-123" {
		t.Errorf("expected call_id=call-123, got %v", data["call_id"])
	}
	if data["tool_name"] != "k8s_query" {
		t.Errorf("expected tool_name=k8s_query, got %v", data["tool_name"])
	}
}

func TestProjectNormalizedEvent_ToolResult(t *testing.T) {
	event := NormalizedEvent{
		Kind:      NormalizedKindToolResult,
		AgentName: "executor",
		Tool: &NormalizedTool{
			CallID:   "call-123",
			ToolName: "k8s_query",
			Content:  "tool output",
		},
	}

	got := projectNormalizedEvent(event, &ProjectionState{})

	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Event != "tool_result" {
		t.Fatalf("expected event tool_result, got %s", got[0].Event)
	}

	data, ok := got[0].Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be map[string]any")
	}
	if data["content"] != "tool output" {
		t.Errorf("expected content='tool output', got %v", data["content"])
	}
}

func TestProjectNormalizedEvent_ToolCall_NilTool(t *testing.T) {
	event := NormalizedEvent{
		Kind:      NormalizedKindToolCall,
		AgentName: "executor",
		Tool:      nil,
	}

	got := projectNormalizedEvent(event, &ProjectionState{})

	if len(got) != 0 {
		t.Fatalf("expected 0 events for nil tool, got %d", len(got))
	}
}

func TestTruncateString_KeepsUTF8Integrity(t *testing.T) {
	input := "内存使用详情"
	got := truncateString(input, 5)
	if !utf8.ValidString(got) {
		t.Fatalf("expected valid utf8, got %q", got)
	}
	if got != "内存使用详" {
		t.Fatalf("unexpected truncate result: %q", got)
	}
}
