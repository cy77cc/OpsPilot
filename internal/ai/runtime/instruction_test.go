package runtime

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildInstruction_Basic(t *testing.T) {
	ctx := RuntimeContext{
		SceneName:   "主机管理",
		ProjectName: "生产环境",
	}

	result := BuildInstruction(ctx)

	assert.Contains(t, result, "场景: 主机管理")
	assert.Contains(t, result, "项目: 生产环境")
	assert.Contains(t, result, "页面: 未指定")
	assert.Contains(t, result, "选中资源: 无")
	assert.Contains(t, result, "你是 OpsPilot 智能运维助手")
}

func TestBuildInstruction_WithSelectedResources(t *testing.T) {
	ctx := RuntimeContext{
		SceneName:   "Kubernetes",
		ProjectName: "测试集群",
		SelectedResources: []SelectedResource{
			{Type: "pod", Name: "nginx-123", Namespace: "default"},
			{Type: "service", Name: "my-service", Namespace: "default"},
		},
	}

	result := BuildInstruction(ctx)

	assert.Contains(t, result, "场景: Kubernetes")
	assert.Contains(t, result, "项目: 测试集群")
	assert.Contains(t, result, "页面: 未指定")
	assert.Contains(t, result, "nginx-123(pod)")
	assert.Contains(t, result, "my-service(service)")
}

func TestBuildInstruction_EmptyContext(t *testing.T) {
	result := BuildInstruction(RuntimeContext{})

	assert.Contains(t, result, "场景: 通用")
	assert.Contains(t, result, "项目: 未指定")
	assert.Contains(t, result, "页面: 未指定")
	assert.Contains(t, result, "选中资源: 无")
}

func TestBuildInstruction_FallbackToSceneProjectAndPageContext(t *testing.T) {
	result := BuildInstruction(RuntimeContext{
		Scene:       "deployment:hosts",
		Route:       "/api/v1/ai/chat",
		ProjectID:   "1",
		CurrentPage: "http://localhost:8080/deployment/infrastructure/hosts",
		Metadata: map[string]any{
			"scene": "deployment:hosts",
		},
	})

	assert.Contains(t, result, "场景: deployment:hosts")
	assert.Contains(t, result, "项目: 1")
	assert.Contains(t, result, "页面: http://localhost:8080/deployment/infrastructure/hosts")
}

func TestBuildInstruction_CompleteTemplate(t *testing.T) {
	result := BuildInstruction(RuntimeContext{
		SceneName:   "主机管理",
		ProjectName: "生产环境",
	})

	assert.NotContains(t, result, "{")
	assert.NotContains(t, result, "}")
	assert.True(t, strings.Contains(result, "核心能力"))
	assert.True(t, strings.Contains(result, "工作原则"))
	assert.True(t, strings.Contains(result, "当前上下文"))
}
