package agent

import (
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
)

type ToolCategory string

const (
	CategoryK8s     ToolCategory = "k8s"
	CategoryHost    ToolCategory = "host"
	CategoryService ToolCategory = "service"
	CategoryMonitor ToolCategory = "monitor"
	CategoryMCP     ToolCategory = "mcp"
	CategoryOther   ToolCategory = "other"
)

type ToolDefinition struct {
	Name         string
	Category     ToolCategory
	Description  string
	Risk         aitools.ToolRisk
	Mode         aitools.ToolMode
	Provider     string
	Permission   string
	Schema       map[string]any
	Required     []string
	RelatedTools []string
	Tool         tool.InvokableTool
}

type ToolRegistry struct {
	tools map[string]*ToolDefinition
}

func NewToolRegistry(registered []aitools.RegisteredTool) *ToolRegistry {
	registry := &ToolRegistry{tools: make(map[string]*ToolDefinition, len(registered))}
	for _, item := range registered {
		name := aitools.NormalizeToolName(item.Meta.Name)
		if name == "" {
			continue
		}
		meta := item.Meta
		registry.tools[name] = &ToolDefinition{
			Name:         name,
			Category:     classifyTool(meta),
			Description:  strings.TrimSpace(meta.Description),
			Risk:         meta.Risk,
			Mode:         meta.Mode,
			Provider:     strings.TrimSpace(meta.Provider),
			Permission:   strings.TrimSpace(meta.Permission),
			Schema:       cloneMap(meta.Schema),
			Required:     append([]string(nil), meta.Required...),
			RelatedTools: append([]string(nil), meta.RelatedTools...),
			Tool:         item.Tool,
		}
	}
	return registry
}

func (r *ToolRegistry) Get(name string) (*ToolDefinition, bool) {
	if r == nil {
		return nil, false
	}
	item, ok := r.tools[aitools.NormalizeToolName(name)]
	return item, ok
}

func (r *ToolRegistry) List() []*ToolDefinition {
	if r == nil {
		return nil
	}
	out := make([]*ToolDefinition, 0, len(r.tools))
	for _, item := range r.tools {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (r *ToolRegistry) ByCategory(category ToolCategory) []*ToolDefinition {
	if r == nil {
		return nil
	}
	var out []*ToolDefinition
	for _, item := range r.tools {
		if item.Category == category {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (r *ToolRegistry) BaseTools() []tool.BaseTool {
	if r == nil {
		return nil
	}
	items := r.List()
	out := make([]tool.BaseTool, 0, len(items))
	for _, item := range items {
		meta := aitools.ToolMeta{
			Name:        item.Name,
			Description: item.Description,
			Mode:        item.Mode,
			Risk:        item.Risk,
			Provider:    item.Provider,
			Permission:  item.Permission,
			Schema:      cloneMap(item.Schema),
			Required:    append([]string(nil), item.Required...),
		}
		out = append(out, aitools.WrapRegisteredTool(aitools.RegisteredTool{
			Meta: meta,
			Tool: item.Tool,
		}))
	}
	return out
}

func (r *ToolRegistry) ToolMap() map[string]tool.InvokableTool {
	if r == nil {
		return nil
	}
	out := make(map[string]tool.InvokableTool, len(r.tools))
	for name, item := range r.tools {
		out[name] = item.Tool
	}
	return out
}

func (r *ToolRegistry) MetaMap() map[string]aitools.ToolMeta {
	if r == nil {
		return nil
	}
	out := make(map[string]aitools.ToolMeta, len(r.tools))
	for name, item := range r.tools {
		out[name] = aitools.ToolMeta{
			Name:         item.Name,
			Description:  item.Description,
			Mode:         item.Mode,
			Risk:         item.Risk,
			Provider:     item.Provider,
			Permission:   item.Permission,
			Schema:       cloneMap(item.Schema),
			Required:     append([]string(nil), item.Required...),
			RelatedTools: append([]string(nil), item.RelatedTools...),
		}
	}
	return out
}

func classifyTool(meta aitools.ToolMeta) ToolCategory {
	name := aitools.NormalizeToolName(meta.Name)
	switch {
	case strings.EqualFold(strings.TrimSpace(meta.Provider), "mcp"), strings.HasPrefix(name, "mcp."):
		return CategoryMCP
	case strings.HasPrefix(name, "k8s_"), strings.HasPrefix(name, "cluster_"):
		return CategoryK8s
	case strings.HasPrefix(name, "host_"), strings.HasPrefix(name, "os_"):
		return CategoryHost
	case strings.HasPrefix(name, "service_"), strings.HasPrefix(name, "deployment_"), strings.HasPrefix(name, "config_"), strings.HasPrefix(name, "cicd_"), strings.HasPrefix(name, "job_"):
		return CategoryService
	case strings.HasPrefix(name, "monitor_"):
		return CategoryMonitor
	default:
		return CategoryOther
	}
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
