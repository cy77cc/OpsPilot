package rewrite

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func NewWithADK(ctx context.Context, model einomodel.BaseChatModel) (*Rewriter, error) {
	if model == nil {
		return nil, fmt.Errorf("rewrite model is required")
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "rewrite-stage",
		Description:   "Rewrite user requests into a stable semi-structured task draft.",
		Instruction:   SystemPrompt(),
		Model:         model,
		MaxIterations: 1,
	})
	if err != nil {
		return nil, err
	}
	return &Rewriter{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}),
	}, nil
}

func runADKRewrite(ctx context.Context, runner *adk.Runner, input string) (string, error) {
	if runner == nil {
		return "", fmt.Errorf("rewrite ADK runner is not configured")
	}
	iter := runner.Query(ctx, input)
	var last string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			return "", event.Err
		}
		msg, _, err := adk.GetMessage(event)
		if err != nil || msg == nil {
			continue
		}
		if msg.Role == schema.Assistant {
			last = strings.TrimSpace(msg.Content)
		}
	}
	if last == "" {
		return "", fmt.Errorf("rewrite stage produced empty output")
	}
	return last, nil
}
