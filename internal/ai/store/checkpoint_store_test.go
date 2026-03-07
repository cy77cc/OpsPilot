package store

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestInMemoryCheckPointStore_SetAndGet(t *testing.T) {
	store := NewInMemoryCheckPointStore()

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

func TestRedisCheckPointStore_SetAndGet(t *testing.T) {
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	store := NewRedisCheckPointStore(client)

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

func TestRedisCheckPointStore_GetNotFound(t *testing.T) {
	miniRedis := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: miniRedis.Addr()})
	store := NewRedisCheckPointStore(client)

	val, ok, err := store.Get(context.Background(), "missing")
	if err != nil {
		t.Fatalf("get missing checkpoint failed: %v", err)
	}
	if ok {
		t.Fatalf("expected missing checkpoint")
	}
	if val != nil {
		t.Fatalf("expected nil value for missing checkpoint")
	}
}
