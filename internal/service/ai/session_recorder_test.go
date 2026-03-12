package ai

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
	aistate "github.com/cy77cc/OpsPilot/internal/ai/state"
	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestChatRecorderPreservesStreamingMarkdownFormatting(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:session-recorder?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AIChatSession{}, &model.AIChatMessage{}, &model.AIChatTurn{}, &model.AIChatBlock{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	store := aistate.NewChatStore(db)
	recorder := newChatRecorder(store, 1, "global", "检查主机状态")
	ctx := context.Background()

	recorder.HandleEvent(ctx, events.Meta, map[string]any{
		"session_id": "session-1",
		"turn_id":    "turn-1",
	})
	recorder.HandleEvent(ctx, events.ThinkingDelta, map[string]any{
		"content_chunk": "## 标题\n",
	})
	recorder.HandleEvent(ctx, events.ThinkingDelta, map[string]any{
		"content_chunk": "- 列表项\n",
	})
	recorder.HandleEvent(ctx, events.Delta, map[string]any{
		"content_chunk": "## 结果\n",
	})
	recorder.HandleEvent(ctx, events.Delta, map[string]any{
		"content_chunk": "- 正常\n",
	})
	recorder.HandleEvent(ctx, events.Done, map[string]any{})

	session, err := store.GetSession(ctx, 1, "global", "session-1", true)
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if session == nil || len(session.Messages) < 2 {
		t.Fatalf("session messages = %#v", session)
	}

	assistant := session.Messages[1]
	if assistant.Thinking != "## 标题\n- 列表项\n" {
		t.Fatalf("thinking = %q, want preserved markdown", assistant.Thinking)
	}
	if assistant.Content != "## 结果\n- 正常\n" {
		t.Fatalf("content = %q, want preserved markdown", assistant.Content)
	}
	if len(session.Turns) < 2 || len(session.Turns[1].Blocks) == 0 {
		t.Fatalf("turn replay blocks = %#v", session.Turns)
	}

	var textBlockFound bool
	var thinkingBlockFound bool
	for _, block := range session.Turns[1].Blocks {
		switch block.BlockType {
		case "text":
			textBlockFound = true
			if block.ContentText != "## 结果\n- 正常\n" {
				t.Fatalf("text block = %q, want preserved markdown", block.ContentText)
			}
		case "thinking":
			thinkingBlockFound = true
			if block.ContentText != "## 标题\n- 列表项\n" {
				t.Fatalf("thinking block = %q, want preserved markdown", block.ContentText)
			}
		}
	}
	if !textBlockFound || !thinkingBlockFound {
		t.Fatalf("missing text/thinking blocks: %#v", session.Turns[1].Blocks)
	}
}
