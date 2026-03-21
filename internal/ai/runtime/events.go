package runtime

import (
	"maps"
	"time"
)

type PublicStreamEvent = StreamEvent

type EventEnvelope struct {
	Version   string
	Type      string
	Timestamp time.Time
	RunID     string
	AgentName string
}

type a2uiProjectionState struct {
	totalPlanSteps int
	lastIterations int
	currentSteps   []string
}

func NewRunStateEvent(status string, payload map[string]any) PublicStreamEvent {
	data := map[string]any{
		"status": status,
	}
	maps.Copy(data, payload)
	return PublicStreamEvent{
		Event: "run_state",
		Data:  data,
	}
}

func NewToolResultEvent(callID, toolName, content, status, agentName string) PublicStreamEvent {
	return PublicStreamEvent{
		Event: "tool_result",
		Data: map[string]any{
			"call_id":   callID,
			"tool_name": toolName,
			"content":   content,
			"status":    status,
			"agent":     agentName,
		},
	}
}

func mapAgentNameToIntentType(agentName string) string {
	switch agentName {
	case "QAAgent":
		return "qa"
	case "DiagnosisAgent":
		return "diagnosis"
	case "ChangeAgent":
		return "change"
	default:
		return "unknown"
	}
}
