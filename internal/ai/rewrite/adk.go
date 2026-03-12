// Package rewrite 实现 AI 编排的改写阶段。
//
// 本文件提供 ADK 集成，构建改写器 Agent。
package rewrite

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// NewWithADK 使用 ADK 创建改写器实例。
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
		runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true}),
	}, nil
}

func runADKRewrite(ctx context.Context, runner *adk.Runner, input string, onDelta func(string)) (string, error) {
	if runner == nil {
		return "", fmt.Errorf("rewrite ADK runner is not configured")
	}
	iter := runner.Run(ctx, []adk.Message{schema.UserMessage(input)})
	var final string
	var streamed string
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
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		output := event.Output.MessageOutput
		if output.IsStreaming && output.MessageStream != nil {
			for {
				msg, err := output.MessageStream.Recv()
				if err != nil {
					if err == io.EOF {
						break
					}
					return "", err
				}
				if msg == nil || msg.Role != schema.Assistant {
					continue
				}
				content := msg.Content
				final = mergeStreamContent(final, content)
				streamed = emitContentDelta(streamed, content, onDelta)
			}
			continue
		}
		msg := output.Message
		if msg != nil && msg.Role == schema.Assistant {
			content := msg.Content
			final = mergeStreamContent(final, content)
			streamed = emitContentDelta(streamed, content, onDelta)
		}
	}
	final = strings.TrimSpace(final)
	if final == "" {
		return "", fmt.Errorf("rewrite stage produced empty output")
	}
	return final, nil
}

func emitContentDelta(previous, current string, onDelta func(string)) string {
	if onDelta == nil {
		return current
	}
	if current == "" || current == previous {
		return current
	}
	if previous != "" && strings.HasPrefix(current, previous) {
		onDelta(current[len(previous):])
		return current
	}
	onDelta(current)
	return current
}

func mergeStreamContent(previous, current string) string {
	if current == "" {
		return previous
	}
	if previous == "" {
		return current
	}
	if strings.HasPrefix(current, previous) {
		return current
	}
	return previous + current
}
