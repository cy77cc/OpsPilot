package migration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const fakeMySQLDriverName = "ops_pilot_fake_mysql_ai_approval_migration"

var registerFakeMySQLDriverOnce sync.Once
var fakeMySQLStates sync.Map

func TestAIApprovalRiskPolicyMigration(t *testing.T) {
	tempRoot := t.TempDir()
	copyMigrationFixture(t, tempRoot, "../migrations/20260317_0003_create_ai_approval_tasks.sql")
	copyMigrationFixture(t, tempRoot, "../migrations/20260321_0005_add_ai_tool_risk_policy_and_approval_locking.sql")

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

	db, err := openFakeMySQLTestDB(t, t.Name())
	if err != nil {
		t.Fatalf("open fake mysql db: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	if !db.Migrator().HasTable(&model.AIToolRiskPolicy{}) {
		t.Fatal("expected ai_tool_risk_policies table")
	}
	for _, column := range []string{"lock_expires_at", "matched_rule_id", "policy_version", "decision_source"} {
		if !db.Migrator().HasColumn(&model.AIApprovalTask{}, column) {
			t.Fatalf("expected ai_approval_tasks.%s column", column)
		}
	}
	for _, indexName := range []string{"idx_ai_approval_tasks_lock_expires_at", "idx_ai_approval_tasks_matched_rule_id"} {
		if !db.Migrator().HasIndex(&model.AIApprovalTask{}, indexName) {
			t.Fatalf("expected ai_approval_tasks.%s index", indexName)
		}
	}

	if err := Migrate(db, DirectionDown, 1); err != nil {
		t.Fatalf("rollback one migration: %v", err)
	}

	if db.Migrator().HasTable(&model.AIToolRiskPolicy{}) {
		t.Fatal("expected ai_tool_risk_policies table to be removed after rollback")
	}
	for _, column := range []string{"lock_expires_at", "matched_rule_id", "policy_version", "decision_source"} {
		if db.Migrator().HasColumn(&model.AIApprovalTask{}, column) {
			t.Fatalf("expected ai_approval_tasks.%s column to be removed after rollback", column)
		}
	}
	for _, indexName := range []string{"idx_ai_approval_tasks_lock_expires_at", "idx_ai_approval_tasks_matched_rule_id"} {
		if db.Migrator().HasIndex(&model.AIApprovalTask{}, indexName) {
			t.Fatalf("expected ai_approval_tasks.%s index to be removed after rollback", indexName)
		}
	}
	if !db.Migrator().HasTable(&model.AIApprovalTask{}) {
		t.Fatal("expected ai_approval_tasks table to remain after rollback")
	}
}

func copyMigrationFixture(t *testing.T, tempRoot, src string) {
	t.Helper()

	content, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read migration fixture %s: %v", src, err)
	}

	dstDir := filepath.Join(tempRoot, "storage", "migrations")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("create temp migration dir: %v", err)
	}

	dstPath := filepath.Join(dstDir, filepath.Base(src))
	if err := os.WriteFile(dstPath, content, 0o644); err != nil {
		t.Fatalf("write migration fixture %s: %v", dstPath, err)
	}
}

func openFakeMySQLTestDB(t *testing.T, dsn string) (*gorm.DB, error) {
	t.Helper()

	registerFakeMySQLDriverOnce.Do(func() {
		sql.Register(fakeMySQLDriverName, &fakeMySQLDriver{})
	})

	sqlDB, err := sql.Open(fakeMySQLDriverName, dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	gormDB, err := gorm.Open(mysqlDriver.New(mysqlDriver.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return gormDB, nil
}

type fakeMySQLDriver struct{}

type fakeMySQLState struct {
	mu      sync.Mutex
	dbName  string
	tables  map[string]map[string]struct{}
	indexes map[string]map[string]struct{}
	applied map[string]struct{}
}

type fakeMySQLConn struct {
	state *fakeMySQLState
}

type fakeMySQLTx struct {
	state *fakeMySQLState
}

type fakeMySQLResult struct{}

type fakeRows struct {
	columns []string
	rows    [][]driver.Value
	index   int
}

var (
	createTableNamePattern = regexp.MustCompile(`(?i)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([^\s(]+)`)
	alterTableNamePattern  = regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+([^\s]+)`)
	addColumnPattern       = regexp.MustCompile(`(?i)ADD\s+COLUMN\s+([` + "`" + `\w]+)`)
	addIndexPattern        = regexp.MustCompile(`(?i)(?:ADD\s+)?(?:UNIQUE\s+)?INDEX\s+([` + "`" + `\w]+)`)
	dropIndexPattern       = regexp.MustCompile(`(?i)DROP\s+INDEX\s+([` + "`" + `\w]+)`)
)

func (d *fakeMySQLDriver) Open(name string) (driver.Conn, error) {
	return &fakeMySQLConn{state: getFakeMySQLState(name)}, nil
}

func getFakeMySQLState(name string) *fakeMySQLState {
	if state, ok := fakeMySQLStates.Load(name); ok {
		return state.(*fakeMySQLState)
	}
	state := &fakeMySQLState{
		dbName:  "ops_pilot_test",
		tables:  map[string]map[string]struct{}{},
		indexes: map[string]map[string]struct{}{},
		applied: map[string]struct{}{},
	}
	actual, _ := fakeMySQLStates.LoadOrStore(name, state)
	return actual.(*fakeMySQLState)
}

func (c *fakeMySQLConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeMySQLStmt{conn: c, query: query}, nil
}

func (c *fakeMySQLConn) Close() error { return nil }

func (c *fakeMySQLConn) Begin() (driver.Tx, error) { return &fakeMySQLTx{state: c.state}, nil }

func (c *fakeMySQLConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &fakeMySQLTx{state: c.state}, nil
}

func (c *fakeMySQLConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.exec(query, args)
	return fakeMySQLResult{}, nil
}

func (c *fakeMySQLConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.state.query(query, args)
}

func (c *fakeMySQLConn) Ping(context.Context) error { return nil }

func (t *fakeMySQLTx) Commit() error   { return nil }
func (t *fakeMySQLTx) Rollback() error { return nil }

func (r fakeMySQLResult) LastInsertId() (int64, error) { return 0, nil }
func (r fakeMySQLResult) RowsAffected() (int64, error) { return 0, nil }

func (s *fakeMySQLState) exec(query string, args []driver.NamedValue) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized := strings.ReplaceAll(strings.TrimSpace(query), "`", "")
	upper := strings.ToUpper(normalized)

	switch {
	case strings.HasPrefix(upper, "CREATE TABLE"):
		if table := firstSubmatch(createTableNamePattern, normalized); table != "" {
			s.ensureTable(table)
			for _, indexName := range allSubmatches(addIndexPattern, normalized) {
				s.addIndex(table, indexName)
			}
		}
	case strings.HasPrefix(upper, "ALTER TABLE"):
		if table := firstSubmatch(alterTableNamePattern, normalized); table != "" {
			s.ensureTable(table)
			for _, column := range allSubmatches(addColumnPattern, normalized) {
				s.addColumn(table, column)
			}
			for _, indexName := range allSubmatches(addIndexPattern, normalized) {
				s.addIndex(table, indexName)
			}
			for _, indexName := range allSubmatches(dropIndexPattern, normalized) {
				s.dropIndex(table, indexName)
			}
			if strings.Contains(upper, "DROP COLUMN") {
				for _, column := range allSubmatches(dropColumnPattern, normalized) {
					s.dropColumn(table, column)
				}
			}
		}
	case strings.HasPrefix(upper, "DROP TABLE"):
		if table := firstSubmatch(dropTableNamePattern, normalized); table != "" {
			s.dropTable(table)
		}
	case strings.HasPrefix(upper, "INSERT INTO SCHEMA_MIGRATIONS"):
		if version := namedArgString(args, 0); version != "" {
			s.applied[version] = struct{}{}
		}
	case strings.HasPrefix(upper, "DELETE FROM SCHEMA_MIGRATIONS"):
		if version := namedArgString(args, 0); version != "" {
			delete(s.applied, version)
		}
	}
}

func (s *fakeMySQLState) query(query string, args []driver.NamedValue) (driver.Rows, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalized := strings.ReplaceAll(strings.TrimSpace(query), "`", "")
	upper := strings.ToUpper(normalized)

	switch {
	case strings.HasPrefix(upper, "SELECT DATABASE()"):
		return newFakeRows([]string{"DATABASE()"}, [][]driver.Value{{s.dbName}}), nil
	case strings.Contains(upper, "FROM INFORMATION_SCHEMA.SCHEMATA"):
		return newFakeRows([]string{"SCHEMA_NAME"}, [][]driver.Value{{s.dbName}}), nil
	case strings.Contains(upper, "FROM INFORMATION_SCHEMA.TABLES"):
		table := namedArgString(args, 1)
		if s.hasTable(table) {
			return newFakeRows([]string{"count(*)"}, [][]driver.Value{{int64(1)}}), nil
		}
		return newFakeRows([]string{"count(*)"}, [][]driver.Value{{int64(0)}}), nil
	case strings.Contains(upper, "FROM INFORMATION_SCHEMA.COLUMNS"):
		table := namedArgString(args, 1)
		column := namedArgString(args, 2)
		if s.hasColumn(table, column) {
			return newFakeRows([]string{"count(*)"}, [][]driver.Value{{int64(1)}}), nil
		}
		return newFakeRows([]string{"count(*)"}, [][]driver.Value{{int64(0)}}), nil
	case strings.Contains(upper, "FROM INFORMATION_SCHEMA.STATISTICS"):
		table := namedArgString(args, 1)
		indexName := namedArgString(args, 2)
		if s.hasIndex(table, indexName) {
			return newFakeRows([]string{"count(*)"}, [][]driver.Value{{int64(1)}}), nil
		}
		return newFakeRows([]string{"count(*)"}, [][]driver.Value{{int64(0)}}), nil
	case strings.Contains(upper, "FROM SCHEMA_MIGRATIONS"):
		versions := s.appliedVersions()
		rows := make([][]driver.Value, 0, len(versions))
		for _, version := range versions {
			rows = append(rows, []driver.Value{version})
		}
		return newFakeRows([]string{"version"}, rows), nil
	default:
		return newFakeRows([]string{"result"}, nil), nil
	}
}

func (s *fakeMySQLState) ensureTable(name string) {
	if _, ok := s.tables[name]; !ok {
		s.tables[name] = map[string]struct{}{}
	}
	if _, ok := s.indexes[name]; !ok {
		s.indexes[name] = map[string]struct{}{}
	}
}

func (s *fakeMySQLState) dropTable(name string) {
	delete(s.tables, name)
	delete(s.indexes, name)
}

func (s *fakeMySQLState) addColumn(table, column string) {
	s.ensureTable(table)
	s.tables[table][column] = struct{}{}
}

func (s *fakeMySQLState) dropColumn(table, column string) {
	cols, ok := s.tables[table]
	if !ok {
		return
	}
	delete(cols, column)
}

func (s *fakeMySQLState) addIndex(table, indexName string) {
	s.ensureTable(table)
	s.indexes[table][indexName] = struct{}{}
}

func (s *fakeMySQLState) dropIndex(table, indexName string) {
	indexes, ok := s.indexes[table]
	if !ok {
		return
	}
	delete(indexes, indexName)
}

func (s *fakeMySQLState) hasTable(name string) bool {
	_, ok := s.tables[name]
	return ok
}

func (s *fakeMySQLState) hasColumn(table, column string) bool {
	cols, ok := s.tables[table]
	if !ok {
		return false
	}
	_, ok = cols[column]
	return ok
}

func (s *fakeMySQLState) hasIndex(table, indexName string) bool {
	indexes, ok := s.indexes[table]
	if !ok {
		return false
	}
	_, ok = indexes[indexName]
	return ok
}

func (s *fakeMySQLState) appliedVersions() []string {
	versions := make([]string, 0, len(s.applied))
	for version := range s.applied {
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return versions
}

type fakeMySQLStmt struct {
	conn  *fakeMySQLConn
	query string
}

func (s *fakeMySQLStmt) Close() error  { return nil }
func (s *fakeMySQLStmt) NumInput() int { return -1 }

func (s *fakeMySQLStmt) Exec(args []driver.Value) (driver.Result, error) {
	s.conn.state.exec(s.query, namedValues(args))
	return fakeMySQLResult{}, nil
}

func (s *fakeMySQLStmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.conn.state.query(s.query, namedValues(args))
}

func newFakeRows(columns []string, rows [][]driver.Value) driver.Rows {
	return &fakeRows{columns: columns, rows: rows}
}

func (r *fakeRows) Columns() []string { return r.columns }

func (r *fakeRows) Close() error { return nil }

func (r *fakeRows) Next(dest []driver.Value) error {
	if r.index >= len(r.rows) {
		return io.EOF
	}
	row := r.rows[r.index]
	r.index++
	for i := range dest {
		if i < len(row) {
			dest[i] = row[i]
		} else {
			dest[i] = nil
		}
	}
	return nil
}

func firstSubmatch(pattern *regexp.Regexp, input string) string {
	match := pattern.FindStringSubmatch(input)
	if len(match) < 2 {
		return ""
	}
	return strings.Trim(match[1], "`")
}

func allSubmatches(pattern *regexp.Regexp, input string) []string {
	matches := pattern.FindAllStringSubmatch(input, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		out = append(out, strings.Trim(match[1], "`"))
	}
	return out
}

func namedArgString(args []driver.NamedValue, index int) string {
	if index < 0 || index >= len(args) {
		return ""
	}
	return fmt.Sprint(args[index].Value)
}

func namedValues(values []driver.Value) []driver.NamedValue {
	out := make([]driver.NamedValue, len(values))
	for i, value := range values {
		out[i] = driver.NamedValue{Ordinal: i + 1, Value: value}
	}
	return out
}

var (
	dropColumnPattern    = regexp.MustCompile(`(?i)DROP\s+COLUMN\s+([` + "`" + `\w]+)`)
	dropTableNamePattern = regexp.MustCompile(`(?i)^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?([^\s;]+)`)
)
