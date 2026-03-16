package agents

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/prompt"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
)

func NewRouterAgent(ctx context.Context) (*adk.ChatModelAgent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.2,
	})
	if err != nil {
		return nil, err
	}
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "OpsPilotAgent",
		Description:   "OpsPilot infrastructure operations assistant with approval-gated tool execution",
		Instruction:   prompt.ROUTERPROMPT,
		Model:         model,
		MaxIterations: 3,
	})
}
