package middleware

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/sceneutil"
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

	// sceneToolMap 场景到工具的映射（O(1) 查找）
	sceneToolMap map[string]*sceneutil.AllowedToolSet

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

	// 将 []string 转换为 AllowedToolSet 以实现 O(1) 查找
	toolSetMap := make(map[string]*sceneutil.AllowedToolSet, len(cfg.SceneToolMap))
	for scene, names := range cfg.SceneToolMap {
		toolSetMap[scene] = sceneutil.NewAllowedToolSet(names)
	}

	// 初始化时尝试从上下文获取场景
	sceneMeta := runtimectx.AIMetadataFrom(ctx)
	currentScene := NormalizeScene(sceneMeta.Scene)

	return &SceneRouterMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		sceneToolMap:                 toolSetMap,
		currentScene:                 currentScene,
	}, nil
}

// WrapInvokableToolCall 拦截同步工具调用，过滤场景专属工具。
//
// 该方法会检查工具是否在当前场景允许列表中：
//   - 如果允许，直接调用原始端点
//   - 如果不允许，返回错误信息
func (m *SceneRouterMiddleware) WrapInvokableToolCall(
	ctx context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx == nil {
		return endpoint, nil
	}

	scene := m.resolveScene(ctx)

	// 检查工具是否在当前场景允许列表中
	allowedTools := m.sceneToolMap[scene]
	if !allowedTools.IsAllowed(tCtx.Name) {
		return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
			return "", fmt.Errorf("tool '%s' is not available in scene '%s'", tCtx.Name, scene)
		}, nil
	}

	// 工具可用，直接调用
	return endpoint, nil
}

// WrapStreamableToolCall 拦截流式工具调用，过滤场景专属工具。
//
// 与 WrapInvokableToolCall 类似，但处理流式输出。
func (m *SceneRouterMiddleware) WrapStreamableToolCall(
	ctx context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	if tCtx == nil {
		return endpoint, nil
	}

	scene := m.resolveScene(ctx)
	allowedTools := m.sceneToolMap[scene]
	if !allowedTools.IsAllowed(tCtx.Name) {
		return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
			return nil, fmt.Errorf("tool '%s' is not available in scene '%s'", tCtx.Name, scene)
		}, nil
	}

	return endpoint, nil
}

func (m *SceneRouterMiddleware) resolveScene(ctx context.Context) string {
	if scene := NormalizeScene(runtimectx.AIMetadataFrom(ctx).Scene); strings.TrimSpace(scene) != "" {
		return scene
	}
	if scene := NormalizeScene(m.currentScene); strings.TrimSpace(scene) != "" {
		return scene
	}
	return "ai"
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

func DefaultSceneToolMap() map[string][]string {
	return map[string][]string{
		"kubernetes":     {"k8s_query", "k8s_list_resources", "service_get_detail", "service_status", "host_list_inventory"},
		"cicd":           {"cicd_pipeline_list", "cicd_pipeline_status", "cicd_pipeline_trigger", "job_list", "job_execution_status", "job_run", "deployment_target_list", "deployment_target_detail", "deployment_bootstrap_status"},
		"monitoring":     {"monitor_alert_rule_list", "monitor_alert", "monitor_metric"},
		"host":           {"host_exec", "host_list_inventory"},
		"cluster":        {"k8s_query", "k8s_list_resources", "deployment_bootstrap_status", "cluster_list_inventory", "service_list_inventory", "service_get_detail", "service_status"},
		"deployment":     {"deployment_target_list", "deployment_target_detail", "deployment_bootstrap_status", "cluster_list_inventory", "service_list_inventory", "service_get_detail", "service_status", "service_deploy_preview", "service_deploy_apply"},
		"service":        {"service_get_detail", "service_status", "service_status_by_target", "service_catalog_list", "service_category_tree", "service_visibility_check", "service_deploy_preview"},
		"infrastructure": {"credential_list", "credential_test", "host_list_inventory", "cluster_list_inventory"},
		"governance":     {"user_list", "role_list", "permission_check", "topology_get", "audit_log_search"},
		"ai":             {"service_get_detail", "service_status", "service_catalog_list", "host_list_inventory", "monitor_alert", "monitor_metric", "k8s_query", "k8s_list_resources"},
	}
}

func DefaultScenePromptMap() map[string]string {
	return map[string]string{
		"kubernetes":     "Kubernetes operations and diagnosis.",
		"cicd":           "CI/CD pipeline administration and status checks.",
		"monitoring":     "Monitoring and alert investigation.",
		"host":           "Host operations with guarded execution.",
		"cluster":        "Cluster inventory and deployment visibility.",
		"deployment":     "Deployment planning and rollout controls.",
		"service":        "Service status and release context.",
		"infrastructure": "Infrastructure inventory and credential health.",
		"governance":     "Governance, audit, and topology review.",
		"ai":             "General AI assistant scene with cross-domain diagnostics.",
	}
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
