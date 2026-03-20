package ai

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
)

func TestRunProjectionDAO_UpsertAndGetByRunID(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIRunProjectionDAO(db)
	ctx := context.Background()

	first := &model.AIRunProjection{
		ID:             "proj-1",
		RunID:          "run-1",
		SessionID:      "sess-1",
		Version:        1,
		Status:         "running",
		ProjectionJSON: `{"status":"running"}`,
	}
	if err := dao.Upsert(ctx, first); err != nil {
		t.Fatalf("upsert first: %v", err)
	}

	second := &model.AIRunProjection{
		ID:             "proj-2",
		RunID:          "run-1",
		SessionID:      "sess-1",
		Version:        2,
		Status:         "completed",
		ProjectionJSON: `{"status":"completed"}`,
	}
	if err := dao.Upsert(ctx, second); err != nil {
		t.Fatalf("upsert second: %v", err)
	}

	got, err := dao.GetByRunID(ctx, "run-1")
	if err != nil {
		t.Fatalf("get by run id: %v", err)
	}
	if got == nil {
		t.Fatal("expected projection")
	}
	if got.Status != "completed" || got.Version != 2 {
		t.Fatalf("unexpected projection: %#v", got)
	}
}
