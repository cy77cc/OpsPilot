package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/experts"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
)

const toolCallGuide = `Tool calling rules:
1) NEVER call a tool with empty {} arguments when the tool has required fields.
2) Prefer using runtime context (scene/page/host_id/cluster_id/service_id/namespace) as arguments.
3) If any required field is missing, ask for it or choose a safe readonly tool first.
4) For mutating tools, require approval token before execution.
5) For inventory/list/assets/清单/已添加资源 requests, call inventory tools first (host_list_inventory/cluster_list_inventory/service_list_inventory).
6) Return concise explanation after each tool result.`

type PlatformAgent struct {
	Runnable     *react.Agent
	Model        model.ToolCallingChatModel
	registry     experts.ExpertRegistry
	router       *experts.HybridRouter
	orchestrator *experts.Orchestrator
	tools        map[string]tool.InvokableTool
	metas        map[string]tools.ToolMeta
	mcp          *tools.MCPClientManager
}

var scenePattern = regexp.MustCompile(`scene:([a-z0-9:_-]+)`)

func NewPlatformAgent(ctx context.Context, chatModel model.ToolCallingChatModel, deps tools.PlatformDeps) (*PlatformAgent, error) {
	if chatModel == nil {
		return nil, nil
	}

	localTools, err := tools.BuildLocalTools(deps)
	if err != nil {
		return nil, err
	}
	mcpManager, err := tools.NewMCPClientManager(ctx, tools.MCPConfigFromEnv())
	if err != nil {
		return nil, err
	}
	mcpTools, err := tools.BuildMCPProxyTools(mcpManager)
	if err != nil {
		return nil, err
	}
	registered := append(localTools, mcpTools...)
	baseTools := make([]tool.BaseTool, 0, len(registered))
	toolMap := make(map[string]tool.InvokableTool, len(registered))
	metaMap := make(map[string]tools.ToolMeta, len(registered))
	for _, item := range registered {
		baseTools = append(baseTools, item.Tool)
		toolMap[item.Meta.Name] = item.Tool
		metaMap[item.Meta.Name] = item.Meta
	}

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: baseTools},
		MaxStep:          20,
		MessageModifier:  react.NewPersonaModifier("You are Platform Ops Agent. Use tools safely and with complete parameters.\n" + toolCallGuide),
	})
	if err != nil {
		return nil, err
	}

	registry, err := experts.NewExpertRegistry(ctx, "configs/experts.yaml", toolMap, chatModel)
	if err != nil {
		return nil, err
	}
	router, err := experts.NewHybridRouter(registry, "configs/scene_mappings.yaml")
	if err != nil {
		return nil, err
	}
	orchestrator := experts.NewOrchestrator(registry, experts.NewResultAggregator(experts.AggregationTemplate, chatModel))

	return &PlatformAgent{
		Runnable:     agent,
		Model:        chatModel,
		registry:     registry,
		router:       router,
		orchestrator: orchestrator,
		tools:        toolMap,
		metas:        metaMap,
		mcp:          mcpManager,
	}, nil
}

func (p *PlatformAgent) ToolMetas() []tools.ToolMeta {
	if p == nil {
		return nil
	}
	out := make([]tools.ToolMeta, 0, len(p.metas))
	for _, m := range p.metas {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (p *PlatformAgent) Stream(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	if p == nil {
		return nil, fmt.Errorf("agent not initialized")
	}
	req := p.buildExecuteRequest(ctx, messages)
	if req == nil || req.Decision == nil {
		return p.Runnable.Stream(ctx, messages)
	}
	if req.Decision.Strategy == experts.StrategySingle {
		if exp, ok := p.registry.GetExpert(req.Decision.PrimaryExpert); ok && exp != nil && exp.Agent != nil {
			return exp.Agent.Stream(ctx, messages)
		}
		return p.Runnable.Stream(ctx, messages)
	}
	if p.orchestrator != nil {
		stream, err := p.orchestrator.StreamExecute(ctx, req)
		if err == nil && stream != nil {
			return stream, nil
		}
	}
	return p.Runnable.Stream(ctx, messages)
}

func (p *PlatformAgent) Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	if p == nil {
		return nil, fmt.Errorf("agent not initialized")
	}
	req := p.buildExecuteRequest(ctx, messages)
	if req == nil || req.Decision == nil || p.orchestrator == nil {
		return p.Runnable.Generate(ctx, messages)
	}
	result, err := p.orchestrator.Execute(ctx, req)
	if err != nil {
		return nil, err
	}
	return schema.AssistantMessage(result.Response, nil), nil
}

func sceneFromMessages(content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	matches := scenePattern.FindStringSubmatch(strings.ToLower(content))
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func (p *PlatformAgent) buildExecuteRequest(ctx context.Context, messages []*schema.Message) *experts.ExecuteRequest {
	if p == nil || p.router == nil {
		return nil
	}
	content := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] != nil && messages[i].Role == schema.User {
			content = strings.TrimSpace(messages[i].Content)
			break
		}
	}
	scene := sceneFromMessages(content)
	decision := p.router.Route(ctx, &experts.RouteRequest{
		Message: content,
		Scene:   scene,
		History: messages,
		RuntimeContext: map[string]any{
			"scene": scene,
		},
	})
	return &experts.ExecuteRequest{
		Message:  content,
		Decision: decision,
		RuntimeContext: map[string]any{
			"scene": scene,
		},
		History: messages,
	}
}

func (p *PlatformAgent) RunTool(ctx context.Context, toolName string, params map[string]any) (tools.ToolResult, error) {
	if p == nil {
		return tools.ToolResult{
				OK:     false,
				Error:  "agent not initialized",
				Source: "platform",
			},
			fmt.Errorf("agent not initialized")
	}
	normalizedName := tools.NormalizeToolName(toolName)
	t, ok := p.tools[normalizedName]
	if !ok {
		return tools.ToolResult{
				OK:     false,
				Error:  "tool not found",
				Source: "platform",
			},
			fmt.Errorf("tool not found")
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return tools.ToolResult{
				OK:     false,
				Error:  err.Error(),
				Source: "platform",
			},
			err
	}
	out, err := t.InvokableRun(ctx, string(raw))
	if err != nil {
		return tools.ToolResult{
				OK:     false,
				Error:  err.Error(),
				Source: "platform",
			},
			nil
	}
	var result tools.ToolResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return tools.ToolResult{
				OK:     true,
				Data:   out,
				Source: "platform",
			},
			nil
	}
	return result, nil
}

func (p *PlatformAgent) Close() error {
	if p == nil || p.mcp == nil {
		return nil
	}
	return p.mcp.Close()
}
