package runtime

import "testing"

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
