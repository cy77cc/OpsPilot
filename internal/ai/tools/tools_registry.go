package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/cicd"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/deployment"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/governance"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/host"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/infrastructure"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/kubernetes"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/monitor"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/impl/service"
)

// Registry 提供工具的快速查找能力。
type Registry struct {
	byName     map[string]RegisteredTool
	byDomain   map[ToolDomain][]RegisteredTool
	byCategory map[ToolCategory][]RegisteredTool
}

// NewRegistry 从工具列表创建注册表。
func NewRegistry(registered []RegisteredTool) *Registry {
	r := &Registry{
		byName:     make(map[string]RegisteredTool, len(registered)),
		byDomain:   make(map[ToolDomain][]RegisteredTool),
		byCategory: make(map[ToolCategory][]RegisteredTool),
	}
	for _, item := range registered {
		item.Meta = normalizeToolMeta(item.Meta)
		name := NormalizeToolName(item.Meta.Name)
		if name == "" {
			continue
		}
		r.byName[name] = item
		r.byDomain[item.Meta.Domain] = append(r.byDomain[item.Meta.Domain], item)
		r.byCategory[item.Meta.Category] = append(r.byCategory[item.Meta.Category], item)
	}
	return r
}

// Get 按名称获取工具。
func (r *Registry) Get(name string) (RegisteredTool, bool) {
	if r == nil {
		return RegisteredTool{}, false
	}
	item, ok := r.byName[NormalizeToolName(name)]
	return item, ok
}

// ByDomain 按领域获取工具列表。
func (r *Registry) ByDomain(domain ToolDomain) []RegisteredTool {
	if r == nil {
		return nil
	}
	items := r.byDomain[domain]
	return append([]RegisteredTool(nil), items...)
}

// ByCategory 按类别获取工具列表。
func (r *Registry) ByCategory(category ToolCategory) []RegisteredTool {
	if r == nil {
		return nil
	}
	items := r.byCategory[category]
	return append([]RegisteredTool(nil), items...)
}

// BuildAllTools 返回所有工具（带风险包装）。
func BuildAllTools(ctx context.Context, deps PlatformDeps) ([]tool.BaseTool, error) {
	registered, err := BuildRegisteredTools(deps)
	if err != nil {
		return nil, err
	}
	all := make([]tool.BaseTool, 0, len(registered))
	for _, item := range registered {
		all = append(all, WrapRegisteredTool(item))
	}
	return all, nil
}

// BuildRegisteredTools 聚合所有本地工具。
func BuildRegisteredTools(deps PlatformDeps) ([]RegisteredTool, error) {
	return BuildLocalTools(deps)
}

// BuildRegisteredToolsWithMCP 聚合本地工具和 MCP 代理工具。
func BuildRegisteredToolsWithMCP(deps PlatformDeps, manager *MCPClientManager) ([]RegisteredTool, error) {
	localTools, err := BuildLocalTools(deps)
	if err != nil {
		return nil, err
	}
	mcpTools, err := BuildMCPProxyTools(manager)
	if err != nil {
		return nil, err
	}
	return append(localTools, mcpTools...), nil
}

// BuildLocalTools 聚合所有 impl 包的工具。
func BuildLocalTools(deps PlatformDeps) ([]RegisteredTool, error) {
	ctx := context.Background()
	var tools []RegisteredTool

	// 聚合各领域的工具
	tools = append(tools, host.Register(ctx, deps)...)
	tools = append(tools, kubernetes.Register(ctx, deps)...)
	tools = append(tools, service.Register(ctx, deps)...)
	tools = append(tools, monitor.Register(ctx, deps)...)
	tools = append(tools, cicd.Register(ctx, deps)...)
	tools = append(tools, deployment.Register(ctx, deps)...)
	tools = append(tools, governance.Register(ctx, deps)...)
	tools = append(tools, infrastructure.Register(ctx, deps)...)

	return tools, nil
}

// WrapRegisteredTool 根据风险级别包装工具。
func WrapRegisteredTool(item RegisteredTool) tool.BaseTool {
	switch item.Meta.Risk {
	case ToolRiskHigh:
		return NewApprovableTool(item.Tool, ToolRiskHigh, buildDefaultPreview(item.Meta.Name))
	case ToolRiskMedium:
		return NewReviewableTool(item.Tool)
	default:
		return item.Tool
	}
}

func buildDefaultPreview(toolName string) ApprovalPreviewFn {
	return func(_ context.Context, args string) (map[string]any, error) {
		preview := map[string]any{"tool": toolName}
		if args == "" {
			preview["arguments"] = map[string]any{}
			return preview, nil
		}
		preview["arguments"] = args
		return preview, nil
	}
}

func normalizeToolMeta(meta ToolMeta) ToolMeta {
	if meta.Domain == "" {
		meta.Domain = classifyToolDomain(meta.Name)
	}
	if meta.Category == "" {
		meta.Category = classifyToolCategory(meta)
	}
	return meta
}
