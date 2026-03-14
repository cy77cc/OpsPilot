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

func TestSSEConverter_ChainNodeCarriesStructuredRuntimeLayers(t *testing.T) {
	converter := NewSSEConverter()
	event := converter.OnChainNodePatch(ChainNodeInfo{
		TurnID:   "turn-1",
		NodeID:   "tool:step-1",
		Kind:     ChainNodeTool,
		Headline: "已获取 2 台主机",
		Body:     "详细结果已就绪",
		Structured: map[string]any{
			"resource": "hosts",
			"rows": []map[string]any{
				{"id": 1, "name": "test", "status": "online"},
			},
		},
		Raw: map[string]any{
			"total": 1,
		},
	})

	if got := event.Data["headline"]; got != "已获取 2 台主机" {
		t.Fatalf("expected headline to be preserved, got %#v", got)
	}
	if got := event.Data["body"]; got != "详细结果已就绪" {
		t.Fatalf("expected body to be preserved, got %#v", got)
	}
	if _, ok := event.Data["structured"].(map[string]any); !ok {
		t.Fatalf("expected structured payload map, got %#v", event.Data["structured"])
	}
	if _, ok := event.Data["raw"].(map[string]any); !ok {
		t.Fatalf("expected raw payload map, got %#v", event.Data["raw"])
	}
}
