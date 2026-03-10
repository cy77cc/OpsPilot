package planner

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

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
		runner: adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}),
	}, nil
}

func runADKPlanner(ctx context.Context, runner *adk.Runner, input string) (string, error) {
	if runner == nil {
		return "", fmt.Errorf("planner ADK runner is not configured")
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
		if isDecisionOutput(msg) {
			last = strings.TrimSpace(msg.Content)
		}
	}
	if last == "" {
		return "", fmt.Errorf("planner stage produced empty output")
	}
	return last, nil
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
