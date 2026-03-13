package ai

import (
	"net/http/httptest"
	"strconv"
	"testing"

	coreai "github.com/cy77cc/OpsPilot/internal/ai"
	aitools "github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestHintResolverResolvesBuiltInSourcesWithProjectAndSelectionContext(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	project := model.Project{Name: "proj-a"}
	if err := suite.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	cluster := suite.SeedCluster(func(c *model.Cluster) {
		c.Name = "cluster-a"
	})
	service := suite.SeedService(func(s *model.Service) {
		s.Name = "deploy-a"
		s.ProjectID = project.ID
		s.RuntimeType = "k8s"
	})
	target := model.DeploymentTarget{
		Name:      "target-a",
		ClusterID: cluster.ID,
		ProjectID: project.ID,
		Env:       "staging",
		Status:    "active",
	}
	if err := suite.DB.Create(&target).Error; err != nil {
		t.Fatalf("create deployment target: %v", err)
	}
	if err := suite.DB.Create(&model.DeploymentRelease{
		ServiceID:          service.ID,
		TargetID:           target.ID,
		RuntimeType:        "k8s",
		NamespaceOrProject: "payments",
		Status:             "success",
	}).Error; err != nil {
		t.Fatalf("create deployment release: %v", err)
	}

	handler := NewHTTPHandler(suite.SvcCtx)
	spec := aitools.ToolSpec{
		Capability: aitools.Capability{
			Name: "k8s_logs",
			Schema: map[string]aitools.ParamHint{
				"cluster_id": {Type: "integer", EnumSource: "clusters"},
				"namespace":  {Type: "string", EnumSource: "namespaces"},
				"pod":        {Type: "string"},
			},
		},
		Input: kubernetes.K8sLogsInput{},
	}
	runtimeCtx := handler.normalizeRuntimeContext(testGinContext(t, project.ID), map[string]any{
		"selectedResources": []any{
			map[string]any{"type": "pod", "name": "pod-a", "namespace": "payments"},
		},
	})

	hints, err := handler.hintResolver.Resolve(t.Context(), spec, runtimeCtx, map[string]any{})
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}

	if got := hints["cluster_id"].EnumSource; got != "clusters" {
		t.Fatalf("cluster_id enum_source = %q", got)
	}
	if len(hints["cluster_id"].Options) != 1 || hints["cluster_id"].Options[0].Label != "cluster-a" {
		t.Fatalf("unexpected cluster options: %#v", hints["cluster_id"].Options)
	}
	if hints["namespace"].Default != "payments" {
		t.Fatalf("namespace default = %#v, want payments", hints["namespace"].Default)
	}
	if len(hints["pod"].Options) != 1 || hints["pod"].Options[0].Label != "pod-a" {
		t.Fatalf("unexpected pod options: %#v", hints["pod"].Options)
	}
}

func TestHintResolverCascadesDynamicNameSourceForK8sQuery(t *testing.T) {
	suite := testutil.NewIntegrationSuite(t)
	t.Cleanup(suite.Cleanup)

	project := model.Project{Name: "proj-b"}
	if err := suite.DB.Create(&project).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	_ = suite.SeedService(func(s *model.Service) {
		s.Name = "deploy-b"
		s.ProjectID = project.ID
		s.RuntimeType = "k8s"
	})

	handler := NewHTTPHandler(suite.SvcCtx)
	spec, ok := handler.registry.Get("k8s_query")
	if !ok {
		t.Fatal("missing k8s_query tool")
	}
	runtimeCtx := handler.normalizeRuntimeContext(testGinContext(t, project.ID), map[string]any{
		"selectedResources": []any{
			map[string]any{"type": "deployment", "name": "deploy-selected", "namespace": "payments"},
		},
	})

	hints, err := handler.hintResolver.Resolve(t.Context(), spec, runtimeCtx, map[string]any{
		"resource": "deployments",
	})
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}

	if got := hints["name"].EnumSource; got != "deployments" {
		t.Fatalf("name enum_source = %q, want deployments", got)
	}
	if got := hints["name"].Default; got != "deploy-selected" {
		t.Fatalf("name default = %#v, want deploy-selected", got)
	}
	if len(hints["name"].DependsOn) != 2 || hints["name"].DependsOn[1] != "namespace" {
		t.Fatalf("name depends_on = %#v", hints["name"].DependsOn)
	}
}

func testGinContext(t *testing.T, projectID uint) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/ai/tools/k8s_logs/params/hints", nil)
	c.Request.Header.Set("X-Project-ID", strconv.Itoa(int(projectID)))
	c.Set(ginRuntimeContextKey, coreai.RuntimeContext{
		Scene:       "global",
		Route:       "/api/v1/ai/tools/k8s_logs/params/hints",
		ProjectID:   strconv.Itoa(int(projectID)),
		UserContext: map[string]any{"uid": uint64(1)},
		Metadata:    map[string]any{},
	})
	c.Set("uid", uint64(1))
	return c
}
