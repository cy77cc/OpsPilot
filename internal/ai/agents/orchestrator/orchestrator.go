package orchestrator

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/change"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/host"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/monitor"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/ai/common/middleware"
	"github.com/cy77cc/OpsPilot/internal/ai/common/todo"
)

func NewOpsPilotAgent(ctx context.Context) (adk.ResumableAgent, error) {
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
	changeAgent, err := change.New(ctx)
	if err != nil {
		return nil, err
	}
	writeOpsTodos, err := todo.NewWriteOpsTodosMiddleware()
	if err != nil {
		return nil, err
	}
	handlers, err := middleware.BuildAgentHandlers(ctx, nil)
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
		Instruction:            opsPilotSystemPrompt,
		ChatModel:              model,
		SubAgents:              []adk.Agent{k8sAgent, hostAgent, monitorAgent, changeAgent},
		WithoutGeneralSubAgent: true,
		Handlers:               handlers,
		MaxIteration:           20,
	})
}
