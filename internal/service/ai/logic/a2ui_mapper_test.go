package logic

import (
	"reflect"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func TestNewMetaEvent(t *testing.T) {
	t.Parallel()

	got := newMetaEvent("sess-1", "run-1", 1)
	if got.Event != "meta" {
		t.Fatalf("expected meta event, got %q", got.Event)
	}

	data := got.Data.(map[string]any)
	if data["session_id"] != "sess-1" || data["run_id"] != "run-1" || data["turn"] != 1 {
		t.Fatalf("unexpected meta payload: %#v", data)
	}
}

func TestProjectAgentHandoff(t *testing.T) {
	t.Parallel()

	event := &adk.AgentEvent{
		AgentName: "OpsPilotAgent",
		Action: &adk.AgentAction{
			TransferToAgent: &adk.TransferToAgentAction{DestAgentName: "DiagnosisAgent"},
		},
	}

	got := projectAgentHandoff(event)
	if got == nil {
		t.Fatal("expected handoff event")
	}
	if got.Event != "agent_handoff" {
		t.Fatalf("unexpected event: %q", got.Event)
	}

	data := got.Data.(map[string]any)
	if data["from"] != "OpsPilotAgent" || data["to"] != "DiagnosisAgent" || data["intent"] != "diagnosis" {
		t.Fatalf("unexpected handoff payload: %#v", data)
	}
}

func TestProjectAssistantPlannerMessage(t *testing.T) {
	t.Parallel()

	state := &a2uiProjectionState{}
	events := projectAssistantMessage("planner", &schema.Message{
		Role:    schema.Assistant,
		Content: `{"steps":["inspect pods","check quota"]}`,
	}, state)

	if len(events) != 1 || events[0].Event != "plan" {
		t.Fatalf("unexpected events: %#v", events)
	}

	data := events[0].Data.(map[string]any)
	if !reflect.DeepEqual(data["steps"], []string{"inspect pods", "check quota"}) {
		t.Fatalf("unexpected plan payload: %#v", data)
	}
	if data["iteration"] != 0 {
		t.Fatalf("expected iteration 0, got %#v", data["iteration"])
	}
}

func TestProjectAssistantReplannerResponse(t *testing.T) {
	t.Parallel()

	state := &a2uiProjectionState{totalPlanSteps: 2}
	events := projectAssistantMessage("replanner", &schema.Message{
		Role:    schema.Assistant,
		Content: `{"response":"quota exhausted"}`,
	}, state)

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %#v", events)
	}
	if events[0].Event != "replan" || events[1].Event != "delta" {
		t.Fatalf("unexpected event order: %#v", events)
	}

	replan := events[0].Data.(map[string]any)
	if replan["is_final"] != true || replan["completed"] != 2 || replan["iteration"] != 1 {
		t.Fatalf("unexpected replan payload: %#v", replan)
	}

	delta := events[1].Data.(map[string]any)
	if delta["content"] != "quota exhausted" {
		t.Fatalf("unexpected delta payload: %#v", delta)
	}
}

func TestProjectAssistantToolCallAndDelta(t *testing.T) {
	t.Parallel()

	events := projectAssistantMessage("executor", &schema.Message{
		Role:    schema.Assistant,
		Content: "running checks",
		ToolCalls: []schema.ToolCall{{
			ID:   "call-1",
			Type: "function",
			Function: schema.FunctionCall{
				Name:      "host_exec",
				Arguments: `{"host_id":1,"command":"uptime"}`,
			},
		}},
	}, &a2uiProjectionState{})

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %#v", events)
	}
	if events[0].Event != "delta" || events[1].Event != "tool_call" {
		t.Fatalf("unexpected event order: %#v", events)
	}

	call := events[1].Data.(map[string]any)
	if call["call_id"] != "call-1" || call["tool_name"] != "host_exec" {
		t.Fatalf("unexpected tool call payload: %#v", call)
	}
}

func TestProjectToolMessage(t *testing.T) {
	t.Parallel()

	events := projectToolMessage(&schema.Message{
		Role:       schema.Tool,
		ToolName:   "host_exec",
		ToolCallID: "call-1",
		Content:    `{"stdout":"ok"}`,
	})

	if len(events) != 1 || events[0].Event != "tool_result" {
		t.Fatalf("unexpected events: %#v", events)
	}

	data := events[0].Data.(map[string]any)
	if data["call_id"] != "call-1" || data["tool_name"] != "host_exec" {
		t.Fatalf("unexpected tool result payload: %#v", data)
	}
}
