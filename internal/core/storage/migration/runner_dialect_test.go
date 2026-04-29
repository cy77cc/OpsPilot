package migration

import (
	"os"
	"path/filepath"
	"testing"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadMigrationFiles_FiltersByDialect(t *testing.T) {
	tempRoot := t.TempDir()
	writeMigrationFixture(t, tempRoot, "20260429_0001_common.sql", `
-- +migrate Up
CREATE TABLE common_table (id INTEGER PRIMARY KEY);
-- +migrate Down
DROP TABLE IF EXISTS common_table;
`)
	writeMigrationFixture(t, tempRoot, "20260429_0002_sqlite_only.sqlite.sql", `
-- +migrate Up
CREATE TABLE sqlite_only_table (id INTEGER PRIMARY KEY);
-- +migrate Down
DROP TABLE IF EXISTS sqlite_only_table;
`)
	writeMigrationFixture(t, tempRoot, "20260429_0003_postgres_only.postgres.sql", `
-- +migrate Up
CREATE TABLE postgres_only_table (id BIGINT PRIMARY KEY);
-- +migrate Down
DROP TABLE IF EXISTS postgres_only_table;
`)

	files, err := loadMigrationFiles(filepath.Join(tempRoot, "storage", "migrations"), "sqlite")
	if err != nil {
		t.Fatalf("load sqlite migration files: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 sqlite-visible migrations, got %d", len(files))
	}
	if files[0].Name != "20260429_0001_common.sql" {
		t.Fatalf("expected common migration first, got %q", files[0].Name)
	}
	if files[1].Name != "20260429_0002_sqlite_only.sqlite.sql" {
		t.Fatalf("expected sqlite-specific migration second, got %q", files[1].Name)
	}
}

func TestRunMigrations_SQLiteAppliesSharedAndDialectSpecificFiles(t *testing.T) {
	tempRoot := t.TempDir()
	copyMigrationFixture(t, tempRoot, filepath.Join("..", "..", "..", "..", "storage", "migrations", "20260429_0001_create_host_plugin_tables.sqlite.sql"))
	writeMigrationFixture(t, tempRoot, "20260429_0002_create_runner_shared_table.sql", `
-- +migrate Up
CREATE TABLE runner_shared_table (
  id INTEGER PRIMARY KEY
);
-- +migrate Down
DROP TABLE IF EXISTS runner_shared_table;
`)
	writeMigrationFixture(t, tempRoot, "20260429_0003_break_mysql_only.mysql.sql", `
-- +migrate Up
THIS IS NOT VALID SQLITE;
-- +migrate Down
DROP TABLE IF EXISTS should_never_exist;
`)

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(tempRoot); err != nil {
		t.Fatalf("chdir temp root: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(oldwd); chdirErr != nil {
			t.Errorf("restore cwd: %v", chdirErr)
		}
	})

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("run sqlite migrations: %v", err)
	}

	if !db.Migrator().HasTable(&hostpluginmodel.HostPlugin{}) {
		t.Fatal("expected host_plugins table from sqlite hostplugin migration")
	}
	if !db.Migrator().HasTable("runner_shared_table") {
		t.Fatal("expected shared migration table")
	}
	if db.Migrator().HasTable("should_never_exist") {
		t.Fatal("expected mysql-only migration to be skipped for sqlite")
	}

	items, err := Status(db)
	if err != nil {
		t.Fatalf("status after sqlite migrations: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 visible sqlite status items, got %d", len(items))
	}
	for _, item := range items {
		if !item.Applied {
			t.Fatalf("expected sqlite-visible migration %q to be applied", item.Name)
		}
		if item.Name == "20260429_0003_break_mysql_only.mysql.sql" {
			t.Fatal("expected mysql-only migration to be omitted from sqlite status")
		}
	}
}

func writeMigrationFixture(t *testing.T, tempRoot, name, content string) {
	t.Helper()

	dstDir := filepath.Join(tempRoot, "storage", "migrations")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("create temp migration dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write migration fixture %s: %v", name, err)
	}
}
