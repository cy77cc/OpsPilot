package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/k8s-manage/internal/ai/model"
)

func NewExecutor(ctx context.Context, tools []tool.BaseTool) (adk.Agent, error) {	// Get travel tools for the executor
	
	chatModel, err := model.NewToolCallingChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}
	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})
}
