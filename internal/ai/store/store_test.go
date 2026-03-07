package store

import (
	"context"
	"testing"

	"github.com/cy77cc/k8s-manage/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:ai_checkpoint_store_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&model.AICheckPoint{}); err != nil {
		t.Fatalf("migrate checkpoint table failed: %v", err)
	}
	return db
}

func TestDBCheckPointStore_SetAndGet(t *testing.T) {
	db := newStoreTestDB(t)
	store := NewDBCheckPointStore(db)

	ctx := context.Background()
	if err := store.Set(ctx, "cp-1", []byte("value-1")); err != nil {
		t.Fatalf("set checkpoint failed: %v", err)
	}

	val, ok, err := store.Get(ctx, "cp-1")
	if err != nil {
		t.Fatalf("get checkpoint failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected checkpoint to exist")
	}
	if string(val) != "value-1" {
		t.Fatalf("unexpected value: %q", string(val))
	}
}

func TestDBCheckPointStore_SetUpsert(t *testing.T) {
	db := newStoreTestDB(t)
	store := NewDBCheckPointStore(db)

	ctx := context.Background()
	if err := store.Set(ctx, "cp-upsert", []byte("old")); err != nil {
		t.Fatalf("initial set failed: %v", err)
	}
	if err := store.Set(ctx, "cp-upsert", []byte("new")); err != nil {
		t.Fatalf("upsert set failed: %v", err)
	}

	val, ok, err := store.Get(ctx, "cp-upsert")
	if err != nil {
		t.Fatalf("get checkpoint failed: %v", err)
	}
	if !ok || string(val) != "new" {
		t.Fatalf("expected upserted value new, got ok=%v val=%q", ok, string(val))
	}
}

func TestDBCheckPointStore_GetNotFound(t *testing.T) {
	db := newStoreTestDB(t)
	store := NewDBCheckPointStore(db)

	val, ok, err := store.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("get missing checkpoint failed: %v", err)
	}
	if ok {
		t.Fatalf("expected checkpoint to be missing")
	}
	if val != nil {
		t.Fatalf("expected nil value for missing checkpoint")
	}
}

func TestDBCheckPointStore_NilDB(t *testing.T) {
	store := &DBCheckPointStore{}

	if err := store.Set(context.Background(), "x", []byte("v")); err == nil {
		t.Fatalf("expected error for nil db in set")
	}
	_, _, err := store.Get(context.Background(), "x")
	if err == nil {
		t.Fatalf("expected error for nil db in get")
	}
}
