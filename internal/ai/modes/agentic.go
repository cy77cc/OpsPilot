package modes

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/agent"
	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/ai/types"
)

type AgenticMode struct {
	runner *agent.PlatformRunner
}

func NewAgenticMode(ctx context.Context, chatModel model.ToolCallingChatModel, deps aitools.PlatformDeps, cfg *agent.RunnerConfig) (*AgenticMode, error) {
	runner, err := agent.NewPlatformRunner(ctx, chatModel, deps, cfg)
	if err != nil {
		return nil, err
	}
	return &AgenticMode{runner: runner}, nil
}

func (m *AgenticMode) Execute(ctx context.Context, sessionID, message string, gen *adk.AsyncGenerator[*types.AgentResult]) {
	if gen == nil {
		return
	}
	if m == nil || m.runner == nil {
		gen.Send(&types.AgentResult{Type: "error", Content: "agentic runner not initialized"})
		return
	}

	iter := m.runner.Query(ctx, sessionID, message)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if result := m.processEvent(event); result != nil {
			gen.Send(result)
		}
	}
}

func (m *AgenticMode) Resume(ctx context.Context, sessionID, askID string, response any) (*types.AgentResult, error) {
	if m == nil || m.runner == nil {
		return nil, fmt.Errorf("agentic runner not initialized")
	}
	targets := map[string]any{}
	if id := strings.TrimSpace(askID); id != "" {
		targets[id] = response
	}
	iter, err := m.runner.Resume(ctx, strings.TrimSpace(sessionID), targets)
	if err != nil {
		return nil, err
	}
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if result := m.processEvent(event); result != nil {
			return result, nil
		}
	}
	return nil, fmt.Errorf("no output from agent")
}

func (m *AgenticMode) processEvent(event *adk.AgentEvent) *types.AgentResult {
	if event == nil {
		return nil
	}
	if event.Err != nil {
		return &types.AgentResult{Type: "error", Content: event.Err.Error()}
	}
	if event.Action != nil && event.Action.Interrupted != nil {
		return processInterrupt(event.Action.Interrupted)
	}
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil
	}
	if stream := event.Output.MessageOutput.MessageStream; stream != nil {
		var content strings.Builder
		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return &types.AgentResult{Type: "error", Content: err.Error()}
			}
			if chunk != nil {
				content.WriteString(strings.TrimSpace(chunk.Content))
			}
		}
		stream.Close()
		if text := strings.TrimSpace(content.String()); text != "" {
			return &types.AgentResult{Type: "text", Content: text}
		}
	}
	msg := event.Output.MessageOutput.Message
	if msg == nil {
		return nil
	}
	switch msg.Role {
	case schema.Tool:
		return &types.AgentResult{
			Type:     "tool_result",
			Content:  strings.TrimSpace(msg.Content),
			ToolName: strings.TrimSpace(msg.ToolName),
			ToolData: map[string]any{"content": strings.TrimSpace(msg.Content)},
		}
	default:
		return &types.AgentResult{
			Type:    "text",
			Content: strings.TrimSpace(msg.Content),
		}
	}
}

func processInterrupt(info *adk.InterruptInfo) *types.AgentResult {
	if info == nil {
		return nil
	}

	id := firstInterruptTarget(info.InterruptContexts)
	switch data := info.Data.(type) {
	case *aitools.ApprovalInfo:
		return &types.AgentResult{
			Type: "ask_user",
			Ask: &types.AskRequest{
				ID:          id,
				Title:       data.ToolName,
				Description: "高风险操作需要审批后继续执行",
				Risk:        string(data.Risk),
				Details: map[string]any{
					"arguments":         data.ArgumentsInJSON,
					"preview":           data.Preview,
					"interrupt_targets": interruptTargets(info.InterruptContexts),
				},
			},
		}
	case *aitools.ReviewEditInfo:
		return &types.AgentResult{
			Type: "ask_user",
			Ask: &types.AskRequest{
				ID:          id,
				Title:       data.ToolName,
				Description: "参数需要确认后继续执行",
				Risk:        "medium",
				Details: map[string]any{
					"arguments":         data.ArgumentsInJSON,
					"interrupt_targets": interruptTargets(info.InterruptContexts),
				},
			},
		}
	default:
		return &types.AgentResult{
			Type: "ask_user",
			Ask: &types.AskRequest{
				ID:          id,
				Title:       "需要继续确认",
				Description: "Agent 执行被中断，等待恢复输入",
				Details: map[string]any{
					"interrupt_targets": interruptTargets(info.InterruptContexts),
				},
			},
		}
	}
}

func interruptTargets(contexts []*adk.InterruptCtx) []string {
	out := make([]string, 0, len(contexts))
	for _, item := range contexts {
		if item == nil || !item.IsRootCause {
			continue
		}
		id := strings.TrimSpace(item.ID)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func firstInterruptTarget(contexts []*adk.InterruptCtx) string {
	targets := interruptTargets(contexts)
	if len(targets) == 0 {
		return ""
	}
	return targets[0]
}
