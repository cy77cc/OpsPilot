package qa

import (
	"context"

	"github.com/cloudwego/eino/adk"
)

func NewQAAgent(ctx context.Context) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{

	})
}