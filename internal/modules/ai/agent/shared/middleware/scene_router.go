package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

// SceneRouterMiddleware 根据场景路由工具调用。
//
// 职责：
//  1. 根据 scene 过滤可用工具集
//  2. 拦截非授权工具调用
//
// 该中间件是洋葱模型的第一层，在所有其他中间件之前执行，
// 确保后续中间件和工具调用都在正确的场景边界内。
type SceneRouterMiddleware struct {
	*adk.BaseChatModelAgentMiddleware

	// sceneToolMap 场景到工具的映射
	sceneToolMap map[string][]string

	// currentScene 当前场景（运行时动态设置）
	currentScene string
}

// SceneRouterConfig 场景路由器配置。
type SceneRouterConfig struct {
	// SceneToolMap 场景到工具名称的映射
	// 如果为 nil，则使用 DefaultSceneToolMap
	SceneToolMap map[string][]string
}

// NewSceneRouter 创建场景路由器中间件。
func NewSceneRouter(ctx context.Context, cfg *SceneRouterConfig) (*SceneRouterMiddleware, error) {
	if cfg == nil {
		cfg = &SceneRouterConfig{}
	}
	if cfg.SceneToolMap == nil {
		cfg.SceneToolMap = DefaultSceneToolMap()
	}

	// 初始化时尝试从上下文获取场景
	sceneMeta := runtimectx.AIMetadataFrom(ctx)
	currentScene := NormalizeScene(sceneMeta.Scene)

	return &SceneRouterMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		sceneToolMap:                 cfg.SceneToolMap,
		currentScene:                 currentScene,
	}, nil
}

// WrapInvokableToolCall 拦截同步工具调用，过滤场景专属工具。
//
// 该方法会检查工具是否在当前场景允许列表中：
//  - 如果允许，直接调用原始端点
//  - 如果不允许，返回错误信息
func (m *SceneRouterMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx == nil {
		return endpoint, nil
	}

	// 如果场景未设置，尝试从上下文获取
	if m.currentScene == "" {
		sceneMeta := runtimectx.AIMetadataFrom(context.Background())
		m.currentScene = NormalizeScene(sceneMeta.Scene)
	}

	// 检查工具是否在当前场景允许列表中
	allowedTools := m.sceneToolMap[m.currentScene]
	if !isToolAllowed(tCtx.Name, allowedTools) {
		return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
			return "", fmt.Errorf("tool '%s' is not available in scene '%s'", tCtx.Name, m.currentScene)
		}, nil
	}

	// 工具可用，直接调用
	return endpoint, nil
}

// WrapStreamableToolCall 拦截流式工具调用，过滤场景专属工具。
//
// 与 WrapInvokableToolCall 类似，但处理流式输出。
func (m *SceneRouterMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	if tCtx == nil {
		return endpoint, nil
	}

	// 如果场景未设置，尝试从上下文获取
	if m.currentScene == "" {
		sceneMeta := runtimectx.AIMetadataFrom(context.Background())
		m.currentScene = NormalizeScene(sceneMeta.Scene)
	}

	allowedTools := m.sceneToolMap[m.currentScene]
	if !isToolAllowed(tCtx.Name, allowedTools) {
		return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
			return nil, fmt.Errorf("tool '%s' is not available in scene '%s'", tCtx.Name, m.currentScene)
		}, nil
	}

	return endpoint, nil
}

// isToolAllowed 检查工具是否在允许列表中。
func isToolAllowed(toolName string, allowedTools []string) bool {
	for _, name := range allowedTools {
		if name == toolName {
			return true
		}
	}
	return false
}

// NormalizeScene 规范化场景名称。
//
// 将场景名称转换为小写，去除前后空格。
// 如果场景名称为空，返回默认场景 "ai"。
func NormalizeScene(scene string) string {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return "ai"
	}
	return strings.ToLower(scene)
}

// prependSystemMessage 在消息列表前插入系统消息。
//
// 如果 content 为空，直接返回原始消息列表。
func prependSystemMessage(messages []*schema.Message, content string) []*schema.Message {
	if content == "" {
		return messages
	}
	sysMsg := schema.SystemMessage(content)
	return append([]*schema.Message{sysMsg}, messages...)
}
