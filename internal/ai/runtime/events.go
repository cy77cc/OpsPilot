package runtime

import "time"

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
}

func NewRunStateEvent(status string, payload map[string]any) PublicStreamEvent {
	data := map[string]any{
		"status": status,
	}
	for key, value := range payload {
		data[key] = value
	}
	return PublicStreamEvent{
		Event: "run_state",
		Data:  data,
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
