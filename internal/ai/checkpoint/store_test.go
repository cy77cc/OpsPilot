package checkpoint

import (
	"context"
	"fmt"
	"testing"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStore_SetWritesDBAndRedis(t *testing.T) {
	db := newStoreTestDB(t)
	dao := aidao.NewAICheckpointDAO(db)
	redisClient := newFakeRedis()
	store := NewStore(dao, redisClient, "test:cp:")

	ctx := ContextWithMetadata(context.Background(), Metadata{
		SessionID: "sess-1",
		RunID:     "run-1",
		UserID:    9,
		Scene:     "cluster",
	})
	if err := store.Set(ctx, "cp-1", []byte("payload")); err != nil {
		t.Fatalf("set checkpoint: %v", err)
	}

	record, err := dao.Get(context.Background(), "cp-1")
	if err != nil {
		t.Fatalf("get checkpoint from db: %v", err)
	}
	if record == nil {
		t.Fatal("expected checkpoint to be persisted in db")
	}
	if string(record.Payload) != "payload" {
		t.Fatalf("expected db payload, got %q", string(record.Payload))
	}
	if record.SessionID != "sess-1" || record.RunID != "run-1" || record.Scene != "cluster" || record.UserID != 9 {
		t.Fatalf("expected metadata to be persisted, got %#v", record)
	}
	if got, ok := redisClient.values["test:cp:cp-1"]; !ok || string(got) != "payload" {
		t.Fatalf("expected redis payload, got %q, ok=%v", string(got), ok)
	}
}

func TestStore_GetFallsBackToDBAndWarmsRedis(t *testing.T) {
	db := newStoreTestDB(t)
	dao := aidao.NewAICheckpointDAO(db)
	redisClient := newFakeRedis()
	store := NewStore(dao, redisClient, "test:cp:")

	expiresAt := time.Now().Add(time.Hour)
	if err := dao.Upsert(context.Background(), &model.AICheckpoint{
		CheckpointID: "cp-2",
		Payload:      []byte("db-payload"),
		ExpiresAt:    &expiresAt,
	}); err != nil {
		t.Fatalf("seed checkpoint: %v", err)
	}

	got, ok, err := store.Get(context.Background(), "cp-2")
	if err != nil {
		t.Fatalf("get checkpoint: %v", err)
	}
	if !ok {
		t.Fatal("expected checkpoint to exist")
	}
	if string(got) != "db-payload" {
		t.Fatalf("expected db payload, got %q", string(got))
	}
	if warmed := string(redisClient.values["test:cp:cp-2"]); warmed != "db-payload" {
		t.Fatalf("expected redis to be warmed, got %q", warmed)
	}
}

func newStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AICheckpoint{}); err != nil {
		t.Fatalf("migrate checkpoint table: %v", err)
	}
	return db
}

type fakeRedis struct {
	values map[string][]byte
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{values: make(map[string][]byte)}
}

func (f *fakeRedis) Get(_ context.Context, key string) *redis.StringCmd {
	value, ok := f.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(string(value), nil)
}

func (f *fakeRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	switch v := value.(type) {
	case []byte:
		f.values[key] = append([]byte(nil), v...)
	default:
		f.values[key] = []byte(fmt.Sprint(v))
	}
	return redis.NewStatusResult("OK", nil)
}
