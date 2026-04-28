package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTool 创建模拟工具用于测试。
func mockTool(name string) tool.BaseTool {
	t, _ := einoutils.InferTool(
		name,
		"Mock tool for testing",
		func(ctx context.Context, input map[string]any) (string, error) {
			return "mock result", nil
		},
	)
	return t
}

func TestSceneRouterMiddleware_FiltersToolsByScene(t *testing.T) {
	tests := []struct {
		name        string
		scene       string
		toolName    string
		shouldAllow bool
	}{
		{
			name:        "kubernetes scene allows k8s_query",
			scene:       "kubernetes",
			toolName:    "k8s_query",
			shouldAllow: true,
		},
		{
			name:        "kubernetes scene blocks cicd_pipeline_trigger",
			scene:       "kubernetes",
			toolName:    "cicd_pipeline_trigger",
			shouldAllow: false,
		},
		{
			name:        "cicd scene allows cicd_pipeline_list",
			scene:       "cicd",
			toolName:    "cicd_pipeline_list",
			shouldAllow: true,
		},
		{
			name:        "cicd scene blocks k8s_query",
			scene:       "cicd",
			toolName:    "k8s_query",
			shouldAllow: false,
		},
		{
			name:        "default scene allows service_get_detail",
			scene:       "ai",
			toolName:    "service_get_detail",
			shouldAllow: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建场景路由器
			router, err := NewSceneRouter(context.Background(), &SceneRouterConfig{
				SceneToolMap: DefaultSceneToolMap(),
			})
			require.NoError(t, err)

			callCtx := runtimectx.WithAIMetadata(context.Background(), runtimectx.AIMetadata{Scene: tt.scene})

			// 创建模拟工具上下文
			tCtx := &adk.ToolContext{
				Name: tt.toolName,
			}

			// 测试工具调用拦截
			wrapped, err := router.WrapInvokableToolCall(
				callCtx,
				func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
					return "executed", nil
				},
				tCtx,
			)
			require.NoError(t, err)

			// 执行工具调用
			result, err := wrapped(callCtx, "{}")
			if tt.shouldAllow {
				assert.NoError(t, err)
				assert.Equal(t, "executed", result)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not available in scene")
			}
		})
	}
}

func TestNormalizeScene_NormalizesCorrectly(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"kubernetes", "kubernetes"},
		{"KUBERNETES", "kubernetes"},
		{"  CICD  ", "cicd"},
		{"", "ai"},
		{"   ", "ai"},
		{"Monitoring", "monitoring"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeScene(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultSceneToolMap_ContainsExpectedTools(t *testing.T) {
	sceneMap := DefaultSceneToolMap()

	// 测试 kubernetes 场景
	k8sTools := sceneMap["kubernetes"]
	assert.Contains(t, k8sTools, "k8s_query")
	assert.Contains(t, k8sTools, "k8s_list_resources")
	assert.NotContains(t, k8sTools, "cicd_pipeline_trigger")

	//测试 cicd 场景
	cicdTools := sceneMap["cicd"]
	assert.Contains(t, cicdTools, "cicd_pipeline_list")
	assert.Contains(t, cicdTools, "cicd_pipeline_trigger")
	assert.NotContains(t, cicdTools, "k8s_query")

	// 测试默认场景
	aiTools := sceneMap["ai"]
	assert.NotEmpty(t, aiTools, "default scene should have tools")
}

func TestSceneRouterMiddleware_ReadsSceneFromRuntime(t *testing.T) {
	// 创建带有场景元数据的上下文
	ctx := runtimectx.WithAIMetadata(
		context.Background(),
		runtimectx.AIMetadata{
			Scene: "kubernetes",
		},
	)

	// 创建场景路由器
	router, err := NewSceneRouter(ctx, &SceneRouterConfig{
		SceneToolMap: DefaultSceneToolMap(),
	})
	require.NoError(t, err)

	// 验证路由器从上下文读取了正确的场景
	assert.Equal(t, "kubernetes", router.currentScene)
}

func TestSceneRouterMiddleware_UsesPerRequestRuntimeScene(t *testing.T) {
	router, err := NewSceneRouter(context.Background(), &SceneRouterConfig{
		SceneToolMap: DefaultSceneToolMap(),
	})
	require.NoError(t, err)
	router.currentScene = "ai"

	hostCtx := runtimectx.WithAIMetadata(context.Background(), runtimectx.AIMetadata{Scene: "host"})
	wrapped, err := router.WrapInvokableToolCall(
		hostCtx,
		func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
			return "executed", nil
		},
		&adk.ToolContext{Name: "host_list_inventory"},
	)
	require.NoError(t, err)
	out, callErr := wrapped(hostCtx, "{}")
	assert.NoError(t, callErr)
	assert.Equal(t, "executed", out)
}

func TestDefaultScenePromptMap_ContainsAllScenes(t *testing.T) {
	promptMap := DefaultScenePromptMap()
	sceneMap := DefaultSceneToolMap()

	// 验证每个场景都有对应的 Prompt
	for scene := range sceneMap {
		prompt, exists := promptMap[scene]
		assert.True(t, exists, "scene %s should have a prompt", scene)
		assert.NotEmpty(t, prompt, "prompt for scene %s should not be empty", scene)
	}
}
