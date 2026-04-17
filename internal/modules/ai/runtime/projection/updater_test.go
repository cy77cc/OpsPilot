package projection

import "testing"

func TestApplyEvent_IncrementsVersion(t *testing.T) {
	state := State{Version: 1}
	next := ApplyEvent(state, Event{Type: "assistant.delta", Text: "hello"})

	if next.Version != 2 {
		t.Fatalf("expected version 2, got %d", next.Version)
	}
	if len(next.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(next.Blocks))
	}
	if next.Blocks[0].Text != "hello" {
		t.Fatalf("unexpected block text: %+v", next.Blocks[0])
	}
}
