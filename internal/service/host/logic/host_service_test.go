package logic

import (
	"context"
	"testing"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHostTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Node{},
		&model.HostProbeSession{},
		&model.SSHKey{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func newTestHostService(t *testing.T) *HostService {
	t.Helper()
	db := setupHostTestDB(t)
	return NewHostService(&svc.ServiceContext{DB: db})
}

// TestHostOnboarding tests creating a host with probe token.
func TestHostOnboarding(t *testing.T) {
	svc := newTestHostService(t)
	ctx := context.Background()

	// Create a probe session first
	token := "test-token-123"
	hash := hashToken(token)
	now := time.Now()
	probe := &model.HostProbeSession{
		TokenHash:    hash,
		Name:         "test-node",
		IP:           "10.0.0.1",
		Port:         22,
		AuthType:     "password",
		Username:     "root",
		Reachable:    true,
		FactsJSON:    `{"hostname":"test-host","os":"linux","arch":"amd64"}`,
		ExpiresAt:    now.Add(10 * time.Minute),
		PasswordCipher: "encrypted-password",
	}
	if err := svc.svcCtx.DB.Create(probe).Error; err != nil {
		t.Fatalf("create probe: %v", err)
	}

	// Create host with probe token
	node, err := svc.CreateWithProbe(ctx, 1, true, CreateReq{
		ProbeToken: token,
		Name:       "my-host",
		Role:       "worker",
	})
	if err != nil {
		t.Fatalf("create with probe: %v", err)
	}

	if node.Name != "my-host" {
		t.Fatalf("expected name my-host, got %s", node.Name)
	}
	if node.IP != "10.0.0.1" {
		t.Fatalf("expected IP 10.0.0.1, got %s", node.IP)
	}
	if node.Status != "online" {
		t.Fatalf("expected status online, got %s", node.Status)
	}
}

// TestHostProbe tests the probe validation logic.
func TestHostProbe(t *testing.T) {
	svc := newTestHostService(t)
	ctx := context.Background()

	// Test validation - missing IP
	resp, err := svc.Probe(ctx, 1, ProbeReq{
		IP:       "",
		Port:     22,
		AuthType: "password",
		Username: "root",
		Password: "test",
	})
	if err != nil {
		t.Fatalf("probe should not error: %v", err)
	}
	if resp.Reachable {
		t.Fatal("expected not reachable for invalid input")
	}
	if resp.ErrorCode != "validation_error" {
		t.Fatalf("expected validation_error, got %s", resp.ErrorCode)
	}
}

// TestHostList tests listing hosts.
func TestHostList(t *testing.T) {
	svc := newTestHostService(t)
	ctx := context.Background()

	// Create test hosts
	for i := 0; i < 3; i++ {
		node := &model.Node{
			Name:   "test-node-" + string(rune('A'+i)),
			IP:     "10.0.1." + string(rune('0'+i)),
			Port:   22,
			Status: "active",
		}
		if err := svc.svcCtx.DB.Create(node).Error; err != nil {
			t.Fatalf("create node: %v", err)
		}
	}

	list, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(list) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(list))
	}
}

// TestHostGet tests getting a single host.
func TestHostGet(t *testing.T) {
	svc := newTestHostService(t)
	ctx := context.Background()

	// Create test host
	node := &model.Node{
		Name:   "test-node",
		IP:     "10.0.0.1",
		Port:   22,
		Status: "active",
	}
	if err := svc.svcCtx.DB.Create(node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	// Get host
	found, err := svc.Get(ctx, uint64(node.ID))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if found.Name != "test-node" {
		t.Fatalf("expected name test-node, got %s", found.Name)
	}
}

// TestHostUpdateStatus tests updating host status.
func TestHostUpdateStatus(t *testing.T) {
	svc := newTestHostService(t)
	ctx := context.Background()

	// Create test host
	node := &model.Node{
		Name:   "test-node",
		IP:     "10.0.0.1",
		Port:   22,
		Status: "active",
	}
	if err := svc.svcCtx.DB.Create(node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	// Update status
	if err := svc.UpdateStatus(ctx, uint64(node.ID), "maintenance"); err != nil {
		t.Fatalf("update status: %v", err)
	}

	// Verify
	found, _ := svc.Get(ctx, uint64(node.ID))
	if found.Status != "maintenance" {
		t.Fatalf("expected status maintenance, got %s", found.Status)
	}
}

// TestHostDelete tests deleting a host.
func TestHostDelete(t *testing.T) {
	svc := newTestHostService(t)
	ctx := context.Background()

	// Create test host
	node := &model.Node{
		Name:   "test-node",
		IP:     "10.0.0.1",
		Port:   22,
		Status: "active",
	}
	if err := svc.svcCtx.DB.Create(node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	// Delete
	if err := svc.Delete(ctx, uint64(node.ID)); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Verify deleted
	_, err := svc.Get(ctx, uint64(node.ID))
	if err == nil {
		t.Fatal("expected error for deleted node")
	}
}

// TestParseLabels tests label parsing.
func TestParseLabels(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"[]", nil},
		{`["web","api"]`, []string{"web", "api"}},
		{"web,api", []string{"web", "api"}},
		{`["  web  ","  api  "]`, []string{"web", "api"}},
	}

	for _, tt := range tests {
		result := ParseLabels(tt.input)
		if len(result) != len(tt.expected) {
			t.Fatalf("ParseLabels(%q) = %v, want %v", tt.input, result, tt.expected)
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Fatalf("ParseLabels(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

// TestEncodeLabels tests label encoding.
func TestEncodeLabels(t *testing.T) {
	tests := []struct {
		input    []string
		expected string
	}{
		{nil, "[]"},
		{[]string{}, "[]"},
		{[]string{"web"}, `["web"]`},
		{[]string{"web", "api"}, `["web","api"]`},
	}

	for _, tt := range tests {
		result := EncodeLabels(tt.input)
		if result != tt.expected {
			t.Fatalf("EncodeLabels(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
