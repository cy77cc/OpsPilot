package replan

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

func NewReplanner(ctx context.Context, processor *airuntime.ContextProcessor) (adk.Agent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.5,
	})
	if err != nil {
		return nil, err
	}
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model,
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			if processor == nil {
				return nil, nil
			}
			return processor.BuildReplannerInput(ctx, in)
		},
	})
}
