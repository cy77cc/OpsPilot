package cluster

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPhase3Repository_ExtendsBaseRepository(t *testing.T) {
	repo := NewRepository(testDB(t))
	if repo == nil {
		t.Fatalf("repo is nil")
	}
	if repo.Phase3 == nil {
		t.Fatalf("phase3 repository extension missing")
	}
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:cluster-phase3-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}
