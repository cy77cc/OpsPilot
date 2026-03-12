// Package planner 实现 AI 编排的规划阶段。
//
// 本文件提供 ADK (Agent Development Kit) 集成，构建规划器 Agent。
package planner

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

// NewWithADK 使用 ADK 创建规划器实例。
// 配置 Agent 的系统提示、工具集和决策工具。
func NewWithADK(ctx context.Context, model einomodel.BaseChatModel, deps common.PlatformDeps) (*Planner, error) {
	if model == nil {
		return nil, fmt.Errorf("planner model is required")
	}
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "planner-stage",
		Description:   "Plan clarified AI operations tasks into structured decisions and steps.",
		Instruction:   SystemPrompt(),
		Model:         model,
		MaxIterations: 4,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: append(tools.NewCommonTools(ctx, deps), decisionTools()...),
			},
			ReturnDirectly: map[string]bool{
				"clarify":      true,
				"reject":       true,
				"direct_reply": true,
				"plan":         true,
			},
		},
	})

	if err != nil {
		return nil, err
	}
	return &Planner{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: true}),
	}, nil
}

// runADKPlanner 运行 ADK 规划器并收集输出。
func runADKPlanner(ctx context.Context, runner *adk.Runner, input string, onDelta func(string)) (string, error) {
	if runner == nil {
		return "", fmt.Errorf("planner ADK runner is not configured")
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
				if msg == nil {
					continue
				}
				if msg.Role == schema.Assistant {
					streamed = emitPlannerDelta(streamed, msg.Content, onDelta)
				}
				if isDecisionOutput(msg) {
					final = mergeDecisionOutput(final, msg.Content)
				}
			}
			continue
		}
		msg := output.Message
		if msg == nil {
			continue
		}
		if msg.Role == schema.Assistant {
			streamed = emitPlannerDelta(streamed, msg.Content, onDelta)
		}
		if isDecisionOutput(msg) {
			final = mergeDecisionOutput(final, msg.Content)
		}
	}
	final = strings.TrimSpace(final)
	if final == "" {
		return "", fmt.Errorf("planner stage produced empty output")
	}
	return final, nil
}

func emitPlannerDelta(previous, current string, onDelta func(string)) string {
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

func isDecisionOutput(msg *schema.Message) bool {
	if msg == nil {
		return false
	}
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return false
	}
	switch msg.Role {
	case schema.Tool:
		return isDecisionToolName(msg.ToolName)
	case schema.Assistant:
		return len(msg.ToolCalls) == 0
	default:
		return false
	}
}

func isDecisionToolName(name string) bool {
	switch strings.TrimSpace(name) {
	case "clarify", "reject", "direct_reply", "plan":
		return true
	default:
		return false
	}
}

func mergeDecisionOutput(previous, current string) string {
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
