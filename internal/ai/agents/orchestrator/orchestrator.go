package orchestrator

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/change"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/cicd"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/deployment"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/governance"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/history"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/host"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/infrastructure"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/monitor"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/platform"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/service"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/ai/common/middleware"
	"github.com/cy77cc/OpsPilot/internal/ai/common/todo"
)

func NewOpsPilotAgent(ctx context.Context) (adk.ResumableAgent, error) {
	changeAgent, err := change.New(ctx)
	if err != nil {
		return nil, err
	}
	cicdAgent, err := cicd.New(ctx)
	if err != nil {
		return nil, err
	}
	k8sAgent, err := kubernetes.New(ctx)
	if err != nil {
		return nil, err
	}
	hostAgent, err := host.New(ctx)
	if err != nil {
		return nil, err
	}
	monitorAgent, err := monitor.New(ctx)
	if err != nil {
		return nil, err
	}
	deploymentAgent, err := deployment.New(ctx)
	if err != nil {
		return nil, err
	}
	governanceAgent, err := governance.New(ctx)
	if err != nil {
		return nil, err
	}
	historyAgent, err := history.New(ctx)
	if err != nil {
		return nil, err
	}
	infrastructureAgent, err := infrastructure.New(ctx)
	if err != nil {
		return nil, err
	}
	platformAgent, err := platform.New(ctx)
	if err != nil {
		return nil, err
	}
	serviceAgent, err := service.New(ctx)
	if err != nil {
		return nil, err
	}
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
	return deep.New(ctx, &deep.Config{
		Name:                   "OpsPilotAgent",
		Description:            "Main DeepAgents entrypoint for OpsPilot.",
		// Instruction:            opsPilotSystemPrompt,
		ChatModel:              model,
		ToolsConfig:            adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools}},
		SubAgents:              []adk.Agent{k8sAgent, hostAgent, monitorAgent, changeAgent, cicdAgent, deploymentAgent, governanceAgent, historyAgent, infrastructureAgent, platformAgent, serviceAgent},
		WithoutGeneralSubAgent: true,
		Handlers:               handlers,
		MaxIteration:           100,
	})
}

func newTools(ctx context.Context) []tool.BaseTool {
	return []tool.BaseTool{platform.PlatformDiscoverResources(ctx)}
}
