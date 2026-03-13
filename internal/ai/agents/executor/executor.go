package executor

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

func NewExecutor(ctx context.Context, deps common.PlatformDeps, processor *airuntime.ContextProcessor) (adk.Agent, error) {
	toolset := tools.NewAllTools(ctx, deps)

	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
	})
	if err != nil {
		return nil, err
	}

	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolset,
			},
		},
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			if processor != nil {
				return processor.BuildExecutorInput(ctx, in, toolset)
			}
			return defaultExecutorInput(ctx, in)
		},
	})
}

func defaultExecutorInput(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
	planContent, err := in.Plan.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return executorPrompt.Format(ctx, map[string]any{
		"input":          in.UserInput,
		"plan":           string(planContent),
		"executed_steps": in.ExecutedSteps,
		"step":           in.Plan.FirstStep(),
	})
}
