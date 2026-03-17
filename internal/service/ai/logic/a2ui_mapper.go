package logic

import (
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type a2uiEvent struct {
	Event string
	Data  any
}

type a2uiProjectionState struct {
	totalPlanSteps int
	lastIterations int
}

func newMetaEvent(sessionID, runID string, turn int) a2uiEvent {
	return a2uiEvent{
		Event: "meta",
		Data: map[string]any{
			"session_id": sessionID,
			"run_id":     runID,
			"turn":       turn,
		},
	}
}

func projectAgentHandoff(event *adk.AgentEvent) *a2uiEvent {
	if event == nil || event.Action == nil || event.Action.TransferToAgent == nil {
		return nil
	}

	dest := strings.TrimSpace(event.Action.TransferToAgent.DestAgentName)
	if dest == "" {
		return nil
	}

	return &a2uiEvent{
		Event: "agent_handoff",
		Data: map[string]any{
			"from":   strings.TrimSpace(event.AgentName),
			"to":     dest,
			"intent": mapAgentNameToIntentType(dest),
		},
	}
}

func projectAgentEvent(event *adk.AgentEvent, state *a2uiProjectionState) []a2uiEvent {
	if event == nil {
		return nil
	}

	projected := make([]a2uiEvent, 0, 4)
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

func projectMessageVariant(agentName string, variant *adk.MessageVariant, state *a2uiProjectionState) []a2uiEvent {
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

func projectAssistantMessage(agentName string, message *schema.Message, state *a2uiProjectionState) []a2uiEvent {
	if message == nil {
		return nil
	}

	trimmedAgent := strings.TrimSpace(agentName)
	trimmedContent := strings.TrimSpace(message.Content)

	if trimmedAgent == "planner" {
		if steps, ok := decodeStepsEnvelope(trimmedContent); ok {
			state.totalPlanSteps = len(steps)
			state.lastIterations = 0
			return []a2uiEvent{{
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
			state.lastIterations++
			return []a2uiEvent{
				{
					Event: "replan",
					Data: map[string]any{
						"steps":     []string{},
						"completed": state.totalPlanSteps,
						"iteration": state.lastIterations,
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
			state.lastIterations++
			completed := state.totalPlanSteps - len(steps)
			if completed < 0 {
				completed = 0
			}
			return []a2uiEvent{{
				Event: "replan",
				Data: map[string]any{
					"steps":     steps,
					"completed": completed,
					"iteration": state.lastIterations,
					"is_final":  false,
				},
			}}
		}
	}

	projected := make([]a2uiEvent, 0, len(message.ToolCalls)+1)
	if trimmedContent != "" {
		projected = append(projected, a2uiEvent{
			Event: "delta",
			Data: map[string]any{
				"content": message.Content,
				"agent":   trimmedAgent,
			},
		})
	}

	for _, toolCall := range message.ToolCalls {
		projected = append(projected, a2uiEvent{
			Event: "tool_call",
			Data: map[string]any{
				"call_id":   toolCall.ID,
				"tool_name": toolCall.Function.Name,
				"arguments": decodeToolArguments(toolCall.Function.Arguments),
			},
		})
	}

	return projected
}

func projectToolMessage(message *schema.Message) []a2uiEvent {
	if message == nil || strings.TrimSpace(message.ToolName) == "transfer_to_agent" {
		return nil
	}

	return []a2uiEvent{{
		Event: "tool_result",
		Data: map[string]any{
			"call_id":   message.ToolCallID,
			"tool_name": message.ToolName,
			"content":   message.Content,
		},
	}}
}

func projectApprovalEvent(event *adk.AgentEvent) *a2uiEvent {
	if event == nil || event.Action == nil || event.Action.Interrupted == nil {
		return nil
	}

	payload, ok := event.Action.Interrupted.Data.(map[string]any)
	if !ok || payload == nil {
		return nil
	}

	return &a2uiEvent{
		Event: "tool_approval",
		Data: map[string]any{
			"approval_id":     stringValue(payload["approval_id"]),
			"call_id":         stringValue(payload["call_id"]),
			"tool_name":       stringValue(payload["tool_name"]),
			"preview":         mapValue(payload["preview"]),
			"timeout_seconds": intValue(payload["timeout_seconds"]),
		},
	}
}

func doneEvent(runID string, iterations int) a2uiEvent {
	return a2uiEvent{
		Event: "done",
		Data: map[string]any{
			"run_id":     runID,
			"status":     "completed",
			"iterations": iterations,
		},
	}
}

func errorEvent(runID string, err error) a2uiEvent {
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
	return a2uiEvent{Event: "error", Data: payload}
}

func decodeStepsEnvelope(raw string) ([]string, bool) {
	var payload struct {
		Steps []string `json:"steps"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || len(payload.Steps) == 0 {
		return nil, false
	}
	return payload.Steps, true
}

func decodeResponseEnvelope(raw string) (string, bool) {
	var payload struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || strings.TrimSpace(payload.Response) == "" {
		return "", false
	}
	return payload.Response, true
}

func decodeToolArguments(raw string) map[string]any {
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
