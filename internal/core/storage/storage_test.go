package storage

import (
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
)

func withStorageConfig(t *testing.T, cfg config.Config) {
	t.Helper()
	previous := config.CFG
	config.CFG = cfg
	t.Cleanup(func() {
		config.CFG = previous
	})
}

func TestNewDB_ReturnsErrorWhenNoDatabaseConfigured(t *testing.T) {
	t.Parallel()

	withStorageConfig(t, config.Config{})

	db, err := NewDB()
	if err == nil {
		t.Fatal("expected NewDB to return an error when no database is configured")
	}
	if db != nil {
		t.Fatalf("expected nil db on configuration error, got %#v", db)
	}
}

func TestNewRdb_ReturnsErrorWhenPingFails(t *testing.T) {
	t.Parallel()

	withStorageConfig(t, config.Config{
		Redis: config.Redis{
			Enable:       true,
			Addr:         "127.0.0.1:1",
			DialTimeout:  50 * time.Millisecond,
			ReadTimeout:  50 * time.Millisecond,
			WriteTimeout: 50 * time.Millisecond,
		},
	})

	rdb, err := NewRdb()
	if err == nil {
		t.Fatal("expected NewRdb to return an error when redis ping fails")
	}
	if rdb != nil {
		t.Fatalf("expected nil redis client on ping failure, got %#v", rdb)
	}
}
