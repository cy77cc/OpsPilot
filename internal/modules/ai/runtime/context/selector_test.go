package context

import "testing"

func TestSelectBudgeted_PrefersPinnedThenRecentThenHistory(t *testing.T) {
	history := []Message{
		{Role: "system", Content: "pinned-1", Pinned: true},
		{Role: "user", Content: "h1"},
		{Role: "assistant", Content: "h2"},
		{Role: "user", Content: "recent-1"},
		{Role: "assistant", Content: "recent-2"},
	}

	got := SelectBudgeted(history, Budget{
		Pinned:  1,
		Recent:  2,
		History: 1,
	})

	if len(got) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(got))
	}
	if got[0].Content != "pinned-1" {
		t.Fatalf("expected pinned message first, got %+v", got)
	}
	if got[2].Content != "recent-1" || got[3].Content != "recent-2" {
		t.Fatalf("expected recent messages at tail, got %+v", got)
	}
}

func TestSelectBudgeted_ClampsNegativePinned(t *testing.T) {
	history := []Message{
		{Role: "system", Content: "pinned-1", Pinned: true},
		{Role: "user", Content: "h1"},
		{Role: "assistant", Content: "h2"},
	}

	got := SelectBudgeted(history, Budget{
		Pinned:  -1,
		Recent:  1,
		History: 1,
	})

	if len(got) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(got))
	}
	if got[0].Content != "h1" || got[1].Content != "h2" {
		t.Fatalf("expected negative pinned to behave like zero, got %+v", got)
	}
}
