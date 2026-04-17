package run

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

func TestRunContentDAO_CreateAndGet(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunContentDAO(db)
	ctx := context.Background()

	content := &model.AIRunContent{
		ID:          "content-1",
		RunID:       "run-1",
		SessionID:   "sess-1",
		ContentKind: "executor_content",
		Encoding:    "text",
		SummaryText: "summary",
		BodyText:    "body",
		SizeBytes:   4,
	}
	if err := dao.Create(ctx, content); err != nil {
		t.Fatalf("create content: %v", err)
	}

	got, err := dao.Get(ctx, "content-1")
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if got == nil {
		t.Fatal("expected content")
	}
	if got.BodyText != "body" || got.ContentKind != "executor_content" {
		t.Fatalf("unexpected content: %#v", got)
	}
}

func TestRunContentDAO_UpsertUpdatesExistingRow(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunContentDAO(db)
	ctx := context.Background()

	content := &model.AIRunContent{
		ID:          "content-1",
		RunID:       "run-1",
		SessionID:   "sess-1",
		ContentKind: "executor_content",
		Encoding:    "text",
		SummaryText: "summary",
		BodyText:    "body",
		SizeBytes:   4,
	}
	if err := dao.Create(ctx, content); err != nil {
		t.Fatalf("create content: %v", err)
	}

	content.BodyText = "body extended"
	content.SummaryText = "body extended"
	content.SizeBytes = int64(len(content.BodyText))
	if err := dao.Upsert(ctx, content); err != nil {
		t.Fatalf("upsert content: %v", err)
	}

	got, err := dao.Get(ctx, "content-1")
	if err != nil {
		t.Fatalf("get content: %v", err)
	}
	if got == nil {
		t.Fatal("expected content")
	}
	if got.BodyText != "body extended" || got.SizeBytes != int64(len("body extended")) {
		t.Fatalf("unexpected updated content: %#v", got)
	}
}
