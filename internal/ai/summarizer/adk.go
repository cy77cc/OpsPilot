// Package summarizer 实现 AI 编排的总结阶段。
//
// 本文件提供 ADK 集成，构建总结器 Agent。
package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// NewWithADK 使用 ADK 创建总结器实例。
func NewWithADK(ctx context.Context, model einomodel.BaseChatModel) (*Summarizer, error) {
	if model == nil {
		return nil, fmt.Errorf("summarizer model is required")
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "summarizer-stage",
		Description:   "Summarize executor outputs into structured conclusions and replan hints.",
		Instruction:   SystemPrompt(),
		Model:         model,
		MaxIterations: 2,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{summaryDecisionTool()},
			},
			ReturnDirectly: map[string]bool{
				"emit_summary": true,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Summarizer{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}),
	}, nil
}

func runADKSummarizer(ctx context.Context, runner *adk.Runner, input string, onDelta func(string)) (string, error) {
	if runner == nil {
		return "", fmt.Errorf("summarizer ADK runner is not configured")
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
			streamed = emitSummaryDelta(streamed, msg.Content, onDelta)
		}
		if isSummaryOutput(msg) {
			last = strings.TrimSpace(msg.Content)
		}
	}
	if last == "" {
		return "", fmt.Errorf("summarizer stage produced empty output")
	}
	return last, nil
}

func emitSummaryDelta(previous, current string, onDelta func(string)) string {
	if onDelta == nil {
		return strings.TrimSpace(current)
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

func isSummaryOutput(msg *schema.Message) bool {
	if msg == nil || strings.TrimSpace(msg.Content) == "" {
		return false
	}
	switch msg.Role {
	case schema.Tool:
		return strings.TrimSpace(msg.ToolName) == "emit_summary"
	case schema.Assistant:
		return len(msg.ToolCalls) == 0
	default:
		return false
	}
}
