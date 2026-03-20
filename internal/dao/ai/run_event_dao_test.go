package ai

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
)

func TestRunEventDAO_ListByRunOrdered(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunEventDAO(db)
	ctx := context.Background()

	for _, event := range []model.AIRunEvent{
		{ID: "evt-2", RunID: "run-1", SessionID: "sess-1", Seq: 2, EventType: "delta", PayloadJSON: `{"content":"b"}`},
		{ID: "evt-1", RunID: "run-1", SessionID: "sess-1", Seq: 1, EventType: "plan", PayloadJSON: `{"steps":["a"]}`},
		{ID: "evt-3", RunID: "run-2", SessionID: "sess-2", Seq: 1, EventType: "meta", PayloadJSON: `{"turn":1}`},
	} {
		event := event
		if err := dao.Create(ctx, &event); err != nil {
			t.Fatalf("create event: %v", err)
		}
	}

	events, err := dao.ListByRun(ctx, "run-1")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Seq != 1 || events[1].Seq != 2 {
		t.Fatalf("expected ordered seqs [1 2], got [%d %d]", events[0].Seq, events[1].Seq)
	}
}
