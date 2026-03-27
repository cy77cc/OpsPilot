package runtime

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ProjectionState 跟踪流式投影状态。
//
// 包含两部分状态：
//   - 解析状态：用于增量解析 SSE 事件
//   - 持久化状态：用于存储到数据库（Persisted）
type ProjectionState struct {
	ReplanIteration int
	RunPhase        string
	PendingToolCall *PendingToolCallState // 用于合并流式 tool_call chunk

	// Persisted 累积的持久化状态，用于存储到数据库。
	Persisted *PersistedRuntime
}

type PendingToolCallState struct {
	AgentName    string
	CallID       string
	ToolName     string
	ArgumentsRaw string
	Arguments    map[string]any
}

func projectNormalizedEvents(events []NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	projected := make([]PublicStreamEvent, 0, len(events))
	for _, event := range events {
		projected = append(projected, projectNormalizedEvent(event, state)...)
	}
	return projected
}

func projectNormalizedEvent(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	// 确保 Persisted 已初始化
	if state.Persisted == nil {
		state.Persisted = &PersistedRuntime{}
	}

	switch event.Kind {
	case NormalizedKindHandoff:
		if event.Handoff == nil {
			return nil
		}
		// 更新持久化状态
		state.Persisted.Phase = "executing"
		state.Persisted.PhaseLabel = fmt.Sprintf("%s 开始处理", strings.TrimSpace(event.Handoff.To))
		state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
			ID:     fmt.Sprintf("handoff:%s", strings.TrimSpace(event.Handoff.To)),
			Kind:   "agent_handoff",
			Label:  strings.TrimSpace(event.Handoff.To),
			Status: "done",
		})
		return []PublicStreamEvent{{
			Event: "agent_handoff",
			Data: map[string]any{
				"from":   strings.TrimSpace(event.Handoff.From),
				"to":     strings.TrimSpace(event.Handoff.To),
				"intent": mapAgentNameToIntentType(strings.TrimSpace(event.Handoff.To)),
			},
		}}
	case NormalizedKindInterrupt:
		if event.Interrupt == nil {
			return nil
		}
		state.RunPhase = "waiting_approval"
		// 更新持久化状态
		state.Persisted.Phase = "waiting_approval"
		state.Persisted.PhaseLabel = "等待审批"
		state.Persisted.Status = &PersistedStatus{
			Kind:  "waiting_approval",
			Label: "等待审批",
		}
		state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
			ID:     event.Interrupt.CallID,
			Kind:   "tool_approval",
			Label:  event.Interrupt.ToolName,
			Detail: fmt.Sprintf("等待审批 %ds", event.Interrupt.TimeoutSeconds),
			Status: "pending",
		})
		return []PublicStreamEvent{
			{
				Event: "tool_approval",
				Data: map[string]any{
					"approval_id":     event.Interrupt.ApprovalID,
					"call_id":         event.Interrupt.CallID,
					"tool_name":       event.Interrupt.ToolName,
					"preview":         event.Interrupt.Preview,
					"timeout_seconds": event.Interrupt.TimeoutSeconds,
				},
			},
			NewRunStateEvent("waiting_approval", map[string]any{
				"agent": event.AgentName,
			}),
		}
	case NormalizedKindToolCall:
		if event.Tool == nil {
			return nil
		}
		// 更新持久化状态
		state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
			ID:        event.Tool.CallID,
			Kind:      "tool_call",
			Label:     event.Tool.ToolName,
			Status:    "active",
			Arguments: event.Tool.Arguments,
		})
		return []PublicStreamEvent{{
			Event: "tool_call",
			Data: map[string]any{
				"call_id":   event.Tool.CallID,
				"tool_name": event.Tool.ToolName,
				"arguments": event.Tool.Arguments,
				"agent":     strings.TrimSpace(event.AgentName),
			},
		}}
	case NormalizedKindToolResult:
		if event.Tool == nil {
			return nil
		}
		status := "done"
		if strings.TrimSpace(event.Tool.Phase) == "error" {
			status = "error"
		}

		// 更新持久化状态：找到对应的 activity 并更新
		for i := range state.Persisted.Activities {
			if state.Persisted.Activities[i].ID == event.Tool.CallID {
				state.Persisted.Activities[i].Status = status
			}
		}

		resultActivityID := toolResultActivityID(event.Tool.CallID)
		state.Persisted.Activities = append(state.Persisted.Activities, PersistedActivity{
			ID:         resultActivityID,
			Kind:       "tool_result",
			Label:      event.Tool.ToolName,
			Detail:     truncateString(event.Tool.Content, 200),
			RawContent: event.Tool.Content,
			Status:     status,
		})

		return []PublicStreamEvent{{
			Event: "tool_result",
			Data: map[string]any{
				"call_id":   event.Tool.CallID,
				"tool_name": event.Tool.ToolName,
				"content":   event.Tool.Content,
				"status":    status,
				"agent":     strings.TrimSpace(event.AgentName),
			},
		}}
	case NormalizedKindMessage:
		return projectNormalizedMessage(event, state)
	default:
		return nil
	}
}

func toolResultActivityID(callID string) string {
	return callID + ":result"
}

func projectNormalizedMessage(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	if event.Message == nil {
		return nil
	}

	trimmedAgent := strings.TrimSpace(event.AgentName)
	trimmedContent := strings.TrimSpace(event.Message.Content)

	// 确保 Persisted 已初始化
	if state.Persisted == nil {
		state.Persisted = &PersistedRuntime{}
	}

	projected := make([]PublicStreamEvent, 0, 1)
	if trimmedContent != "" {
		projected = append(projected, PublicStreamEvent{
			Event: "delta",
			Data: map[string]any{
				"content": event.Message.Content,
				"agent":   trimmedAgent,
			},
		})
	}
	return projected
}

// truncateString 截断字符串到指定长度。
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	return string([]rune(s)[:maxLen])
}
