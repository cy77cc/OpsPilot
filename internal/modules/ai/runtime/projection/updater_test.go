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

func TestApplyEvent_PreservesWhitespaceDeltaChunks(t *testing.T) {
	state := State{RunID: "run-1", SessionID: "sess-1"}

	next := ApplyEvent(state, Event{
		ID:   "evt-1",
		Type: "assistant.delta",
		Text: " \n",
		Data: map[string]any{"agent": "executor"},
	})

	if len(next.Blocks) != 1 || next.Blocks[0].Text != " \n" {
		t.Fatalf("expected whitespace delta to be preserved, got %#v", next.Blocks)
	}
	if len(next.Contents) != 1 || next.Contents[0].BodyText != " \n" {
		t.Fatalf("expected whitespace content to be preserved, got %#v", next.Contents)
	}
}

func TestApplyEvent_CoalescesAdjacentDeltaChunks(t *testing.T) {
	state := State{RunID: "run-1", SessionID: "sess-1"}

	state = ApplyEvent(state, Event{
		ID:   "evt-1",
		Type: "assistant.delta",
		Text: "hello",
		Data: map[string]any{"agent": "executor"},
	})
	state = ApplyEvent(state, Event{
		ID:   "evt-2",
		Type: "assistant.delta",
		Text: " world",
		Data: map[string]any{"agent": "executor"},
	})

	if len(state.Blocks) != 1 || state.Blocks[0].Text != "hello world" {
		t.Fatalf("expected combined executor text, got %#v", state.Blocks)
	}
	if len(state.Contents) != 1 || state.Contents[0].BodyText != "hello world" {
		t.Fatalf("expected one coalesced content item, got %#v", state.Contents)
	}
	if len(state.Blocks[0].Items) != 1 {
		t.Fatalf("expected one content item in block, got %#v", state.Blocks[0].Items)
	}
	if state.Blocks[0].Items[0].StartEventID != "evt-1" || state.Blocks[0].Items[0].EndEventID != "evt-2" {
		t.Fatalf("expected combined event span, got %#v", state.Blocks[0].Items[0])
	}
}

func TestApplyEvent_UsesDeterministicBlockIDs(t *testing.T) {
	state := State{RunID: "run-1", SessionID: "sess-1"}

	state = ApplyEvent(state, Event{
		ID:   "evt-1",
		Type: "assistant.delta",
		Text: "hello",
		Data: map[string]any{"agent": "executor"},
	})
	state = ApplyEvent(state, Event{
		ID:   "evt-2",
		Type: "error",
		Data: map[string]any{"message": "boom", "code": "AI_STREAM_INTERNAL"},
	})

	if len(state.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %#v", state.Blocks)
	}
	if state.Blocks[0].ID != "block_executor_1" {
		t.Fatalf("expected deterministic executor block id, got %#v", state.Blocks[0])
	}
	if state.Blocks[1].ID != "block_error_2" {
		t.Fatalf("expected deterministic error block id, got %#v", state.Blocks[1])
	}
}
