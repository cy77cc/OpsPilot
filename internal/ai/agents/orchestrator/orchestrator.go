package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/change"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/cicd"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/deployment"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/governance"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/host"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/infrastructure"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/monitor"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/service"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/ai/common/middleware"
	"github.com/cy77cc/OpsPilot/internal/ai/common/todo"
)

type SpecialistFactory func(context.Context) (adk.Agent, error)

type SpecialistRegistry map[string]adk.Agent

func defaultSpecialistFactories() map[string]SpecialistFactory {
	return map[string]SpecialistFactory{
		"kubernetes":     func(ctx context.Context) (adk.Agent, error) { return kubernetes.New(ctx) },
		"host":           func(ctx context.Context) (adk.Agent, error) { return host.New(ctx) },
		"monitor":        func(ctx context.Context) (adk.Agent, error) { return monitor.New(ctx) },
		"change":         func(ctx context.Context) (adk.Agent, error) { return change.New(ctx) },
		"cicd":           func(ctx context.Context) (adk.Agent, error) { return cicd.New(ctx) },
		"deployment":     func(ctx context.Context) (adk.Agent, error) { return deployment.New(ctx) },
		"governance":     func(ctx context.Context) (adk.Agent, error) { return governance.New(ctx) },
		"infrastructure": func(ctx context.Context) (adk.Agent, error) { return infrastructure.New(ctx) },
		"service":        func(ctx context.Context) (adk.Agent, error) { return service.New(ctx) },
	}
}

func registerSpecialists(ctx context.Context, factories map[string]SpecialistFactory) (SpecialistRegistry, error) {
	registry := make(SpecialistRegistry, len(factories))
	ordered := []string{
		"kubernetes", "host", "monitor", "change", "cicd",
		"deployment", "governance", "infrastructure", "service",
	}
	for _, name := range ordered {
		factory, ok := factories[name]
		if !ok || factory == nil {
			continue
		}
		agent, err := factory(ctx)
		if err != nil {
			return nil, fmt.Errorf("init specialist %s: %w", name, err)
		}
		registry[name] = agent
	}
	return registry, nil
}

func specialistsInDefaultOrder(registry SpecialistRegistry) []adk.Agent {
	ordered := []string{
		"kubernetes", "host", "monitor", "change", "cicd",
		"deployment", "governance", "infrastructure", "service",
	}
	out := make([]adk.Agent, 0, len(registry))
	for _, name := range ordered {
		agent, ok := registry[name]
		if !ok {
			continue
		}
		out = append(out, agent)
	}
	return out
}

func buildPrimaryAgent(
	ctx context.Context,
	model einomodel.ToolCallingChatModel,
	tools []tool.BaseTool,
	handlers []adk.ChatModelAgentMiddleware,
	specialists []adk.Agent,
) (adk.ResumableAgent, error) {
	return deep.New(ctx, &deep.Config{
		Name:        "OpsPilotAgent",
		Description: "Primary DeepAgents entrypoint for OpsPilot.",
		ChatModel:   model,
		ToolsConfig: adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools}},
		// Primary-agent-first seam: specialists remain registerable/available,
		// but orchestration now enters through a single primary surface.
		SubAgents:              specialists,
		WithoutGeneralSubAgent: true,
		Handlers:               handlers,
		MaxIteration:           100,
	})
}

func NewOpsPilotAgent(ctx context.Context) (adk.ResumableAgent, error) {
	registry, err := registerSpecialists(ctx, defaultSpecialistFactories())
	if err != nil {
		return nil, err
	}
	specialists := specialistsInDefaultOrder(registry)

	tools := newTools(ctx)
	writeOpsTodos, err := todo.NewWriteOpsTodosMiddleware()
	if err != nil {
		return nil, err
	}
	handlers, err := middleware.BuildAgentHandlers(ctx, tools)
	if err != nil {
		return nil, err
	}
	handlers = append(handlers, writeOpsTodos)

	model, err := chatmodel.GetDefaultChatModel(ctx, nil, chatmodel.ChatModelConfig{
		Timeout: 45 * time.Second,
		Temp:    0.2,
	})
	if err != nil {
		return nil, err
	}
	return buildPrimaryAgent(ctx, model, tools, handlers, specialists)
}

func newTools(ctx context.Context) []tool.BaseTool {
	return []tool.BaseTool{
		ToolSearch(ctx),
		LoadSessionHistory(ctx),
		LoadTaskContext(ctx),
		LoadArtifactContext(ctx),
	}
}
