package security

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	clustermodel "github.com/cy77cc/OpsPilot/internal/modules/cluster/model"
)

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
	if err := db.AutoMigrate(&clustermodel.AdmissionPolicy{}); err != nil {
		t.Fatalf("auto migrate admission_policies: %v", err)
	}

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
	if err := db.AutoMigrate(&clustermodel.RuntimeSecurityEvent{}); err != nil {
		t.Fatalf("auto migrate runtime_security_events: %v", err)
	}

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

	hasClusterIndex := false
	hasSeverityIndex := false
	for _, idx := range indexes {
		if strings.Contains(idx, "cluster_id") {
			hasClusterIndex = true
		}
		if strings.Contains(idx, "severity") {
			hasSeverityIndex = true
		}
	}
	if !hasClusterIndex {
		t.Fatalf("missing runtime_security_events cluster_id index: %v", indexes)
	}
	if !hasSeverityIndex {
		t.Fatalf("missing runtime_security_events severity index: %v", indexes)
	}
}

func TestPhase3Model_RequiredEnums(t *testing.T) {
	if clustermodel.DisposalModeAuto == "" || clustermodel.DisposalModeManual == "" || clustermodel.DisposalModeSuggestOnly == "" {
		t.Fatalf("disposal mode constants must be defined")
	}
	if clustermodel.SecuritySeverityCritical == "" || clustermodel.SecuritySeverityHigh == "" {
		t.Fatalf("security severity constants must be defined")
	}
}

func TestPhase3Model_JSONRoundTrip(t *testing.T) {
	record := clustermodel.RuntimeSecurityEvent{
		ClusterID:      42,
		Namespace:      "prod",
		Workload:       "api",
		RuleID:         "Falco-001",
		Severity:       clustermodel.SecuritySeverityHigh,
		Source:         clustermodel.SecurityEventSourceFalco,
		RawPayloadJSON: `{"kind":"falco","priority":"high"}`,
	}

	payload, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal runtime event: %v", err)
	}

	var decoded clustermodel.RuntimeSecurityEvent
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal runtime event: %v", err)
	}

	if decoded.ClusterID != record.ClusterID || decoded.Severity != record.Severity || decoded.Source != record.Source {
		t.Fatalf("roundtrip mismatch: got %+v want %+v", decoded, record)
	}
}
