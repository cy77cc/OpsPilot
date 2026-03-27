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

func TestRunEventDAO_ListAfterEventID(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunEventDAO(db)
	ctx := context.Background()

	for _, event := range []model.AIRunEvent{
		{ID: "evt-1", RunID: "run-1", SessionID: "sess-1", Seq: 1, EventType: "meta", PayloadJSON: `{"turn":1}`},
		{ID: "evt-2", RunID: "run-1", SessionID: "sess-1", Seq: 2, EventType: "delta", PayloadJSON: `{"content":"b"}`},
		{ID: "evt-3", RunID: "run-1", SessionID: "sess-1", Seq: 3, EventType: "done", PayloadJSON: `{"status":"completed"}`},
	} {
		event := event
		if err := dao.Create(ctx, &event); err != nil {
			t.Fatalf("create event: %v", err)
		}
	}

	events, err := dao.ListAfterEventID(ctx, "run-1", "evt-1")
	if err != nil {
		t.Fatalf("list after event id: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events after cursor, got %d", len(events))
	}
	if events[0].ID != "evt-2" || events[1].ID != "evt-3" {
		t.Fatalf("unexpected replay order: %#v", events)
	}

	none, err := dao.ListAfterEventID(ctx, "run-1", "evt-3")
	if err != nil {
		t.Fatalf("list after tail event: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no events after tail cursor, got %d", len(none))
	}
}
