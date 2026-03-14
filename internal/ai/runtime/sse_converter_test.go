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
