// Package host 提供主机运维相关的运行时辅助函数测试。
package host

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResolveNodeByTarget_ReturnsNilForLocalhost(t *testing.T) {
	svcCtx := &svc.ServiceContext{DB: nil}
	node, err := resolveNodeByTarget(svcCtx, "localhost")
	if err != nil {
		t.Fatalf("unexpected error for localhost: %v", err)
	}
	if node != nil {
		t.Fatal("expected nil node for localhost")
	}
}

func TestResolveNodeByTarget_ReturnsNilForEmptyTarget(t *testing.T) {
	svcCtx := &svc.ServiceContext{DB: nil}
	node, err := resolveNodeByTarget(svcCtx, "")
	if err != nil {
		t.Fatalf("unexpected error for empty target: %v", err)
	}
	if node != nil {
		t.Fatal("expected nil node for empty target")
	}
}

func TestResolveNodeByTarget_ReturnsErrorForInvalidTarget(t *testing.T) {
	// When DB is nil, should return error for non-localhost targets
	svcCtx := &svc.ServiceContext{DB: nil}
	_, err := resolveNodeByTarget(svcCtx, "nonexistent-server")
	if err == nil {
		t.Fatal("expected error for invalid target with nil DB")
	}
}

func TestResolveNodeByTarget_ErrorMessageIsActionable(t *testing.T) {
	// This test verifies the error message guides the user to discover valid targets
	// When a target is not found, the error should suggest using host_list_inventory
	db := newRuntimeTestDB(t)
	svcCtx := &svc.ServiceContext{DB: db}

	_, err := resolveNodeByTarget(svcCtx, "test-server")
	if err == nil {
		t.Fatal("expected error for invalid target")
	}

	errMsg := err.Error()

	// The error message should be actionable and guide recovery
	// It should contain:
	// 1. The invalid target value
	// 2. Suggestion to use host_list_inventory
	// 3. What valid targets look like (ID, IP, hostname)
	if !strings.Contains(errMsg, "test-server") {
		t.Errorf("error message should contain the invalid target 'test-server', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "host_list_inventory") {
		t.Errorf("error message should suggest using host_list_inventory, got: %s", errMsg)
	}
}

func TestResolveNodeByTarget_ErrorMessageContainsTargetTypes(t *testing.T) {
	db := newRuntimeTestDB(t)
	svcCtx := &svc.ServiceContext{DB: db}

	_, err := resolveNodeByTarget(svcCtx, "invalid-host")
	if err == nil {
		t.Fatal("expected error for invalid target")
	}

	errMsg := err.Error()

	// Should mention valid target types
	validTypes := []string{"ID", "IP", "hostname"}
	found := 0
	for _, vt := range validTypes {
		if strings.Contains(errMsg, vt) {
			found++
		}
	}
	if found < 2 {
		t.Errorf("error message should mention at least 2 valid target types (ID, IP, hostname), got: %s", errMsg)
	}
}

func TestResolveNodeByTarget_FindsNodeByID(t *testing.T) {
	db := newRuntimeTestDB(t)
	svcCtx := &svc.ServiceContext{DB: db}

	// Seed a node
	node := &model.Node{Name: "web-01", IP: "10.0.0.1", Hostname: "web-01.local", Status: "active"}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	// Find by ID
	found, err := resolveNodeByTarget(svcCtx, fmt.Sprintf("%d", node.ID))
	if err != nil {
		t.Fatalf("expected to find node by ID, got error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find node by ID")
	}
	if found.ID != node.ID {
		t.Errorf("expected node ID %d, got %d", node.ID, found.ID)
	}
}

func TestResolveNodeByTarget_FindsNodeByIP(t *testing.T) {
	db := newRuntimeTestDB(t)
	svcCtx := &svc.ServiceContext{DB: db}

	node := &model.Node{Name: "web-02", IP: "10.0.0.2", Hostname: "web-02.local", Status: "active"}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	found, err := resolveNodeByTarget(svcCtx, "10.0.0.2")
	if err != nil {
		t.Fatalf("expected to find node by IP, got error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find node by IP")
	}
}

func TestResolveNodeByTarget_FindsNodeByHostname(t *testing.T) {
	db := newRuntimeTestDB(t)
	svcCtx := &svc.ServiceContext{DB: db}

	node := &model.Node{Name: "web-03", IP: "10.0.0.3", Hostname: "web-03.local", Status: "active"}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	found, err := resolveNodeByTarget(svcCtx, "web-03.local")
	if err != nil {
		t.Fatalf("expected to find node by hostname, got error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find node by hostname")
	}
}

func TestResolveNodeByTarget_FindsNodeByName(t *testing.T) {
	db := newRuntimeTestDB(t)
	svcCtx := &svc.ServiceContext{DB: db}

	node := &model.Node{Name: "web-04", IP: "10.0.0.4", Hostname: "web-04.local", Status: "active"}
	if err := db.Create(node).Error; err != nil {
		t.Fatalf("seed node: %v", err)
	}

	found, err := resolveNodeByTarget(svcCtx, "web-04")
	if err != nil {
		t.Fatalf("expected to find node by name, got error: %v", err)
	}
	if found == nil {
		t.Fatal("expected to find node by name")
	}
}

func newRuntimeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Node{}); err != nil {
		t.Fatalf("migrate node table: %v", err)
	}
	return db
}
