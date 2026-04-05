package cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cy77cc/OpsPilot/internal/model"
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

func TestPhase3Schema_AdmissionPolicyColumns(t *testing.T) {
	db := testDB(t)
	applyMigrationFromFile(t, db, "storage/migrations/20260405_0002_create_phase3_security_tables.sql")

	rows, err := db.Raw("PRAGMA table_info(admission_policies)").Rows()
	if err != nil {
		t.Fatalf("table_info query: %v", err)
	}
	defer rows.Close()

	columns := make([]string, 0, 12)
	for rows.Next() {
		var (
			cid       int
			name      string
			colType   string
			notnull   int
			defaultV  any
			primaryID int
		)
		if err := rows.Scan(&cid, &name, &colType, &notnull, &defaultV, &primaryID); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		columns = append(columns, name)
	}

	for _, required := range []string{"id", "cluster_id", "policy_name", "version", "status", "created_at", "updated_at"} {
		if !slices.Contains(columns, required) {
			t.Fatalf("missing column %q in admission_policies: %v", required, columns)
		}
	}
}

func TestPhase3Schema_RuntimeEventIndexes(t *testing.T) {
	db := testDB(t)
	applyMigrationFromFile(t, db, "storage/migrations/20260405_0002_create_phase3_security_tables.sql")

	rows, err := db.Raw("PRAGMA index_list(runtime_security_events)").Rows()
	if err != nil {
		t.Fatalf("index_list query: %v", err)
	}
	defer rows.Close()

	indexes := make([]string, 0, 4)
	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan index_list: %v", err)
		}
		indexes = append(indexes, name)
	}

	for _, required := range []string{"idx_runtime_security_events_cluster_severity", "idx_runtime_security_events_cluster_rule"} {
		if !slices.Contains(indexes, required) {
			t.Fatalf("missing runtime_security_events index %q: %v", required, indexes)
		}
	}
}

func TestPhase3Model_RequiredEnums(t *testing.T) {
	if model.DisposalModeAuto == "" || model.DisposalModeManual == "" || model.DisposalModeSuggestOnly == "" {
		t.Fatalf("disposal mode constants must be defined")
	}
	if model.SecuritySeverityCritical == "" || model.SecuritySeverityHigh == "" {
		t.Fatalf("security severity constants must be defined")
	}
}

func TestPhase3Model_JSONRoundTrip(t *testing.T) {
	record := model.RuntimeSecurityEvent{
		ClusterID:      42,
		Namespace:      "prod",
		Workload:       "api",
		RuleID:         "Falco-001",
		Severity:       model.SecuritySeverityHigh,
		Source:         model.SecurityEventSourceFalco,
		RawPayloadJSON: `{"kind":"falco","priority":"high"}`,
	}

	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal runtime event: %v", err)
	}

	var decoded model.RuntimeSecurityEvent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal runtime event: %v", err)
	}

	if decoded.ClusterID != record.ClusterID || decoded.Severity != record.Severity || decoded.Source != record.Source {
		t.Fatalf("roundtrip mismatch: got %+v want %+v", decoded, record)
	}
}

func applyMigrationFromFile(t *testing.T, db *gorm.DB, relativePath string) {
	t.Helper()

	candidates := []string{
		filepath.Clean(relativePath),
		filepath.Clean(filepath.Join("..", "..", "..", relativePath)),
	}
	var (
		content []byte
		err     error
	)
	for _, candidate := range candidates {
		content, err = os.ReadFile(candidate)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("read migration %s: %v", relativePath, err)
	}
	statements := strings.Split(string(content), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("exec migration statement %q: %v", stmt, err)
		}
	}
}
