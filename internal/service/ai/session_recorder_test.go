package ai

import (
	"testing"

	aistate "github.com/cy77cc/OpsPilot/internal/ai/state"
)

func TestChatRecorder_PreservesThoughtChainRuntimeAndFinalAnswer(t *testing.T) {
	recorder := &chatRecorder{
		assistant: aistate.ChatMessageRecord{
			Status: "streaming",
		},
	}

	recorder.handleChainStarted(map[string]any{
		"turn_id": "turn-1",
	})
	recorder.handleChainNodeOpen(map[string]any{
		"node_id":  "tool:step-1",
		"kind":     "tool",
		"title":    "host_list_inventory",
		"status":   "loading",
		"headline": "已获取 5 台主机",
		"body":     "所有主机都在线",
		"structured": map[string]any{
			"resource": "hosts",
			"rows": []map[string]any{
				{"id": 1, "name": "test", "status": "online"},
			},
		},
		"raw": map[string]any{
			"total": 5,
		},
	})
	recorder.handleFinalAnswerStart(nil)
	recorder.handleFinalAnswerDelta(map[string]any{
		"chunk": "## 主机状态\n\n",
	})
	recorder.handleFinalAnswerDelta(map[string]any{
		"chunk": "| 名称 | 状态 |\n| --- | --- |\n| test | online |\n",
	})
	recorder.handleFinalAnswerDone(nil)

	runtime := recorder.assistant.Runtime
	if runtime == nil {
		t.Fatalf("expected runtime state to be initialized")
	}
	if runtime.TurnID != "turn-1" {
		t.Fatalf("unexpected turn id: %q", runtime.TurnID)
	}
	if len(runtime.Nodes) != 1 {
		t.Fatalf("expected one runtime node, got %d", len(runtime.Nodes))
	}
	node := runtime.Nodes[0]
	if node.NodeID != "tool:step-1" {
		t.Fatalf("unexpected node id: %q", node.NodeID)
	}
	if node.Headline != "已获取 5 台主机" {
		t.Fatalf("unexpected node headline: %q", node.Headline)
	}
	if node.Body != "所有主机都在线" {
		t.Fatalf("unexpected node body: %q", node.Body)
	}
	if node.Structured["resource"] != "hosts" {
		t.Fatalf("unexpected structured payload: %#v", node.Structured)
	}
	raw, ok := node.Raw.(map[string]any)
	if !ok {
		t.Fatalf("expected raw payload map, got %#v", node.Raw)
	}
	if raw["total"] != 5 {
		t.Fatalf("unexpected raw payload: %#v", raw)
	}
	if runtime.FinalAnswer == nil {
		t.Fatalf("expected final answer state")
	}
	if got := runtime.FinalAnswer.Content; got != "## 主机状态\n\n| 名称 | 状态 |\n| --- | --- |\n| test | online |\n" {
		t.Fatalf("unexpected final answer content: %q", got)
	}
	if runtime.FinalAnswer.Streaming {
		t.Fatalf("expected final answer streaming to be false after completion")
	}
	if runtime.FinalAnswer.RevealState != "complete" {
		t.Fatalf("unexpected reveal state: %q", runtime.FinalAnswer.RevealState)
	}
}

func TestChatRecorder_ChainCollapsedClosesActiveNode(t *testing.T) {
	recorder := &chatRecorder{
		assistant: aistate.ChatMessageRecord{
			Status: "streaming",
		},
	}

	recorder.handleChainStarted(map[string]any{"turn_id": "turn-2"})
	recorder.handleChainNodeOpen(map[string]any{
		"node_id": "plan:main",
		"kind":    "plan",
		"status":  "loading",
	})
	recorder.handleChainCollapsed(nil)

	runtime := recorder.assistant.Runtime
	if runtime == nil {
		t.Fatalf("expected runtime state")
	}
	if !runtime.IsCollapsed {
		t.Fatalf("expected runtime to be marked collapsed")
	}
	if runtime.ActiveNodeID != "" {
		t.Fatalf("expected no active node after collapse, got %q", runtime.ActiveNodeID)
	}
	if len(runtime.Nodes) != 1 {
		t.Fatalf("expected one node, got %d", len(runtime.Nodes))
	}
	if runtime.Nodes[0].Status != "done" {
		t.Fatalf("expected collapsed active node to be marked done, got %q", runtime.Nodes[0].Status)
	}
}
