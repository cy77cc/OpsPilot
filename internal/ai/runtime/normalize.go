package runtime

import (
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type NormalizedKind string

const (
	NormalizedKindMessage    NormalizedKind = "message"
	NormalizedKindToolCall   NormalizedKind = "tool_call"
	NormalizedKindToolResult NormalizedKind = "tool_result"
	NormalizedKindHandoff    NormalizedKind = "handoff"
	NormalizedKindInterrupt  NormalizedKind = "interrupt"
	NormalizedKindError      NormalizedKind = "error"
)

type NormalizedEvent struct {
	Kind      NormalizedKind
	AgentName string
	Message   *NormalizedMessage
	Tool      *NormalizedTool
	Handoff   *NormalizedHandoff
	Interrupt *NormalizedInterrupt
	Raw       *adk.AgentEvent
}

type NormalizedMessage struct {
	Role        string
	Content     string
	IsStreaming bool
}

type NormalizedTool struct {
	CallID    string
	ToolName  string
	Arguments map[string]any
	Content   string
	Phase     string
}

type NormalizedHandoff struct {
	From string
	To   string
}

type NormalizedInterrupt struct {
	Type           string
	ApprovalID     string
	CallID         string
	ToolName       string
	Preview        map[string]any
	TimeoutSeconds int
}

func NormalizeAgentEvent(event *adk.AgentEvent) []NormalizedEvent {
	if event == nil {
		return nil
	}

	if interrupt := normalizeInterrupt(event); interrupt != nil {
		return []NormalizedEvent{*interrupt}
	}

	if event.Action != nil && event.Action.TransferToAgent != nil {
		return []NormalizedEvent{{
			Kind:      NormalizedKindHandoff,
			AgentName: event.AgentName,
			Handoff: &NormalizedHandoff{
				From: event.AgentName,
				To:   event.Action.TransferToAgent.DestAgentName,
			},
			Raw: event,
		}}
	}

	if normalized := normalizeMessageOutput(event); len(normalized) > 0 {
		return normalized
	}

	return nil
}

func normalizeMessageOutput(event *adk.AgentEvent) []NormalizedEvent {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return nil
	}

	message, err := event.Output.MessageOutput.GetMessage()
	if err != nil || message == nil {
		return nil
	}

	switch message.Role {
	case schema.Assistant:
		// 收集所有事件（消息内容 + 工具调用）
		events := make([]NormalizedEvent, 0, len(message.ToolCalls)+1)

		// 如果有内容，先添加消息事件
		if strings.TrimSpace(message.Content) != "" {
			events = append(events, NormalizedEvent{
				Kind:      NormalizedKindMessage,
				AgentName: event.AgentName,
				Message: &NormalizedMessage{
					Role:        string(message.Role),
					Content:     message.Content,
					IsStreaming: event.Output.MessageOutput.IsStreaming,
				},
				Raw: event,
			})
		}

		// 提取 ToolCalls
		for _, toolCall := range message.ToolCalls {
			if !shouldProjectToolCall(toolCall.ID, toolCall.Function.Name, toolCall.Function.Arguments) {
				continue
			}
			events = append(events, NormalizedEvent{
				Kind:      NormalizedKindToolCall,
				AgentName: event.AgentName,
				Tool: &NormalizedTool{
					CallID:    toolCall.ID,
					ToolName:  toolCall.Function.Name,
					Arguments: decodeToolArguments(toolCall.Function.Arguments),
					Phase:     "call",
				},
				Raw: event,
			})
		}

		return events
	case schema.Tool:
		phase := normalizeToolResultPhase(event.Err, message.Content)
		return []NormalizedEvent{{
			Kind:      NormalizedKindToolResult,
			AgentName: event.AgentName,
			Tool: &NormalizedTool{
				CallID:   message.ToolCallID,
				ToolName: message.ToolName,
				Content:  message.Content,
				Phase:    phase,
			},
			Raw: event,
		}}
	default:
		return nil
	}
}

func shouldProjectToolCall(callID, toolName, rawArguments string) bool {
	if strings.TrimSpace(toolName) == "transfer_to_agent" {
		return false
	}
	return strings.TrimSpace(callID) != "" || strings.TrimSpace(toolName) != "" || strings.TrimSpace(rawArguments) != ""
}

func normalizeToolResultPhase(err error, content string) string {
	if err != nil {
		return "error"
	}

	var payload map[string]any
	if json.Unmarshal([]byte(content), &payload) != nil || payload == nil {
		return "result"
	}

	if status, ok := payload["status"].(string); ok && strings.EqualFold(strings.TrimSpace(status), "error") {
		return "error"
	}
	if okValue, ok := payload["ok"].(bool); ok && !okValue {
		return "error"
	}

	return "result"
}

func normalizeInterrupt(event *adk.AgentEvent) *NormalizedEvent {
	if event == nil || event.Action == nil || event.Action.Interrupted == nil {
		return nil
	}

	interrupt, ok := parseApprovalInterrupt(event.Action.Interrupted)
	if !ok {
		return nil
	}

	return &NormalizedEvent{
		Kind:      NormalizedKindInterrupt,
		AgentName: event.AgentName,
		Interrupt: interrupt,
		Raw: event,
	}
}

func parseApprovalInterrupt(interrupted *adk.InterruptInfo) (*NormalizedInterrupt, bool) {
	payload, ok := extractInterruptPayload(interrupted)
	if !ok {
		return nil, false
	}

	toolName := firstNonEmptyString(
		stringValue(payload["tool_name"]),
		stringValue(payload["toolName"]),
		stringValue(payload["tool"]),
		stringValue(payload["name"]),
	)
	callID := firstNonEmptyString(
		stringValue(payload["call_id"]),
		stringValue(payload["callId"]),
		interruptContextID(interrupted),
	)
	approvalID := firstNonEmptyString(
		stringValue(payload["approval_id"]),
		stringValue(payload["approvalId"]),
		callID,
	)
	if strings.TrimSpace(toolName) == "" || strings.TrimSpace(callID) == "" {
		return nil, false
	}

	timeoutSeconds := intValue(payload["timeout_seconds"])
	if timeoutSeconds == 0 {
		timeoutSeconds = intValue(payload["timeoutSeconds"])
	}
	if timeoutSeconds == 0 {
		timeoutSeconds = intValue(payload["timeout"])
	}

	return &NormalizedInterrupt{
		Type:           "approval",
		ApprovalID:     approvalID,
		CallID:         callID,
		ToolName:       toolName,
		Preview:        mapValue(payload["preview"]),
		TimeoutSeconds: timeoutSeconds,
	}, true
}

func extractInterruptPayload(interrupted *adk.InterruptInfo) (map[string]any, bool) {
	if interrupted == nil {
		return nil, false
	}
	if payload, ok := normalizeInterruptPayloadMap(interrupted.Data); ok {
		return payload, true
	}

	var first map[string]any
	for _, ctx := range interrupted.InterruptContexts {
		if ctx == nil {
			continue
		}
		payload, ok := normalizeInterruptPayloadMap(ctx.Info)
		if !ok {
			continue
		}
		if ctx.IsRootCause {
			return payload, true
		}
		if first == nil {
			first = payload
		}
	}
	if first != nil {
		return first, true
	}
	return nil, false
}

func interruptContextID(interrupted *adk.InterruptInfo) string {
	if interrupted == nil {
		return ""
	}
	var first string
	for _, ctx := range interrupted.InterruptContexts {
		if ctx == nil {
			continue
		}
		id := strings.TrimSpace(ctx.ID)
		if id == "" {
			continue
		}
		if ctx.IsRootCause {
			return id
		}
		if first == "" {
			first = id
		}
	}
	return first
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func normalizeInterruptPayloadMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	if payload, ok := v.(map[string]any); ok && payload != nil {
		return payload, true
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return nil, false
	}
	return out, true
}
