// Package rewrite 实现 AI 编排的改写阶段。
//
// 本文件提供 ADK 集成，构建改写器 Agent。
package rewrite

import (
	"context"
	"fmt"
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
		runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}),
	}, nil
}

func runADKRewrite(ctx context.Context, runner *adk.Runner, input string, onDelta func(string)) (string, error) {
	if runner == nil {
		return "", fmt.Errorf("rewrite ADK runner is not configured")
	}
	iter := runner.Query(ctx, input)
	var last string
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
		msg, _, err := adk.GetMessage(event)
		if err != nil || msg == nil {
			continue
		}
		if msg.Role == schema.Assistant {
			content := strings.TrimSpace(msg.Content)
			last = content
			streamed = emitContentDelta(streamed, content, onDelta)
		}
	}
	if last == "" {
		return "", fmt.Errorf("rewrite stage produced empty output")
	}
	return last, nil
}

func emitContentDelta(previous, current string, onDelta func(string)) string {
	if onDelta == nil {
		return current
	}
	current = strings.TrimSpace(current)
	previous = strings.TrimSpace(previous)
	if current == "" || current == previous {
		return current
	}
	if previous != "" && strings.HasPrefix(current, previous) {
		if delta := strings.TrimSpace(current[len(previous):]); delta != "" {
			onDelta(delta)
		}
		return current
	}
	onDelta(current)
	return current
}
