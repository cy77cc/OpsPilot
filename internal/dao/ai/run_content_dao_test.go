package ai

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
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
