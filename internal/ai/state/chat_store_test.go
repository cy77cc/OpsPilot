package state

import "testing"

func TestBuildBlocks_UsesRuntimeFinalAnswerWhenContentIsEmpty(t *testing.T) {
	record := ChatMessageRecord{
		Status: "completed",
		Runtime: &RuntimeState{
			FinalAnswer: &RuntimeFinalAnswer{
				Visible:     true,
				Streaming:   false,
				Content:     "## 服务器状态\n\n全部在线",
				RevealState: "complete",
			},
		},
	}

	blocks := buildBlocks(record)
	if len(blocks) != 1 {
		t.Fatalf("expected one text block, got %d", len(blocks))
	}
	if blocks[0].BlockType != "text" {
		t.Fatalf("expected text block, got %s", blocks[0].BlockType)
	}
	if blocks[0].ContentText != "## 服务器状态\n\n全部在线" {
		t.Fatalf("unexpected content text: %q", blocks[0].ContentText)
	}
}

func TestDecodeRuntimeState_RetainsStructuredNodeLayers(t *testing.T) {
	state := decodeRuntimeState(map[string]any{
		"turn_id": "turn-1",
		"nodes": []any{
			map[string]any{
				"node_id":    "tool:step-1",
				"kind":       "tool",
				"title":      "host_list_inventory",
				"status":     "done",
				"headline":   "已获取 2 台主机",
				"body":       "所有主机都在线",
				"structured": map[string]any{"resource": "hosts"},
				"raw":        map[string]any{"total": 2},
				"summary":    "兼容摘要",
			},
		},
		"final_answer": map[string]any{
			"visible":      true,
			"streaming":    false,
			"content":      "## 主机状态",
			"reveal_state": "complete",
		},
	})

	if state == nil {
		t.Fatalf("expected runtime state to decode")
	}
	if len(state.Nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(state.Nodes))
	}
	node := state.Nodes[0]
	if node.Headline != "已获取 2 台主机" {
		t.Fatalf("unexpected headline: %q", node.Headline)
	}
	if node.Body != "所有主机都在线" {
		t.Fatalf("unexpected body: %q", node.Body)
	}
	if node.Structured["resource"] != "hosts" {
		t.Fatalf("unexpected structured payload: %#v", node.Structured)
	}
	raw, ok := node.Raw.(map[string]any)
	if !ok {
		t.Fatalf("expected raw payload map, got %#v", node.Raw)
	}
	if raw["total"] != 2 {
		t.Fatalf("unexpected raw total: %#v", raw["total"])
	}
}
