package ai

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
)

func TestAICheckpointDAO_UpsertAndGet(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAICheckpointDAO(db)
	ctx := context.Background()

	record := &model.AICheckpoint{
		CheckpointID: "cp-1",
		SessionID:    "sess-1",
		RunID:        "run-1",
		UserID:       3,
		Scene:        "ai",
		Payload:      []byte("first"),
	}
	if err := dao.Upsert(ctx, record); err != nil {
		t.Fatalf("upsert checkpoint: %v", err)
	}

	record.Payload = []byte("second")
	if err := dao.Upsert(ctx, record); err != nil {
		t.Fatalf("upsert checkpoint second time: %v", err)
	}

	got, err := dao.Get(ctx, "cp-1")
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if got == nil {
		t.Fatal("expected checkpoint record")
	}
	if string(got.Payload) != "second" {
		t.Fatalf("expected payload to be updated, got %q", string(got.Payload))
	}
}
