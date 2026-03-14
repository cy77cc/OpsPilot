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

	replace := converter.OnChainNodeReplace(ChainNodeInfo{
		TurnID:   "turn-1",
		NodeID:   "plan:chain-1",
		Kind:     ChainNodePlan,
		Headline: "已替换计划内容",
		Structured: map[string]any{
			"steps": []map[string]any{{"id": "step-1"}},
		},
	})
	if replace.Type != EventChainNodeReplace {
		t.Fatalf("unexpected replace event type: %s", replace.Type)
	}
	if got := replace.Data["node_id"]; got != "plan:chain-1" {
		t.Fatalf("expected node id, got %#v", got)
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
