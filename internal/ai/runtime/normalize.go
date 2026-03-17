package runtime

import (
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
		return []NormalizedEvent{{
			Kind:      NormalizedKindMessage,
			AgentName: event.AgentName,
			Message: &NormalizedMessage{
				Role:        string(message.Role),
				Content:     message.Content,
				IsStreaming: event.Output.MessageOutput.IsStreaming,
			},
			Raw: event,
		}}
	case schema.Tool:
		return []NormalizedEvent{{
			Kind:      NormalizedKindToolResult,
			AgentName: event.AgentName,
			Tool: &NormalizedTool{
				CallID:   message.ToolCallID,
				ToolName: message.ToolName,
				Content:  message.Content,
				Phase:    "result",
			},
			Raw: event,
		}}
	default:
		return nil
	}
}

func normalizeInterrupt(event *adk.AgentEvent) *NormalizedEvent {
	if event == nil || event.Action == nil || event.Action.Interrupted == nil {
		return nil
	}

	payload, ok := event.Action.Interrupted.Data.(map[string]any)
	if !ok || payload == nil {
		return nil
	}

	return &NormalizedEvent{
		Kind:      NormalizedKindInterrupt,
		AgentName: event.AgentName,
		Interrupt: &NormalizedInterrupt{
			Type:           "approval",
			ApprovalID:     stringValue(payload["approval_id"]),
			CallID:         stringValue(payload["call_id"]),
			ToolName:       stringValue(payload["tool_name"]),
			Preview:        mapValue(payload["preview"]),
			TimeoutSeconds: intValue(payload["timeout_seconds"]),
		},
		Raw: event,
	}
}
