package experts

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
)

func TestSceneMappingsRegressionCompatibility(t *testing.T) {
	cfg, err := LoadSceneMappings("configs/scene_mappings.yaml")
	if err != nil {
		t.Fatalf("load scene mappings: %v", err)
	}
	required := []string{
		"services:list",
		"services:detail",
		"services:deploy",
		"deployment:hosts",
		"governance:permissions",
	}
	for _, key := range required {
		item, ok := cfg.Mappings[key]
		if !ok {
			t.Fatalf("missing mapping: %s", key)
		}
		if item.PrimaryExpert == "" {
			t.Fatalf("mapping %s primary_expert is empty", key)
		}
	}
}

func BenchmarkHybridRouterRoute(b *testing.B) {
	reg, err := NewExpertRegistry(context.Background(), "configs/experts.yaml", buildTestToolsForBenchmark(), nil)
	if err != nil {
		b.Fatalf("new registry: %v", err)
	}
	router, err := NewHybridRouter(reg, "configs/scene_mappings.yaml")
	if err != nil {
		b.Fatalf("new router: %v", err)
	}
	req := &RouteRequest{
		Message: "请排查 k8s 集群 pod 异常和服务发布失败",
		Scene:   "scene:services:detail",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.Route(context.Background(), req)
	}
}

func buildTestToolsForBenchmark() map[string]tool.InvokableTool {
	return map[string]tool.InvokableTool{}
}
