package runtime

import (
	"encoding/json"
	"testing"
)

func TestEncodePublicEvent_AllowsPhase1PublicEvents(t *testing.T) {
	for _, name := range []string{"init", "intent", "status", "delta", "progress", "report_ready", "error", "done"} {
		payload, err := EncodePublicEvent(name, map[string]any{"ok": true})
		if err != nil {
			t.Fatalf("encode %s event: %v", name, err)
		}

		var event StreamEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatalf("decode %s event payload: %v", name, err)
		}
		if event.Event != name {
			t.Fatalf("expected event %q, got %q", name, event.Event)
		}
	}
}

func TestEncodePublicEvent_RejectsInternalOnlyEventNames(t *testing.T) {
	for _, name := range []string{"meta", "thinking_delta", "tool_call", "tool_result", "tool_approval"} {
		if _, err := EncodePublicEvent(name, map[string]any{"ok": true}); err == nil {
			t.Fatalf("expected %s to be rejected", name)
		}
	}
}
