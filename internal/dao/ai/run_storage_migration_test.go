package ai

import (
	"fmt"
	"testing"

	"github.com/cy77cc/OpsPilot/storage/migration"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAIRunStorageTablesExist(t *testing.T) {
	db := newRunStorageMigrationTestDB(t)

	for _, table := range []string{"ai_run_events", "ai_run_projections", "ai_run_contents"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("expected %s table", table)
		}
	}
}

func newRunStorageMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := migration.RunDevAutoMigrate(db); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}
