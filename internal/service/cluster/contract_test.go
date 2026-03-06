package cluster

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/k8s-manage/internal/cache"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/testutil"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// clusterContractTestSuite provides test infrastructure for cluster API contract tests.
type clusterContractTestSuite struct {
	db     *gorm.DB
	svcCtx *svc.ServiceContext
}

func newClusterContractTestSuite(t *testing.T) *clusterContractTestSuite {
	t.Helper()
	dbName := "cluster_contract_" + t.Name()
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

	// Create cache facade
	l1 := expirable.NewLRU[string, string](100, nil, time.Hour)
	cacheFacade := cache.NewFacade(l1, nil)

	return &clusterContractTestSuite{
		db: db,
		svcCtx: &svc.ServiceContext{
			DB:          db,
			CacheFacade: cacheFacade,
		},
	}
}

func (s *clusterContractTestSuite) createTestCluster(t *testing.T, name, status string) *model.Cluster {
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

func (s *clusterContractTestSuite) createTestClusterNode(t *testing.T, clusterID uint, name, role, status string) *model.ClusterNode {
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

// ============================================================================
// GetClusters API Contract Tests
// ============================================================================

func TestGetClusters_Contract_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	// Create test clusters
	suite.createTestCluster(t, "cluster-1", "active")
	suite.createTestCluster(t, "cluster-2", "inactive")

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters", h.GetClusters)

	req := httptest.NewRequest(http.MethodGet, "/clusters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Contract assertions
	testutil.AssertContract(t, w.Body.Bytes()).
		IsSuccess().
		HasCode(xcode.Success).
		HasData()

	// Verify list structure
	testutil.AssertContract(t, w.Body.Bytes()).
		DataHasList("list", 2)

	// Verify total field exists
	data := testutil.AssertContract(t, w.Body.Bytes()).GetResponse().Data.(map[string]any)
	if _, ok := data["total"]; !ok {
		t.Error("expected 'total' field in clusters response")
	}
}

func TestGetClusters_Contract_EmptyList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters", h.GetClusters)

	req := httptest.NewRequest(http.MethodGet, "/clusters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.AssertContract(t, w.Body.Bytes()).
		IsSuccess().
		HasData()

	testutil.AssertContract(t, w.Body.Bytes()).
		DataHasList("list", 0)

	data := testutil.AssertContract(t, w.Body.Bytes()).GetResponse().Data.(map[string]any)
	if data["total"].(float64) != 0 {
		t.Errorf("expected total 0, got %v", data["total"])
	}
}

func TestGetClusters_Contract_FilterByStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	suite.createTestCluster(t, "active-cluster", "active")
	suite.createTestCluster(t, "inactive-cluster", "inactive")

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters", h.GetClusters)

	req := httptest.NewRequest(http.MethodGet, "/clusters?status=active", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.AssertContract(t, w.Body.Bytes()).
		IsSuccess()

	// Should only return active clusters
	testutil.AssertContract(t, w.Body.Bytes()).
		DataHasList("list", 1)
}

// ============================================================================
// GetClusterDetail API Contract Tests
// ============================================================================

func TestGetClusterDetail_Contract_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	cluster := suite.createTestCluster(t, "detail-cluster", "active")

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters/:id", h.GetClusterDetail)

	req := httptest.NewRequest(http.MethodGet, "/clusters/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Contract assertions
	testutil.AssertContract(t, w.Body.Bytes()).
		IsSuccess().
		HasData()

	// Verify data is a map (not list)
	data := testutil.AssertContract(t, w.Body.Bytes()).GetResponse().Data.(map[string]any)

	// Verify required fields
	requiredFields := []string{"id", "name", "status"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			t.Errorf("expected field %q in cluster detail", field)
		}
	}

	if data["id"].(float64) != float64(cluster.ID) {
		t.Errorf("expected id %d, got %v", cluster.ID, data["id"])
	}
}

func TestGetClusterDetail_Contract_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters/:id", h.GetClusterDetail)

	req := httptest.NewRequest(http.MethodGet, "/clusters/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.AssertContract(t, w.Body.Bytes()).
		HasCode(xcode.NotFound).
		IsError()
}

// ============================================================================
// GetClusterNodes API Contract Tests
// ============================================================================

func TestGetClusterNodes_Contract_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	cluster := suite.createTestCluster(t, "nodes-cluster", "active")
	suite.createTestClusterNode(t, cluster.ID, "node-1", "control-plane", "ready")
	suite.createTestClusterNode(t, cluster.ID, "node-2", "worker", "ready")

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters/:id/nodes", h.GetClusterNodes)

	req := httptest.NewRequest(http.MethodGet, "/clusters/1/nodes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.AssertContract(t, w.Body.Bytes()).
		IsSuccess().
		HasData()

	testutil.AssertContract(t, w.Body.Bytes()).
		DataHasList("list", 2)
}

func TestGetClusterNodes_Contract_EmptyCluster(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	suite.createTestCluster(t, "empty-cluster", "active")

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters/:id/nodes", h.GetClusterNodes)

	req := httptest.NewRequest(http.MethodGet, "/clusters/1/nodes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.AssertContract(t, w.Body.Bytes()).
		IsSuccess()

	testutil.AssertContract(t, w.Body.Bytes()).
		DataHasList("list", 0)
}

// ============================================================================
// Response Format Verification Tests
// ============================================================================

func TestClusterAPI_ResponseFormat_StandardFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	suite.createTestCluster(t, "format-cluster", "active")

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters", h.GetClusters)

	req := httptest.NewRequest(http.MethodGet, "/clusters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Verify standard response fields
	requiredFields := []string{"code", "msg", "data"}
	for _, field := range requiredFields {
		if _, exists := resp[field]; !exists {
			t.Errorf("required field %q missing in response", field)
		}
	}

	// Verify code is success
	if resp["code"].(float64) != float64(xcode.Success) {
		t.Errorf("expected success code %d, got %v", xcode.Success, resp["code"])
	}
}

func TestClusterAPI_ErrorResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters/:id", h.GetClusterDetail)

	// Request non-existent cluster
	req := httptest.NewRequest(http.MethodGet, "/clusters/999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	// Error response should have code and msg
	if resp["code"] == nil {
		t.Error("error response missing code field")
	}
	if resp["msg"] == nil {
		t.Error("error response missing msg field")
	}

	// Code should be error code (>= 2000)
	code := int(resp["code"].(float64))
	if code < 2000 {
		t.Errorf("expected error code >= 2000, got %d", code)
	}
}

// ============================================================================
// Cluster Detail Response Structure Tests
// ============================================================================

func TestClusterDetail_ResponseStructure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	cluster := suite.createTestCluster(t, "structure-cluster", "active")

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters/:id", h.GetClusterDetail)

	req := httptest.NewRequest(http.MethodGet, "/clusters/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	data := testutil.AssertContract(t, w.Body.Bytes()).GetResponse().Data.(map[string]any)

	// Verify expected fields and types
	expectedFields := map[string]bool{
		"id":          true,
		"name":        true,
		"status":      true,
		"source":      true,
		"type":        true,
		"endpoint":    true,
		"created_at":  true,
		"updated_at":  true,
	}

	for field := range expectedFields {
		if _, exists := data[field]; !exists {
			t.Errorf("expected field %q in cluster detail response", field)
		}
	}

	// Verify id matches
	if data["id"].(float64) != float64(cluster.ID) {
		t.Errorf("expected id %d, got %v", cluster.ID, data["id"])
	}
}

// ============================================================================
// Pagination and List Response Tests
// ============================================================================

func TestClusterList_ContainsTotal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	suite := newClusterContractTestSuite(t)

	// Create multiple clusters
	for i := 0; i < 5; i++ {
		suite.createTestCluster(t, "cluster-"+string(rune('0'+i)), "active")
	}

	h := NewHandler(suite.svcCtx)
	r := gin.New()
	r.GET("/clusters", h.GetClusters)

	req := httptest.NewRequest(http.MethodGet, "/clusters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	data := testutil.AssertContract(t, w.Body.Bytes()).GetResponse().Data.(map[string]any)

	// Verify total matches list length
	list := data["list"].([]any)
	total := int(data["total"].(float64))

	if total != len(list) {
		t.Errorf("total (%d) does not match list length (%d)", total, len(list))
	}

	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
}
