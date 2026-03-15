// Package runtime 提供 SSE 事件转换器。
//
// SSEConverter 将 AI 运行时状态转换为标准 SSE StreamEvent 格式。
package runtime

import "strings"

// SSEConverter 将 AI 运行时状态转换为 SSE 事件。
type SSEConverter struct{}

// NewSSEConverter 创建 SSE 转换器。
func NewSSEConverter() *SSEConverter {
	return &SSEConverter{}
}

// OnMeta 发送会话元信息事件。
func (c *SSEConverter) OnMeta(sessionID, planID, turnID, traceID string) StreamEvent {
	return StreamEvent{Type: EventMeta, Data: compactMap(map[string]any{
		"session_id": sessionID,
		"plan_id":    planID,
		"turn_id":    turnID,
		"trace_id":   traceID,
	})}
}

// OnTextDelta 发送文本增量事件。
func (c *SSEConverter) OnTextDelta(chunk string) StreamEvent {
	return StreamEvent{Type: EventDelta, Data: map[string]any{"content": chunk}}
}

// OnToolCall 发送工具调用事件。
func (c *SSEConverter) OnToolCall(callID, toolName, toolDisplayName, arguments string) StreamEvent {
	return StreamEvent{Type: EventToolCall, Data: compactMap(map[string]any{
		"call_id":           callID,
		"tool_name":         toolName,
		"tool_display_name": toolDisplayName,
		"arguments":         arguments,
	})}
}

// OnToolApproval 发送工具审批等待事件。
func (c *SSEConverter) OnToolApproval(callID, toolName, toolDisplayName, risk, summary, argumentsJSON, approvalID, checkpointID string) StreamEvent {
	return StreamEvent{Type: EventToolApproval, Data: compactMap(map[string]any{
		"call_id":           callID,
		"tool_name":         toolName,
		"tool_display_name": toolDisplayName,
		"risk":              risk,
		"summary":           summary,
		"arguments_json":    argumentsJSON,
		"approval_id":       approvalID,
		"checkpoint_id":     checkpointID,
	})}
}

// OnToolResult 发送工具结果事件。
func (c *SSEConverter) OnToolResult(callID, toolName, result string) StreamEvent {
	return StreamEvent{Type: EventToolResult, Data: compactMap(map[string]any{
		"call_id":   callID,
		"tool_name": toolName,
		"result":    result,
	})}
}

// OnDone 发送执行完成事件。
func (c *SSEConverter) OnDone(status string) StreamEvent {
	return StreamEvent{Type: EventDone, Data: map[string]any{"status": status}}
}

// OnError 发送执行错误事件。
func (c *SSEConverter) OnError(stage string, err error) StreamEvent {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return StreamEvent{Type: EventError, Data: map[string]any{"phase": stage, "message": message}}
}

func compactMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch v := value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(v) == "" {
				continue
			}
		case map[string]any:
			if len(v) == 0 {
				continue
			}
		case []map[string]any:
			if len(v) == 0 {
				continue
			}
		}
		out[key] = value
	}
	return out
}
