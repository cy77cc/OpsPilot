package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/model"
)

func NewPlanner(ctx context.Context) (adk.Agent, error) {

	chatModel, err := model.NewToolCallingChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: chatModel,
	})
}

var executorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(platformAgentInstruction))

func NewReplanAgent(ctx context.Context) (adk.Agent, error) {
	chatModel, err := model.NewToolCallingChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: chatModel,
	})
}
