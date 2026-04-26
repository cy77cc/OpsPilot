// Package stream 实现事件投影和编组。
package stream

import (
	"encoding/json"
	"strings"

	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
)

// AccumulateProjectedEvents 累积投影事件到 summary builder。
func AccumulateProjectedEvents(events []airuntime.PublicStreamEvent, assistantContent *strings.Builder) RunUpdate {
	update := RunUpdate{}
	for _, projected := range events {
		if projected.Event == "delta" {
			if data, ok := projected.Data.(map[string]any); ok {
				agent := strings.TrimSpace(StringValue(data, "agent"))
				if content, ok := data["content"].(string); ok && agent != "executor" && assistantContent != nil {
					assistantContent.WriteString(content)
				}
			}
		}
		if projected.Event == "agent_handoff" {
			if data, ok := projected.Data.(map[string]any); ok {
				if assistantType, ok := data["to"].(string); ok {
					update.AssistantType = assistantType
				}
				if intentType, ok := data["intent"].(string); ok {
					update.IntentType = intentType
				}
			}
		}
	}
	return update
}

// MarshalProjectedEvent 将事件名和负载编组为类型化事件。
func MarshalProjectedEvent(eventName string, payload any) (airuntime.EventType, string, error) {
	data, _ := payload.(map[string]any)
	switch eventName {
	case "meta":
		return marshalTypedEvent(airuntime.EventTypeMeta, &airuntime.MetaPayload{
			RunID:     StringValue(data, "run_id"), SessionID: StringValue(data, "session_id"), Turn: IntValue(data, "turn"),
		})
	case "agent_handoff":
		return marshalTypedEvent(airuntime.EventTypeAgentHandoff, &airuntime.AgentHandoffPayload{
			From: StringValue(data, "from"), To: StringValue(data, "to"), Intent: StringValue(data, "intent"),
		})
	case "plan":
		return marshalTypedEvent(airuntime.EventTypePlan, &airuntime.PlanPayload{
			Iteration: IntValue(data, "iteration"), Steps: StringSliceValue(data, "steps"),
		})
	case "replan":
		return marshalTypedEvent(airuntime.EventTypeReplan, &airuntime.ReplanPayload{
			Iteration: IntValue(data, "iteration"), Completed: IntValue(data, "completed"), IsFinal: BoolValue(data, "is_final"), Steps: StringSliceValue(data, "steps"),
		})
	case "delta":
		return marshalTypedEvent(airuntime.EventTypeDelta, &airuntime.DeltaPayload{
			Agent: StringValue(data, "agent"), Content: StringValue(data, "content"),
		})
	case "tool_call":
		if strings.TrimSpace(StringValue(data, "call_id")) == "" || strings.TrimSpace(StringValue(data, "tool_name")) == "" {
			return "", "", nil
		}
		return marshalTypedEvent(airuntime.EventTypeToolCall, &airuntime.ToolCallPayload{
			Agent: StringValue(data, "agent"), CallID: StringValue(data, "call_id"), ToolName: StringValue(data, "tool_name"), Arguments: MapValue(data, "arguments"),
		})
	case "tool_approval":
		if strings.TrimSpace(StringValue(data, "approval_id")) == "" || strings.TrimSpace(StringValue(data, "call_id")) == "" || strings.TrimSpace(StringValue(data, "tool_name")) == "" {
			return "", "", nil
		}
		return marshalTypedEvent(airuntime.EventTypeToolApproval, &airuntime.ToolApprovalPayload{
			ApprovalID: StringValue(data, "approval_id"), TargetID: StringValue(data, "target_id"), CallID: StringValue(data, "call_id"), ToolName: StringValue(data, "tool_name"), Preview: MapValue(data, "preview"), TimeoutSeconds: IntValue(data, "timeout_seconds"),
		})
	case "tool_result":
		return marshalTypedEvent(airuntime.EventTypeToolResult, &airuntime.ToolResultPayload{
			Agent: StringValue(data, "agent"), CallID: StringValue(data, "call_id"), ToolName: StringValue(data, "tool_name"), Content: StringValue(data, "content"), Status: StringValue(data, "status"),
		})
	case "run_state":
		if strings.TrimSpace(StringValue(data, "status")) == "" {
			return "", "", nil
		}
		return marshalTypedEvent(airuntime.EventTypeRunState, &airuntime.RunStatePayload{Status: StringValue(data, "status"), Agent: StringValue(data, "agent")})
	case "done":
		return marshalTypedEvent(airuntime.EventTypeDone, &airuntime.DonePayload{RunID: StringValue(data, "run_id"), Status: StringValue(data, "status"), Summary: StringValue(data, "summary"), Iterations: IntValue(data, "iterations")})
	case "error":
		return marshalTypedEvent(airuntime.EventTypeError, &airuntime.ErrorPayload{RunID: StringValue(data, "run_id"), Message: StringValue(data, "message"), Code: StringValue(data, "code")})
	default:
		return "", "", nil
	}
}

func marshalTypedEvent(eventType airuntime.EventType, payload any) (airuntime.EventType, string, error) {
	raw, err := airuntime.MarshalEventPayload(eventType, payload)
	return eventType, raw, err
}

// DecodeRunEventPayload 解码运行事件负载。
func DecodeRunEventPayload(raw string) (any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return map[string]any{}, nil
	}
	return payload, nil
}

// SanitizeUserFacingError 对用户隐藏内部错误详情。
func SanitizeUserFacingError(err error) string {
	if err == nil {
		return "生成中断，请稍后重试。"
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
		return "请求超时，请稍后重试。"
	}
	if strings.Contains(msg, "unauthorized") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "permission") {
		return "权限不足，请联系管理员。"
	}
	if strings.Contains(msg, "rate limit") || strings.Contains(msg, "too many requests") {
		return "请求过于频繁，请稍后重试。"
	}
	return "生成中断，请稍后重试。"
}

// ShouldRetainPartialStreamSnapshot 判断是否应保留部分流快照。
func ShouldRetainPartialStreamSnapshot(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "timeout")
}

// EnsureDoneSummary 确保 done 事件包含 summary。
func EnsureDoneSummary(payload map[string]any, summary string, hasToolErrors bool) {
	if payload == nil {
		return
	}
	resolved := strings.TrimSpace(StringValue(payload, "summary"))
	if resolved == "" {
		resolved = strings.TrimSpace(summary)
	}
	if resolved == "" && hasToolErrors {
		resolved = "工具调用失败，未生成可用结论。请调整参数后重试。"
	}
	if resolved != "" {
		payload["summary"] = resolved
	}
}

// BuildAssistantFailureSnapshot 构建助手失败快照。
func BuildAssistantFailureSnapshot(summaryBody, assistantBody, publicError string) string {
	if strings.TrimSpace(assistantBody) != "" {
		return assistantBody
	}
	if strings.TrimSpace(summaryBody) != "" {
		return ""
	}
	return publicError
}

// EventAgentName 从事件负载提取 agent 名称。
func EventAgentName(payload any) string {
	data, _ := payload.(map[string]any)
	return StringValue(data, "agent")
}

// EventToolCallID 从事件负载提取 tool call ID。
func EventToolCallID(payload any) string {
	data, _ := payload.(map[string]any)
	return StringValue(data, "call_id")
}

// StringValue 从 map 提取字符串值。
func StringValue(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, _ := data[key].(string)
	return value
}

// IntValue 从 map 提取整数值。
func IntValue(data map[string]any, key string) int {
	if data == nil {
		return 0
	}
	switch value := data[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}

// BoolValue 从 map 提取布尔值。
func BoolValue(data map[string]any, key string) bool {
	if data == nil {
		return false
	}
	value, _ := data[key].(bool)
	return value
}

// StringSliceValue 从 map 提取字符串切片。
func StringSliceValue(data map[string]any, key string) []string {
	if data == nil {
		return nil
	}
	raw, ok := data[key].([]any)
	if !ok {
		if direct, ok := data[key].([]string); ok {
			return direct
		}
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

// MapValue 从 map 提取 map 值。
func MapValue(data map[string]any, key string) map[string]any {
	if data == nil {
		return nil
	}
	value, _ := data[key].(map[string]any)
	return value
}
