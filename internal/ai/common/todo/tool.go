package todo

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

// NewWriteOpsTodosMiddleware creates the tool middleware used by DeepAgents.
func NewWriteOpsTodosMiddleware() (adk.ChatModelAgentMiddleware, error) {
	toolDesc := "Write or replace the current ops todo list."
	resultMsg := "Updated ops todo list to %s"

	invokable, err := utils.InferTool(writeOpsTodosToolName, toolDesc, func(ctx context.Context, input writeOpsTodosArguments) (string, error) {
		adk.AddSessionValue(ctx, SessionKeyOpsTodos, input.Todos)
		raw, err := sonic.MarshalString(input.Todos)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(resultMsg, raw), nil
	})
	if err != nil {
		return nil, err
	}

	return &appendPromptTool{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		t:                            invokable,
	}, nil
}

const writeOpsTodosToolName = "write_ops_todos"

type appendPromptTool struct {
	*adk.BaseChatModelAgentMiddleware
	t tool.BaseTool
}

func (w *appendPromptTool) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	if runCtx == nil {
		runCtx = &adk.ChatModelAgentContext{}
	}
	nextCtx := *runCtx
	if w.t != nil {
		nextCtx.Tools = append(nextCtx.Tools, w.t)
	}
	return ctx, &nextCtx, nil
}
