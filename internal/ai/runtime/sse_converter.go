package runtime

import "strings"

type SSEConverter struct{}

func NewSSEConverter() *SSEConverter {
	return &SSEConverter{}
}

func (c *SSEConverter) OnChainStarted(turnID string) StreamEvent {
	return StreamEvent{Type: EventChainStarted, Data: compactMap(map[string]any{
		"turn_id": turnID,
	})}
}

func (c *SSEConverter) OnChainMeta(sessionID, planID, turnID, traceID string) StreamEvent {
	return StreamEvent{Type: EventChainMeta, Data: compactMap(map[string]any{
		"session_id": sessionID,
		"chain_id":   planID,
		"plan_id":    planID,
		"turn_id":    turnID,
		"trace_id":   traceID,
	})}
}

func (c *SSEConverter) OnChainPaused(turnID, reason string) StreamEvent {
	return StreamEvent{Type: EventChainPaused, Data: compactMap(map[string]any{
		"turn_id": turnID,
		"reason":  reason,
	})}
}

func (c *SSEConverter) OnChainResumed(turnID string) StreamEvent {
	return StreamEvent{Type: EventChainResumed, Data: compactMap(map[string]any{
		"turn_id": turnID,
	})}
}

func (c *SSEConverter) OnChainCompleted(turnID, status string) StreamEvent {
	return StreamEvent{Type: EventChainCompleted, Data: compactMap(map[string]any{
		"turn_id": turnID,
		"status":  status,
	})}
}

func (c *SSEConverter) OnChainError(turnID, stage, message string) StreamEvent {
	return StreamEvent{Type: EventChainError, Data: compactMap(map[string]any{
		"turn_id": turnID,
		"phase":   stage,
		"message": message,
	})}
}

func (c *SSEConverter) OnFinalAnswerStarted(turnID string) StreamEvent {
	return StreamEvent{Type: EventFinalAnswerStart, Data: compactMap(map[string]any{
		"turn_id": turnID,
	})}
}

func (c *SSEConverter) OnFinalAnswerDelta(turnID, chunk string) StreamEvent {
	return StreamEvent{Type: EventFinalAnswerDelta, Data: compactMap(map[string]any{
		"turn_id": turnID,
		"chunk":   chunk,
	})}
}

func (c *SSEConverter) OnFinalAnswerDone(turnID string) StreamEvent {
	return StreamEvent{Type: EventFinalAnswerDone, Data: compactMap(map[string]any{
		"turn_id": turnID,
	})}
}

func (c *SSEConverter) OnPlannerStart(sessionID, planID, turnID string) []StreamEvent {
	return []StreamEvent{
		{Type: EventMeta, Data: compactMap(map[string]any{
			"session_id": sessionID,
			"plan_id":    planID,
			"turn_id":    turnID,
		})},
	}
}

func (c *SSEConverter) OnTextDelta(chunk string) StreamEvent {
	return StreamEvent{Type: EventDelta, Data: map[string]any{"content_chunk": chunk}}
}

func (c *SSEConverter) OnDone(status string) StreamEvent {
	return StreamEvent{Type: EventDone, Data: map[string]any{"status": status}}
}

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
