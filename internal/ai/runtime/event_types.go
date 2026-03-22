package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type EventType string

const (
	EventTypeMeta         EventType = "meta"
	EventTypeAgentHandoff EventType = "agent_handoff"
	EventTypePlan         EventType = "plan"
	EventTypeReplan       EventType = "replan"
	EventTypeDelta        EventType = "delta"
	EventTypeToolCall     EventType = "tool_call"
	EventTypeToolApproval EventType = "tool_approval"
	EventTypeToolResult   EventType = "tool_result"
	EventTypeRunState     EventType = "run_state"
	EventTypeDone         EventType = "done"
	EventTypeError        EventType = "error"
)

type MetaPayload struct {
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id"`
	Turn      int    `json:"turn"`
}

type AgentHandoffPayload struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Intent string `json:"intent"`
}

type PlanPayload struct {
	Iteration int      `json:"iteration"`
	Steps     []string `json:"steps"`
}

type ReplanPayload struct {
	Iteration int      `json:"iteration"`
	Completed int      `json:"completed"`
	IsFinal   bool     `json:"is_final"`
	Steps     []string `json:"steps"`
}

type DeltaPayload struct {
	Agent   string `json:"agent"`
	Content string `json:"content"`
}

type ToolCallPayload struct {
	Agent     string         `json:"agent"`
	CallID    string         `json:"call_id"`
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments"`
}

type ToolApprovalPayload struct {
	ApprovalID     string         `json:"approval_id"`
	CallID         string         `json:"call_id"`
	ToolName       string         `json:"tool_name"`
	Preview        map[string]any `json:"preview,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
}

type ToolResultPayload struct {
	Agent    string `json:"agent"`
	CallID   string `json:"call_id"`
	ToolName string `json:"tool_name"`
	Content  string `json:"content"`
	Status   string `json:"status"`
}

type RunStatePayload struct {
	Status string `json:"status"`
	Agent  string `json:"agent,omitempty"`
}

type DonePayload struct {
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
	Summary    string `json:"summary,omitempty"`
	Iterations int    `json:"iterations,omitempty"`
}

type ErrorPayload struct {
	RunID   string `json:"run_id,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func MarshalEventPayload(eventType EventType, payload any) (string, error) {
	if _, err := validatePayload(eventType, payload); err != nil {
		return "", err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func UnmarshalEventPayload(eventType EventType, raw string) (any, error) {
	target, err := newPayloadTarget(eventType)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return nil, err
	}
	return validatePayload(eventType, target)
}

func newPayloadTarget(eventType EventType) (any, error) {
	switch eventType {
	case EventTypeMeta:
		return &MetaPayload{}, nil
	case EventTypeAgentHandoff:
		return &AgentHandoffPayload{}, nil
	case EventTypePlan:
		return &PlanPayload{}, nil
	case EventTypeReplan:
		return &ReplanPayload{}, nil
	case EventTypeDelta:
		return &DeltaPayload{}, nil
	case EventTypeToolCall:
		return &ToolCallPayload{}, nil
	case EventTypeToolApproval:
		return &ToolApprovalPayload{}, nil
	case EventTypeToolResult:
		return &ToolResultPayload{}, nil
	case EventTypeRunState:
		return &RunStatePayload{}, nil
	case EventTypeDone:
		return &DonePayload{}, nil
	case EventTypeError:
		return &ErrorPayload{}, nil
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}
}

func validatePayload(eventType EventType, payload any) (any, error) {
	switch eventType {
	case EventTypeMeta:
		value, ok := payload.(*MetaPayload)
		if !ok {
			return nil, errors.New("meta payload type mismatch")
		}
		if strings.TrimSpace(value.RunID) == "" || strings.TrimSpace(value.SessionID) == "" || value.Turn <= 0 {
			return nil, errors.New("invalid meta payload")
		}
		return value, nil
	case EventTypeAgentHandoff:
		value, ok := payload.(*AgentHandoffPayload)
		if !ok {
			return nil, errors.New("agent_handoff payload type mismatch")
		}
		if strings.TrimSpace(value.From) == "" || strings.TrimSpace(value.To) == "" {
			return nil, errors.New("invalid agent_handoff payload")
		}
		return value, nil
	case EventTypePlan:
		value, ok := payload.(*PlanPayload)
		if !ok {
			return nil, errors.New("plan payload type mismatch")
		}
		if len(value.Steps) == 0 {
			return nil, errors.New("invalid plan payload")
		}
		return value, nil
	case EventTypeReplan:
		value, ok := payload.(*ReplanPayload)
		if !ok {
			return nil, errors.New("replan payload type mismatch")
		}
		if len(value.Steps) == 0 && !value.IsFinal {
			return nil, errors.New("invalid replan payload")
		}
		return value, nil
	case EventTypeDelta:
		value, ok := payload.(*DeltaPayload)
		if !ok {
			return nil, errors.New("delta payload type mismatch")
		}
		if strings.TrimSpace(value.Content) == "" {
			return nil, errors.New("invalid delta payload")
		}
		return value, nil
	case EventTypeToolCall:
		value, ok := payload.(*ToolCallPayload)
		if !ok {
			return nil, errors.New("tool_call payload type mismatch")
		}
		if strings.TrimSpace(value.CallID) == "" || strings.TrimSpace(value.ToolName) == "" {
			return nil, errors.New("invalid tool_call payload")
		}
		if value.Arguments == nil {
			value.Arguments = map[string]any{}
		}
		return value, nil
	case EventTypeToolApproval:
		value, ok := payload.(*ToolApprovalPayload)
		if !ok {
			return nil, errors.New("tool_approval payload type mismatch")
		}
		if strings.TrimSpace(value.ApprovalID) == "" || strings.TrimSpace(value.CallID) == "" || strings.TrimSpace(value.ToolName) == "" {
			return nil, errors.New("invalid tool_approval payload")
		}
		if value.Preview == nil {
			value.Preview = map[string]any{}
		}
		return value, nil
	case EventTypeToolResult:
		value, ok := payload.(*ToolResultPayload)
		if !ok {
			return nil, errors.New("tool_result payload type mismatch")
		}
		if strings.TrimSpace(value.CallID) == "" || strings.TrimSpace(value.ToolName) == "" {
			return nil, errors.New("invalid tool_result payload")
		}
		return value, nil
	case EventTypeRunState:
		value, ok := payload.(*RunStatePayload)
		if !ok {
			return nil, errors.New("run_state payload type mismatch")
		}
		if strings.TrimSpace(value.Status) == "" {
			return nil, errors.New("invalid run_state payload")
		}
		return value, nil
	case EventTypeDone:
		value, ok := payload.(*DonePayload)
		if !ok {
			return nil, errors.New("done payload type mismatch")
		}
		if strings.TrimSpace(value.Status) == "" {
			return nil, errors.New("invalid done payload")
		}
		return value, nil
	case EventTypeError:
		value, ok := payload.(*ErrorPayload)
		if !ok {
			return nil, errors.New("error payload type mismatch")
		}
		if strings.TrimSpace(value.Message) == "" {
			return nil, errors.New("invalid error payload")
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}
}
