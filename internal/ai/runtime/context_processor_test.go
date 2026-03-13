package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

func TestContextProcessorBuildPlannerInputIncludesSceneContext(t *testing.T) {
	resolver := NewSceneConfigResolver(nil)
	processor := NewContextProcessor(resolver)
	ctx := ContextWithRuntimeContext(context.Background(), RuntimeContext{
		Scene:       "deployment",
		SceneName:   "部署管理",
		Route:       "/deployments",
		ProjectID:   "proj-1",
		CurrentPage: "https://example.test/deployments",
		SelectedResources: []SelectedResource{
			{Type: "deployment", Name: "nginx", Namespace: "default"},
		},
	})

	msgs, err := processor.BuildPlannerInput(ctx, []adk.Message{{Content: "scale nginx to 3"}})
	if err != nil {
		t.Fatalf("BuildPlannerInput error = %v", err)
	}
	joined := msgs[0].Content + "\n" + msgs[1].Content
	for _, fragment := range []string{"部署管理", "nginx", "scale nginx to 3"} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("planner input missing %q: %s", fragment, joined)
		}
	}
}

func TestContextProcessorFilterToolsHonorsAllowList(t *testing.T) {
	resolver := NewSceneConfigResolver(nil)
	processor := NewContextProcessor(resolver)
	ctx := context.Background()
	scene := resolver.Resolve("deployment")
	clusterTool, err := toolutils.InferTool("cluster_list_inventory", "cluster list", func(context.Context, *struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	})
	if err != nil {
		t.Fatalf("InferTool(cluster) error = %v", err)
	}
	hostTool, err := toolutils.InferTool("host_list_inventory", "host list", func(context.Context, *struct{}) (*struct{}, error) {
		return &struct{}{}, nil
	})
	if err != nil {
		t.Fatalf("InferTool(host) error = %v", err)
	}
	tools := []tool.BaseTool{
		clusterTool,
		hostTool,
	}

	filtered := processor.FilterTools(ctx, scene, tools)
	if len(filtered) != 1 || filtered[0] != "cluster_list_inventory" {
		t.Fatalf("filtered tools = %#v", filtered)
	}
}

func TestContextProcessorBuildExecutorInputIncludesExecutedSteps(t *testing.T) {
	resolver := NewSceneConfigResolver(nil)
	processor := NewContextProcessor(resolver)
	ctx := ContextWithRuntimeContext(context.Background(), RuntimeContext{Scene: "monitor", SceneName: "监控中心"})

	plan := &testPlan{steps: []string{"check alerts"}}
	in := &planexecute.ExecutionContext{
		UserInput: []adk.Message{{Content: "investigate service health"}},
		Plan:      plan,
		ExecutedSteps: []planexecute.ExecutedStep{
			{Step: "collect metrics", Result: "latency spike"},
		},
	}

	msgs, err := processor.BuildExecutorInput(ctx, in, nil)
	if err != nil {
		t.Fatalf("BuildExecutorInput error = %v", err)
	}
	joined := msgs[0].Content + "\n" + msgs[1].Content
	for _, fragment := range []string{"监控中心", "collect metrics", "latency spike", "check alerts"} {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("executor input missing %q: %s", fragment, joined)
		}
	}
}

type testPlan struct {
	steps []string
}

func (p *testPlan) MarshalJSON() ([]byte, error) {
	return []byte(`{"steps":["` + p.steps[0] + `"]}`), nil
}

func (p *testPlan) UnmarshalJSON([]byte) error { return nil }

func (p *testPlan) FirstStep() string {
	if len(p.steps) == 0 {
		return ""
	}
	return p.steps[0]
}
