package deployment

import (
	"context"
	"testing"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/testutil"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// toUint converts model.NodeID to uint for test convenience.
func toUint(id model.NodeID) uint {
	return uint(id)
}

// targetTestSuite provides test infrastructure for deployment target tests.
type targetTestSuite struct {
	db     *gorm.DB
	svcCtx *svc.ServiceContext
	logic  *Logic
}

func newTargetTestSuite(t *testing.T) *targetTestSuite {
	t.Helper()
	dbName := "target_test_" + t.Name()
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(
		&model.DeploymentTarget{},
		&model.DeploymentTargetNode{},
		&model.Cluster{},
		&model.ClusterCredential{},
		&model.Node{},
		&model.EnvironmentInstallJob{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	svcCtx := &svc.ServiceContext{DB: db}
	return &targetTestSuite{
		db:     db,
		svcCtx: svcCtx,
		logic:  &Logic{svcCtx: svcCtx},
	}
}

func (s *targetTestSuite) createTestCluster(t *testing.T) *model.Cluster {
	t.Helper()
	cluster := testutil.NewClusterBuilder().
		WithName("test-cluster").
		WithStatus("active").
		Build()
	if err := s.db.Create(cluster).Error; err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	return cluster
}

func (s *targetTestSuite) createTestCredential(t *testing.T, clusterID uint) *model.ClusterCredential {
	t.Helper()
	cred := &model.ClusterCredential{
		Name:       "test-cred",
		ClusterID:  clusterID,
		Endpoint:   "https://127.0.0.1:6443",
		Status:     "active",
		RuntimeType: "k8s",
	}
	if err := s.db.Create(cred).Error; err != nil {
		t.Fatalf("failed to create credential: %v", err)
	}
	return cred
}

func (s *targetTestSuite) createTestHost(t *testing.T) *model.Node {
	t.Helper()
	node := testutil.NewNodeBuilder().
		WithName("test-host").
		WithIP("10.0.0.1").
		WithStatus("active").
		Build()
	if err := s.db.Create(node).Error; err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	return node
}

// ============================================================================
// CreateTarget Tests
// ============================================================================

func TestCreateTarget_K8s_Success(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	cluster := suite.createTestCluster(t)

	req := TargetUpsertReq{
		Name:       "k8s-target",
		TargetType: "k8s",
		ClusterID:  cluster.ID,
		Env:        "staging",
	}

	resp, err := suite.logic.CreateTarget(ctx, 1, req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.Name != "k8s-target" {
		t.Errorf("expected name 'k8s-target', got '%s'", resp.Name)
	}
	if resp.TargetType != "k8s" {
		t.Errorf("expected target_type 'k8s', got '%s'", resp.TargetType)
	}
	if resp.Env != "staging" {
		t.Errorf("expected env 'staging', got '%s'", resp.Env)
	}
	if resp.ClusterID != cluster.ID {
		t.Errorf("expected cluster_id %d, got %d", cluster.ID, resp.ClusterID)
	}

	// Verify in database
	var count int64
	suite.db.Model(&model.DeploymentTarget{}).Where("name = ?", "k8s-target").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 target in db, got %d", count)
	}
}

func TestCreateTarget_Compose_Success(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	host := suite.createTestHost(t)

	req := TargetUpsertReq{
		Name:       "compose-target",
		TargetType: "compose",
		Env:        "production",
		Nodes: []TargetNodeReq{
			{HostID: toUint(host.ID), Role: "worker"},
		},
	}

	resp, err := suite.logic.CreateTarget(ctx, 1, req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.Name != "compose-target" {
		t.Errorf("expected name 'compose-target', got '%s'", resp.Name)
	}
	if resp.TargetType != "compose" {
		t.Errorf("expected target_type 'compose', got '%s'", resp.TargetType)
	}

	// Verify nodes were created
	var nodeCount int64
	suite.db.Model(&model.DeploymentTargetNode{}).Where("target_id = ?", resp.ID).Count(&nodeCount)
	if nodeCount != 1 {
		t.Errorf("expected 1 target node, got %d", nodeCount)
	}
}

func TestCreateTarget_K8s_MissingCluster(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	req := TargetUpsertReq{
		Name:       "k8s-target-no-cluster",
		TargetType: "k8s",
		// No ClusterID or CredentialID
	}

	_, err := suite.logic.CreateTarget(ctx, 1, req)
	if err == nil {
		t.Error("expected error for k8s target without cluster_id, got nil")
	}
}

func TestCreateTarget_Compose_MissingNodes(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	req := TargetUpsertReq{
		Name:       "compose-target-no-nodes",
		TargetType: "compose",
		Nodes:      []TargetNodeReq{}, // Empty nodes
	}

	_, err := suite.logic.CreateTarget(ctx, 1, req)
	if err == nil {
		t.Error("expected error for compose target without nodes, got nil")
	}
}

func TestCreateTarget_Compose_BindsCluster(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	cluster := suite.createTestCluster(t)
	host := suite.createHostWithStatus(t, "active")

	req := TargetUpsertReq{
		Name:       "compose-with-cluster",
		TargetType: "compose",
		ClusterID:  cluster.ID, // Should not be allowed for compose
		Nodes: []TargetNodeReq{
			{HostID: toUint(host.ID), Role: "worker"},
		},
	}

	_, err := suite.logic.CreateTarget(ctx, 1, req)
	if err == nil {
		t.Error("expected error for compose target with cluster_id, got nil")
	}
}

func (s *targetTestSuite) createHostWithStatus(t *testing.T, status string) *model.Node {
	t.Helper()
	node := testutil.NewNodeBuilder().
		WithName("test-host-" + status).
		WithIP("10.0.0.100").
		WithStatus(status).
		Build()
	if err := s.db.Create(node).Error; err != nil {
		t.Fatalf("failed to create host: %v", err)
	}
	return node
}

func TestCreateTarget_UnsupportedType(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	req := TargetUpsertReq{
		Name:       "unsupported-target",
		TargetType: "invalid",
	}

	_, err := suite.logic.CreateTarget(ctx, 1, req)
	if err == nil {
		t.Error("expected error for unsupported target type, got nil")
	}
}

// ============================================================================
// UpdateTarget Tests
// ============================================================================

func TestUpdateTarget_Name(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	cluster := suite.createTestCluster(t)

	// Create initial target with cluster binding
	target := testutil.NewDeploymentTargetBuilder().
		WithName("original-name").
		WithTargetType("k8s").
		WithClusterID(cluster.ID).
		Build()
	if err := suite.db.Create(target).Error; err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Update name - pass cluster_id to satisfy validation
	req := TargetUpsertReq{
		Name:      "updated-name",
		ClusterID: cluster.ID, // Required for k8s target validation
	}

	resp, err := suite.logic.UpdateTarget(ctx, target.ID, req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.Name != "updated-name" {
		t.Errorf("expected name 'updated-name', got '%s'", resp.Name)
	}
}

func TestUpdateTarget_Env(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	cluster := suite.createTestCluster(t)

	target := testutil.NewDeploymentTargetBuilder().
		WithName("test-target").
		WithTargetType("k8s").
		WithClusterID(cluster.ID).
		WithEnv("staging").
		Build()
	if err := suite.db.Create(target).Error; err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	req := TargetUpsertReq{
		Env:       "production",
		ClusterID: cluster.ID, // Required for k8s target validation
	}

	resp, err := suite.logic.UpdateTarget(ctx, target.ID, req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.Env != "production" {
		t.Errorf("expected env 'production', got '%s'", resp.Env)
	}
}

func TestUpdateTarget_NotFound(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	req := TargetUpsertReq{
		Name: "nonexistent",
	}

	_, err := suite.logic.UpdateTarget(ctx, 9999, req)
	if err == nil {
		t.Error("expected error for non-existent target, got nil")
	}
}

// ============================================================================
// DeleteTarget Tests
// ============================================================================

func TestDeleteTarget_Success(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	cluster := suite.createTestCluster(t)

	target := testutil.NewDeploymentTargetBuilder().
		WithName("to-delete").
		WithTargetType("k8s").
		WithClusterID(cluster.ID).
		Build()
	if err := suite.db.Create(target).Error; err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	err := suite.logic.DeleteTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify deleted
	var count int64
	suite.db.Model(&model.DeploymentTarget{}).Where("id = ?", target.ID).Count(&count)
	if count != 0 {
		t.Error("expected target to be deleted")
	}
}

func TestDeleteTarget_WithNodes(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	host := suite.createTestHost(t)

	target := testutil.NewDeploymentTargetBuilder().
		WithName("with-nodes").
		WithTargetType("compose").
		Build()
	if err := suite.db.Create(target).Error; err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Create target node
	targetNode := &model.DeploymentTargetNode{
		TargetID: target.ID,
		HostID:   toUint(host.ID),
		Role:     "worker",
		Status:   "active",
	}
	if err := suite.db.Create(targetNode).Error; err != nil {
		t.Fatalf("failed to create target node: %v", err)
	}

	err := suite.logic.DeleteTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	// Verify both deleted
	var targetCount, nodeCount int64
	suite.db.Model(&model.DeploymentTarget{}).Where("id = ?", target.ID).Count(&targetCount)
	suite.db.Model(&model.DeploymentTargetNode{}).Where("target_id = ?", target.ID).Count(&nodeCount)
	if targetCount != 0 || nodeCount != 0 {
		t.Error("expected target and nodes to be deleted")
	}
}

// ============================================================================
// GetTarget Tests
// ============================================================================

func TestGetTarget_Success(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	cluster := suite.createTestCluster(t)

	target := testutil.NewDeploymentTargetBuilder().
		WithName("get-test").
		WithTargetType("k8s").
		WithClusterID(cluster.ID).
		WithEnv("staging").
		Build()
	if err := suite.db.Create(target).Error; err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	resp, err := suite.logic.GetTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if resp.Name != "get-test" {
		t.Errorf("expected name 'get-test', got '%s'", resp.Name)
	}
	if resp.ID != target.ID {
		t.Errorf("expected id %d, got %d", target.ID, resp.ID)
	}
}

func TestGetTarget_NotFound(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	_, err := suite.logic.GetTarget(ctx, 9999)
	if err == nil {
		t.Error("expected error for non-existent target, got nil")
	}
}

// ============================================================================
// ListTargets Tests
// ============================================================================

func TestListTargets_Empty(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	resp, err := suite.logic.ListTargets(ctx, 0, 0)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty list, got %d items", len(resp))
	}
}

func TestListTargets_WithData(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	cluster := suite.createTestCluster(t)

	// Create multiple targets
	for i := 0; i < 3; i++ {
		target := testutil.NewDeploymentTargetBuilder().
			WithName("target-" + string(rune('a'+i))).
			WithTargetType("k8s").
			WithClusterID(cluster.ID).
			Build()
		if err := suite.db.Create(target).Error; err != nil {
			t.Fatalf("failed to create target: %v", err)
		}
	}

	resp, err := suite.logic.ListTargets(ctx, 0, 0)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(resp) != 3 {
		t.Errorf("expected 3 targets, got %d", len(resp))
	}
}

func TestListTargets_FilterByProject(t *testing.T) {
	suite := newTargetTestSuite(t)
	ctx := context.Background()

	cluster := suite.createTestCluster(t)

	// Create targets with different project IDs
	target1 := testutil.NewDeploymentTargetBuilder().
		WithName("project-1-target").
		WithTargetType("k8s").
		WithClusterID(cluster.ID).
		Build()
	target1.ProjectID = 1
	if err := suite.db.Create(target1).Error; err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	target2 := testutil.NewDeploymentTargetBuilder().
		WithName("project-2-target").
		WithTargetType("k8s").
		WithClusterID(cluster.ID).
		Build()
	target2.ProjectID = 2
	if err := suite.db.Create(target2).Error; err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	resp, err := suite.logic.ListTargets(ctx, 1, 0)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 target for project 1, got %d", len(resp))
	}
}
