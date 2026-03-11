// Package summarizer 实现 AI 编排的总结阶段。
//
// 本文件提供 ADK 集成，构建总结器 Agent。
package summarizer

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// NewWithADK 使用 ADK 创建总结器实例。
func NewWithADK(ctx context.Context, model einomodel.BaseChatModel) (*Summarizer, error) {
	if model == nil {
		return nil, fmt.Errorf("summarizer model is required")
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "summarizer-stage",
		Description:   "Stream the final user-facing answer from executor outputs.",
		Instruction:   SystemPrompt(),
		Model:         model,
		MaxIterations: 2,
	})
	if err != nil {
		return nil, err
	}
	return &Summarizer{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}),
	}, nil
}

func runADKSummarizer(ctx context.Context, runner *adk.Runner, input []*schema.Message, onThinkingDelta func(string), onAnswerDelta func(string)) (string, error) {
	if runner == nil {
		return "", fmt.Errorf("summarizer ADK runner is not configured")
	}

	iter := runner.Run(ctx, input)

	var streamedAnswer string
	var streamedReasoning string
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
				streamedReasoning = emitSummaryDelta(streamedReasoning, msg.ReasoningContent, onThinkingDelta)
				streamedAnswer = emitSummaryDelta(streamedAnswer, msg.Content, onAnswerDelta)
			}
			continue
		}
		msg := output.Message
		if msg != nil && msg.Role == schema.Assistant {
			streamedReasoning = emitSummaryDelta(streamedReasoning, msg.ReasoningContent, onThinkingDelta)
			streamedAnswer = emitSummaryDelta(streamedAnswer, msg.Content, onAnswerDelta)
		}
	}
	if strings.TrimSpace(streamedAnswer) == "" {
		return "", fmt.Errorf("summarizer stage produced empty output")
	}
	return strings.TrimSpace(streamedAnswer), nil
}

func emitSummaryDelta(previous, current string, onDelta func(string)) string {
	current = strings.TrimSpace(current)
	previous = strings.TrimSpace(previous)
	if current == "" {
		return previous
	}
	if onDelta == nil {
		if previous != "" && !strings.HasPrefix(current, previous) {
			return strings.TrimSpace(previous + current)
		}
		return current
	}
	if current == previous {
		return current
	}
	if previous != "" && strings.HasPrefix(current, previous) {
		if delta := strings.TrimSpace(current[len(previous):]); delta != "" {
			onDelta(delta)
		}
		return current
	}
	onDelta(current)
	return strings.TrimSpace(previous + current)
}
