// Package runtime 提供 AI 运行时的事件类型定义。
//
// 定义 SSE 流式事件类型和负载结构，用于前后端通信。
package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/ai/common/todo"
)

// EventType 事件类型枚举。
type EventType string

// 事件类型常量定义。
const (
	EventTypeMeta           EventType = "meta"             // 会话元数据
	EventTypeAgentHandoff   EventType = "agent_handoff"    // 代理切换
	EventTypePlan           EventType = "plan"             // 规划步骤
	EventTypeReplan         EventType = "replan"           // 重新规划
	EventTypeDelta          EventType = "delta"            // 文本增量
	EventTypeToolCall       EventType = "tool_call"        // 工具调用
	EventTypeToolApproval   EventType = "tool_approval"    // 工具审批请求
	EventTypeToolResult     EventType = "tool_result"      // 工具调用结果
	EventTypeOpsPlanUpdated EventType = "ops_plan_updated" // Ops 计划快照更新
	EventTypeRunState       EventType = "run_state"        // 运行状态变更
	EventTypeDone           EventType = "done"             // 执行完成
	EventTypeError          EventType = "error"            // 执行错误
)

// MetaPayload 会话元数据负载。
type MetaPayload struct {
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id"`
	Turn      int    `json:"turn"`
}

// AgentHandoffPayload 代理切换负载。
type AgentHandoffPayload struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Intent string `json:"intent"`
}

// PlanPayload 规划步骤负载。
type PlanPayload struct {
	Iteration int      `json:"iteration"`
	Steps     []string `json:"steps"`
}

// ReplanPayload 重新规划负载。
type ReplanPayload struct {
	Iteration int      `json:"iteration"`
	Completed int      `json:"completed"`
	IsFinal   bool     `json:"is_final"`
	Steps     []string `json:"steps"`
}

// DeltaPayload 文本增量负载。
type DeltaPayload struct {
	Agent   string `json:"agent"`
	Content string `json:"content"`
}

// ToolCallPayload 工具调用负载。
type ToolCallPayload struct {
	Agent     string         `json:"agent"`
	CallID    string         `json:"call_id"`
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolApprovalPayload 工具审批请求负载。
type ToolApprovalPayload struct {
	ApprovalID     string         `json:"approval_id"`
	TargetID       string         `json:"target_id,omitempty"`
	CallID         string         `json:"call_id"`
	ToolName       string         `json:"tool_name"`
	Preview        map[string]any `json:"preview,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
}

// ToolResultPayload 工具调用结果负载。
type ToolResultPayload struct {
	Agent    string `json:"agent"`
	CallID   string `json:"call_id"`
	ToolName string `json:"tool_name"`
	Content  string `json:"content"`
	Status   string `json:"status"`
}

// OpsPlanUpdatedPayload Ops 计划快照更新负载。
type OpsPlanUpdatedPayload struct {
	Todos []todo.OpsTODO `json:"todos"`
}

// RunStatePayload 运行状态负载。
type RunStatePayload struct {
	Status string `json:"status"`
	Agent  string `json:"agent,omitempty"`
}

// DonePayload 执行完成负载。
type DonePayload struct {
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
	Summary    string `json:"summary,omitempty"`
	Iterations int    `json:"iterations,omitempty"`
}

// ErrorPayload 执行错误负载。
type ErrorPayload struct {
	RunID   string `json:"run_id,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// MarshalEventPayload 序列化事件负载。
//
// 参数:
//   - eventType: 事件类型
//   - payload: 负载对象
//
// 返回: JSON 字符串或错误
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

// UnmarshalEventPayload 反序列化事件负载。
//
// 参数:
//   - eventType: 事件类型
//   - raw: JSON 字符串
//
// 返回: 负载对象或错误
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

// newPayloadTarget 根据事件类型创建负载对象。
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
	case EventTypeOpsPlanUpdated:
		return &OpsPlanUpdatedPayload{Todos: []todo.OpsTODO{}}, nil
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

// validatePayload 验证负载对象的有效性。
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
	case EventTypeOpsPlanUpdated:
		value, ok := payload.(*OpsPlanUpdatedPayload)
		if !ok {
			return nil, errors.New("ops_plan_updated payload type mismatch")
		}
		if value.Todos == nil {
			value.Todos = []todo.OpsTODO{}
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
