package runtime

import "testing"

func TestSSEConverter_PrimaryPathDoesNotEmitLegacyPhaseEvents(t *testing.T) {
	converter := NewSSEConverter()
	events := converter.OnPlannerStart("sess-1", "plan-1", "turn-1")

	for _, event := range events {
		if string(event.Type) == "turn_started" || string(event.Type) == "phase_started" {
			t.Fatalf("unexpected legacy event on primary path: %s", event.Type)
		}
	}
}

func TestSSEConverter_ChainMetaAndReplaceCarryCanonicalIDs(t *testing.T) {
	converter := NewSSEConverter()

	meta := converter.OnChainMeta("sess-1", "chain-1", "turn-1", "trace-1")
	if meta.Type != EventChainMeta {
		t.Fatalf("unexpected meta event type: %s", meta.Type)
	}
	if got := meta.Data["session_id"]; got != "sess-1" {
		t.Fatalf("expected session id, got %#v", got)
	}
	if got := meta.Data["chain_id"]; got != "chain-1" {
		t.Fatalf("expected chain id, got %#v", got)
	}
	if got := meta.Data["trace_id"]; got != "trace-1" {
		t.Fatalf("expected trace id, got %#v", got)
	}
}

func TestSSEConverter_PrimaryEventsCarryExpectedPayload(t *testing.T) {
	converter := NewSSEConverter()
	started := converter.OnChainStarted("turn-1")
	if started.Type != EventChainStarted {
		t.Fatalf("unexpected start event type: %s", started.Type)
	}
	if got := started.Data["turn_id"]; got != "turn-1" {
		t.Fatalf("expected turn id, got %#v", got)
	}

	done := converter.OnDone("completed")
	if done.Type != EventDone {
		t.Fatalf("unexpected done event type: %s", done.Type)
	}
	if got := done.Data["status"]; got != "completed" {
		t.Fatalf("expected completed status, got %#v", got)
	}
}
