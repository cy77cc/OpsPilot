package cluster

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/k8s-manage/internal/cache"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// clusterHandlerTestSuite provides test infrastructure for cluster handler tests.
type clusterHandlerTestSuite struct {
	db     *gorm.DB
	svcCtx *svc.ServiceContext
}

func newClusterHandlerTestSuite(t *testing.T) *clusterHandlerTestSuite {
	t.Helper()
	dbName := "cluster_handler_test_" + t.Name()
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	if err := db.AutoMigrate(
		&model.Cluster{},
		&model.ClusterCredential{},
		&model.ClusterNode{},
	); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create cache facade for testing
	l1 := expirable.NewLRU[string, string](100, nil, time.Hour)
	cacheFacade := cache.NewFacade(l1, nil) // No L2 for tests

	return &clusterHandlerTestSuite{
		db: db,
		svcCtx: &svc.ServiceContext{
			DB:          db,
			CacheFacade: cacheFacade,
		},
	}
}

func (s *clusterHandlerTestSuite) createTestCluster(t *testing.T, name, status string) *model.Cluster {
	t.Helper()
	cluster := testutil.NewClusterBuilder().
		WithName(name).
		WithStatus(status).
		Build()
	if err := s.db.Create(cluster).Error; err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	return cluster
}

func (s *clusterHandlerTestSuite) createTestClusterNode(t *testing.T, clusterID uint, name, role, status string) *model.ClusterNode {
	t.Helper()
	node := &model.ClusterNode{
		ClusterID: clusterID,
		Name:      name,
		IP:        "10.0.0.1",
		Role:      role,
		Status:    status,
	}
	if err := s.db.Create(node).Error; err != nil {
		t.Fatalf("failed to create cluster node: %v", err)
	}
	return node
}

// parseClusterResponse extracts data from standard response format.
func parseClusterResponse(body []byte) (map[string]any, error) {
	var resp struct {
		Code int            `json:"code"`
		Msg  string         `json:"msg"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// ============================================================================
// GetClusters Handler Tests
// ============================================================================

func TestGetClustersHandler_Empty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterHandlerTestSuite(t)
	h := NewHandler(suite.svcCtx)

	r := gin.New()
	r.GET("/clusters", h.GetClusters)

	req := httptest.NewRequest(http.MethodGet, "/clusters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	data, err := parseClusterResponse(w.Body.Bytes())
	if err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	list, ok := data["list"].([]any)
	if !ok {
		t.Fatalf("expected list in data, got: %v", data)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestGetClustersHandler_WithData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterHandlerTestSuite(t)
	h := NewHandler(suite.svcCtx)

	suite.createTestCluster(t, "cluster-1", "active")
	suite.createTestCluster(t, "cluster-2", "inactive")

	r := gin.New()
	r.GET("/clusters", h.GetClusters)

	req := httptest.NewRequest(http.MethodGet, "/clusters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	data, _ := parseClusterResponse(w.Body.Bytes())
	list := data["list"].([]any)
	if len(list) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(list))
	}
}

// ============================================================================
// GetClusterDetail Handler Tests
// ============================================================================

func TestGetClusterDetailHandler_Found(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterHandlerTestSuite(t)
	h := NewHandler(suite.svcCtx)

	cluster := suite.createTestCluster(t, "test-cluster", "active")

	r := gin.New()
	r.GET("/clusters/:id", h.GetClusterDetail)

	req := httptest.NewRequest(http.MethodGet, "/clusters/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	data, _ := parseClusterResponse(w.Body.Bytes())
	if data["name"] != "test-cluster" {
		t.Errorf("expected name 'test-cluster', got '%v'", data["name"])
	}
	_ = cluster // avoid unused warning
}

func TestGetClusterDetailHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterHandlerTestSuite(t)
	h := NewHandler(suite.svcCtx)

	r := gin.New()
	r.GET("/clusters/:id", h.GetClusterDetail)

	req := httptest.NewRequest(http.MethodGet, "/clusters/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Handler returns 200 with error code in body
	var resp struct {
		Code int `json:"code"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code == 1000 {
		t.Error("expected error code for not found")
	}
}

// ============================================================================
// GetClusterNodes Handler Tests
// ============================================================================

func TestGetClusterNodesHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterHandlerTestSuite(t)
	h := NewHandler(suite.svcCtx)

	cluster := suite.createTestCluster(t, "test-cluster", "active")
	suite.createTestClusterNode(t, cluster.ID, "node-1", "control-plane", "ready")
	suite.createTestClusterNode(t, cluster.ID, "node-2", "worker", "ready")

	r := gin.New()
	r.GET("/clusters/:id/nodes", h.GetClusterNodes)

	req := httptest.NewRequest(http.MethodGet, "/clusters/1/nodes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	data, _ := parseClusterResponse(w.Body.Bytes())
	list, ok := data["list"].([]any)
	if !ok {
		t.Fatalf("expected list in data, got: %v", data)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(list))
	}
}

// ============================================================================
// Repository Tests
// ============================================================================

func TestRepository_CreateCluster(t *testing.T) {
	suite := newClusterHandlerTestSuite(t)
	ctx := context.Background()

	repo := NewRepository(suite.db)

	cluster := &model.Cluster{
		Name:   "repo-test-cluster",
		Status: "active",
		Type:   "kubernetes",
	}

	err := repo.CreateCluster(ctx, cluster)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}

	if cluster.ID == 0 {
		t.Error("expected cluster ID to be set after creation")
	}
}

func TestRepository_GetClusterDetail(t *testing.T) {
	suite := newClusterHandlerTestSuite(t)
	ctx := context.Background()

	repo := NewRepository(suite.db)

	// Create first
	cluster := &model.Cluster{
		Name:   "get-test-cluster",
		Status: "active",
		Type:   "kubernetes",
	}
	repo.CreateCluster(ctx, cluster)

	// Get by ID
	found, err := repo.GetClusterDetail(ctx, cluster.ID)
	if err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}
	if found.Name != "get-test-cluster" {
		t.Errorf("expected name 'get-test-cluster', got '%s'", found.Name)
	}
}

func TestRepository_ListClusters(t *testing.T) {
	suite := newClusterHandlerTestSuite(t)
	ctx := context.Background()

	repo := NewRepository(suite.db)

	// Create clusters
	repo.CreateCluster(ctx, &model.Cluster{Name: "cluster-1", Status: "active", Type: "kubernetes"})
	repo.CreateCluster(ctx, &model.Cluster{Name: "cluster-2", Status: "inactive", Type: "kubernetes"})

	// List all
	list, err := repo.ListClusters(ctx, "", "")
	if err != nil {
		t.Fatalf("failed to list clusters: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 clusters, got %d", len(list))
	}

	// Filter by status
	activeList, err := repo.ListClusters(ctx, "active", "")
	if err != nil {
		t.Fatalf("failed to list active clusters: %v", err)
	}
	if len(activeList) != 1 {
		t.Errorf("expected 1 active cluster, got %d", len(activeList))
	}
}

func TestRepository_DeleteClusterWithRelations(t *testing.T) {
	suite := newClusterHandlerTestSuite(t)
	ctx := context.Background()

	repo := NewRepository(suite.db)

	// Create first
	cluster := &model.Cluster{
		Name:   "delete-test-cluster",
		Status: "active",
		Type:   "kubernetes",
	}
	repo.CreateCluster(ctx, cluster)

	// Create related nodes
	suite.createTestClusterNode(t, cluster.ID, "node-1", "worker", "ready")

	// Delete
	err := repo.DeleteClusterWithRelations(ctx, cluster.ID)
	if err != nil {
		t.Fatalf("failed to delete cluster: %v", err)
	}

	// Verify deleted
	var count int64
	suite.db.Model(&model.Cluster{}).Where("id = ?", cluster.ID).Count(&count)
	if count != 0 {
		t.Error("expected cluster to be deleted")
	}

	// Verify nodes deleted
	suite.db.Model(&model.ClusterNode{}).Where("cluster_id = ?", cluster.ID).Count(&count)
	if count != 0 {
		t.Error("expected cluster nodes to be deleted")
	}
}
