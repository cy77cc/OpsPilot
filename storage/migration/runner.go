package migration

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const migrationTable = "schema_migrations"

const (
	markerUp   = "-- +migrate Up"
	markerDown = "-- +migrate Down"
)

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

type migrationFile struct {
	Version string
	Name    string
	Path    string
	UpSQL   string
	DownSQL string
}

type StatusItem struct {
	Version   string
	Name      string
	Applied   bool
	AppliedAt *time.Time
}

func RunMigrations(db *gorm.DB) error {
	return Migrate(db, DirectionUp, 0)
}

func Migrate(db *gorm.DB, direction Direction, steps int) error {
	if err := ensureTable(db); err != nil {
		return err
	}

	files, err := loadMigrationFiles("storage/migrations")
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}

	appliedMap, err := appliedMigrations(db)
	if err != nil {
		return err
	}

	switch direction {
	case DirectionUp:
		count := 0
		for _, mf := range files {
			if appliedMap[mf.Version] {
				continue
			}
			if strings.TrimSpace(mf.UpSQL) == "" {
				return fmt.Errorf("migration %s has empty up sql", mf.Name)
			}
			if err := db.Transaction(func(tx *gorm.DB) error {
				for _, stmt := range splitSQLStatements(mf.UpSQL) {
					if err := tx.Exec(stmt).Error; err != nil {
						return err
					}
				}
				return tx.Exec(
					"INSERT INTO "+migrationTable+" (version, name, applied_at) VALUES (?, ?, ?)",
					mf.Version,
					mf.Name,
					time.Now(),
				).Error
			}); err != nil {
				return fmt.Errorf("apply migration %s failed: %w", mf.Name, err)
			}
			count++
			if steps > 0 && count >= steps {
				break
			}
		}
		return nil
	case DirectionDown:
		applied := make([]migrationFile, 0)
		for _, mf := range files {
			if appliedMap[mf.Version] {
				applied = append(applied, mf)
			}
		}
		sort.Slice(applied, func(i, j int) bool { return applied[i].Version > applied[j].Version })
		count := 0
		for _, mf := range applied {
			if strings.TrimSpace(mf.DownSQL) == "" {
				return fmt.Errorf("migration %s has empty down sql", mf.Name)
			}
			if err := db.Transaction(func(tx *gorm.DB) error {
				for _, stmt := range splitSQLStatements(mf.DownSQL) {
					if err := tx.Exec(stmt).Error; err != nil {
						return err
					}
				}
				return tx.Exec("DELETE FROM "+migrationTable+" WHERE version = ?", mf.Version).Error
			}); err != nil {
				return fmt.Errorf("rollback migration %s failed: %w", mf.Name, err)
			}
			count++
			if steps > 0 && count >= steps {
				break
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported direction: %s", direction)
	}
}

func Status(db *gorm.DB) ([]StatusItem, error) {
	if err := ensureTable(db); err != nil {
		return nil, err
	}

	files, err := loadMigrationFiles("storage/migrations")
	if err != nil {
		return nil, err
	}

	type row struct {
		Version   string    `gorm:"column:version"`
		AppliedAt time.Time `gorm:"column:applied_at"`
	}
	var rows []row
	if err := db.Table(migrationTable).Select("version, applied_at").Scan(&rows).Error; err != nil {
		return nil, err
	}
	applied := make(map[string]time.Time, len(rows))
	for _, r := range rows {
		applied[r.Version] = r.AppliedAt
	}

	items := make([]StatusItem, 0, len(files))
	for _, mf := range files {
		if t, ok := applied[mf.Version]; ok {
			t := t
			items = append(items, StatusItem{Version: mf.Version, Name: mf.Name, Applied: true, AppliedAt: &t})
			continue
		}
		items = append(items, StatusItem{Version: mf.Version, Name: mf.Name, Applied: false})
	}
	return items, nil
}

func ensureTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  applied_at TIMESTAMP NOT NULL
);
`).Error
}

func appliedMigrations(db *gorm.DB) (map[string]bool, error) {
	type row struct {
		Version string `gorm:"column:version"`
	}
	var rows []row
	if err := db.Table(migrationTable).Select("version").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(rows))
	for _, r := range rows {
		out[r.Version] = true
	}
	return out, nil
}

func loadMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	files := make([]migrationFile, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, ok := parseVersion(entry.Name())
		if !ok {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		upSQL, downSQL, err := parseMigrationSQL(fullPath)
		if err != nil {
			return nil, err
		}
		files = append(files, migrationFile{Version: version, Name: entry.Name(), Path: fullPath, UpSQL: upSQL, DownSQL: downSQL})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Version < files[j].Version })
	return files, nil
}

func parseVersion(name string) (string, bool) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.Split(base, "_")
	if len(parts) < 2 {
		return "", false
	}
	if parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return parts[0] + "_" + parts[1], true
}

func parseMigrationSQL(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	var up, down strings.Builder
	section := "up"
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		switch strings.TrimSpace(line) {
		case markerUp:
			section = "up"
			continue
		case markerDown:
			section = "down"
			continue
		}
		if section == "down" {
			down.WriteString(line)
			down.WriteByte('\n')
		} else {
			up.WriteString(line)
			up.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(up.String()), strings.TrimSpace(down.String()), nil
}

func splitSQLStatements(sqlText string) []string {
	lines := strings.Split(sqlText, "\n")
	var b strings.Builder
	out := make([]string, 0)

	flush := func() {
		stmt := strings.TrimSpace(b.String())
		if stmt != "" {
			out = append(out, stmt)
		}
		b.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
		if strings.HasSuffix(trimmed, ";") {
			flush()
		}
	}
	flush()
	return out
}
