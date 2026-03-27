// Package runtime 提供 AI 运行时的 SSE 流式编码工具。
//
// 本包负责将内部事件序列化为前端可消费的 SSE 消息，
// 并通过白名单机制防止内部事件泄露给外部调用方。
//
// 事件分层说明：
//
//	公开事件（publicEventNames 白名单）：可通过 EncodePublicEvent 编码推送给前端。
//	内部事件（如 thinking_delta）：仅在运行时内部流转，不对外暴露。
package runtime

import (
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// StreamEvent 是推送给前端的 SSE 消息体。
//
// Event 字段与前端约定的事件名称对应，Data 为该事件的结构化载荷。
type StreamEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// publicEventNames 是允许通过 EncodePublicEvent 编码的事件名称白名单。
//
// 白名单与前端 A2UI 事件契约保持一致：
//
//	会话层：meta
//	路由层：agent_handoff
//	执行层：delta, tool_call, tool_result, tool_approval
//	终止层：done, error
//
// 注意：thinking_delta 为内部专用事件，不在白名单内。
var publicEventNames = map[string]struct{}{
	"meta":          {},
	"agent_handoff": {},
	"delta":         {},
	"tool_call":     {},
	"tool_result":   {},
	"tool_approval": {},
	"run_state":     {},
	"done":          {},
	"error":         {},
}

func NewMetaEvent(sessionID, runID string, turn int) StreamEvent {
	return StreamEvent{
		Event: "meta",
		Data: map[string]any{
			"session_id": sessionID,
			"run_id":     runID,
			"turn":       turn,
		},
	}
}

func NewDoneEvent(runID string, iterations int) StreamEvent {
	return doneEvent(runID, iterations, "completed")
}

func NewErrorEvent(runID string, err error) StreamEvent {
	return errorEvent(runID, err)
}

func projectAgentHandoff(event *adk.AgentEvent) *StreamEvent {
	if event == nil || event.Action == nil || event.Action.TransferToAgent == nil {
		return nil
	}

	dest := strings.TrimSpace(event.Action.TransferToAgent.DestAgentName)
	if dest == "" {
		return nil
	}

	return &StreamEvent{
		Event: "agent_handoff",
		Data: map[string]any{
			"from":   strings.TrimSpace(event.AgentName),
			"to":     dest,
			"intent": mapAgentNameToIntentType(dest),
		},
	}
}

func projectAgentEvent(event *adk.AgentEvent, state *a2uiProjectionState) []StreamEvent {
	if event == nil {
		return nil
	}

	projected := make([]StreamEvent, 0, 4)
	if handoff := projectAgentHandoff(event); handoff != nil {
		projected = append(projected, *handoff)
	}
	if approval := projectApprovalEvent(event); approval != nil {
		projected = append(projected, *approval)
	}

	if event.Output != nil && event.Output.MessageOutput != nil {
		projected = append(projected, projectMessageVariant(event.AgentName, event.Output.MessageOutput, state)...)
	}
	return projected
}

func projectMessageVariant(agentName string, variant *adk.MessageVariant, state *a2uiProjectionState) []StreamEvent {
	if variant == nil {
		return nil
	}

	message, err := variant.GetMessage()
	if err != nil {
		return nil
	}

	switch message.Role {
	case schema.Assistant:
		return projectAssistantMessage(agentName, message, state)
	case schema.Tool:
		return projectToolMessage(message)
	default:
		return nil
	}
}

func projectAssistantMessage(agentName string, message *schema.Message, state *a2uiProjectionState) []StreamEvent {
	if message == nil {
		return nil
	}

	trimmedAgent := strings.TrimSpace(agentName)
	trimmedContent := strings.TrimSpace(message.Content)

	projected := make([]StreamEvent, 0, len(message.ToolCalls)+1)
	if trimmedContent != "" {
		projected = append(projected, StreamEvent{
			Event: "delta",
			Data: map[string]any{
				"content": message.Content,
				"agent":   trimmedAgent,
			},
		})
	}

	for _, toolCall := range message.ToolCalls {
		if !shouldProjectToolCall(toolCall.ID, toolCall.Function.Name, toolCall.Function.Arguments) {
			continue
		}
		projected = append(projected, StreamEvent{
			Event: "tool_call",
			Data: map[string]any{
				"call_id":   toolCall.ID,
				"tool_name": toolCall.Function.Name,
				"arguments": decodeToolArgumentsFromRaw(toolCall.Function.Arguments),
			},
		})
	}

	return projected
}

func projectToolMessage(message *schema.Message) []StreamEvent {
	if message == nil || strings.TrimSpace(message.ToolName) == "transfer_to_agent" {
		return nil
	}

	return []StreamEvent{{
		Event: "tool_result",
		Data: map[string]any{
			"call_id":   message.ToolCallID,
			"tool_name": message.ToolName,
			"content":   message.Content,
		},
	}}
}

func projectApprovalEvent(event *adk.AgentEvent) *StreamEvent {
	if event == nil || event.Action == nil || event.Action.Interrupted == nil {
		return nil
	}

	interrupt, ok := parseApprovalInterrupt(event.Action.Interrupted)
	if !ok {
		return nil
	}

	return &StreamEvent{
		Event: "tool_approval",
		Data: map[string]any{
			"approval_id":     interrupt.ApprovalID,
			"call_id":         interrupt.CallID,
			"tool_name":       interrupt.ToolName,
			"preview":         interrupt.Preview,
			"timeout_seconds": interrupt.TimeoutSeconds,
		},
	}
}

func doneEvent(runID string, iterations int, status string) StreamEvent {
	if strings.TrimSpace(status) == "" {
		status = "completed"
	}
	return StreamEvent{
		Event: "done",
		Data: map[string]any{
			"run_id":     runID,
			"status":     status,
			"iterations": iterations,
		},
	}
}

func errorEvent(runID string, err error) StreamEvent {
	payload := map[string]any{
		"code":        "EXECUTION_FAILED",
		"message":     "AI execution failed",
		"recoverable": false,
	}
	if strings.TrimSpace(runID) != "" {
		payload["run_id"] = runID
	}
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		payload["message"] = err.Error()
	}
	return StreamEvent{Event: "error", Data: payload}
}

func decodeToolArgumentsFromRaw(raw string) map[string]any {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err == nil && payload != nil {
		return payload
	}
	return map[string]any{"raw": raw}
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok && typed != nil {
		return typed
	}
	return map[string]any{}
}
