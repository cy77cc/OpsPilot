package kubernetes

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/ai/common/middleware"
)

func New(ctx context.Context) (adk.Agent, error) {
	model, err := newModel(ctx)
	if err != nil {
		return nil, err
	}
	tools := invokableToBaseTools(NewKubernetesTools(ctx))
	handlers, err := middleware.BuildAgentHandlers(ctx, tools)
	if err != nil {
		return nil, err
	}
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "K8sAgent",
		Description:   "Handles Kubernetes investigation and operations.",
		Instruction:   agentPrompt,
		Model:         model,
		ToolsConfig:   adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools}},
		Handlers:      handlers,
		MaxIterations: 20,
	})
}

func invokableToBaseTools(items []tool.InvokableTool) []tool.BaseTool {
	base := make([]tool.BaseTool, 0, len(items))
	for _, t := range items {
		base = append(base, t)
	}
	return base
}

func newModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	return chatmodel.GetDefaultChatModel(ctx, nil, chatmodel.ChatModelConfig{Timeout: 45 * time.Second, Temp: 0.2})
}
