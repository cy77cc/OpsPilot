package runtime

import "github.com/cloudwego/eino/adk"

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
	Handoff   *NormalizedHandoff
	Raw       *adk.AgentEvent
}

type NormalizedHandoff struct {
	From string
	To   string
}

func NormalizeAgentEvent(event *adk.AgentEvent) []NormalizedEvent {
	if event == nil {
		return nil
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

	return nil
}
