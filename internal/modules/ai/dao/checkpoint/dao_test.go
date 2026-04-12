package checkpoint

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

func newAIDAOTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AICheckpoint{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

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
