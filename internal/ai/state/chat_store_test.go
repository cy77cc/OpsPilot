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
