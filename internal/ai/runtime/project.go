package runtime

import "strings"

type ProjectionState struct {
	TotalPlanSteps  int
	ReplanIteration int
	RunPhase        string
}

func projectNormalizedEvents(events []NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	projected := make([]PublicStreamEvent, 0, len(events))
	for _, event := range events {
		projected = append(projected, projectNormalizedEvent(event, state)...)
	}
	return projected
}

func projectNormalizedEvent(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	switch event.Kind {
	case NormalizedKindHandoff:
		if event.Handoff == nil {
			return nil
		}
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
	case NormalizedKindMessage:
		return projectNormalizedMessage(event, state)
	default:
		return nil
	}
}

func projectNormalizedMessage(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	if event.Message == nil {
		return nil
	}

	trimmedAgent := strings.TrimSpace(event.AgentName)
	trimmedContent := strings.TrimSpace(event.Message.Content)

	if trimmedAgent == "planner" {
		if steps, ok := decodeStepsEnvelope(trimmedContent); ok {
			state.TotalPlanSteps = len(steps)
			state.ReplanIteration = 0
			state.RunPhase = "planning"
			return []PublicStreamEvent{{
				Event: "plan",
				Data: map[string]any{
					"steps":     steps,
					"iteration": 0,
				},
			}}
		}
	}

	if trimmedAgent == "replanner" {
		if response, ok := decodeResponseEnvelope(trimmedContent); ok {
			state.ReplanIteration++
			state.RunPhase = "executing"
			return []PublicStreamEvent{
				{
					Event: "replan",
					Data: map[string]any{
						"steps":     []string{},
						"completed": state.TotalPlanSteps,
						"iteration": state.ReplanIteration,
						"is_final":  true,
					},
				},
				{
					Event: "delta",
					Data: map[string]any{
						"content": response,
						"agent":   trimmedAgent,
					},
				},
			}
		}
		if steps, ok := decodeStepsEnvelope(trimmedContent); ok {
			state.ReplanIteration++
			completed := state.TotalPlanSteps - len(steps)
			if completed < 0 {
				completed = 0
			}
			state.RunPhase = "planning"
			return []PublicStreamEvent{{
				Event: "replan",
				Data: map[string]any{
					"steps":     steps,
					"completed": completed,
					"iteration": state.ReplanIteration,
					"is_final":  false,
				},
			}}
		}
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
