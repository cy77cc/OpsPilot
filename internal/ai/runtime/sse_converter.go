package runtime

import (
	"fmt"
	"strings"
)

type SSEConverter struct{}

func NewSSEConverter() *SSEConverter {
	return &SSEConverter{}
}

func (c *SSEConverter) OnPlannerStart(sessionID, planID, turnID string) []StreamEvent {
	return []StreamEvent{
		{Type: EventTurnStarted, Data: map[string]any{"turn_id": turnID, "session_id": sessionID}},
		{Type: EventStageDelta, Data: map[string]any{"stage": "plan", "status": "loading", "plan_id": planID}},
	}
}

func (c *SSEConverter) OnPlanCreated(planID, content string) StreamEvent {
	return StreamEvent{Type: EventStageDelta, Data: map[string]any{
		"stage":   "plan",
		"status":  "success",
		"plan_id": planID,
		"content": strings.TrimSpace(content),
	}}
}

func (c *SSEConverter) OnToolCallStart(stepID, title, toolName string) StreamEvent {
	return StreamEvent{Type: EventStepUpdate, Data: map[string]any{
		"step_id": stepID,
		"title":   title,
		"tool":    toolName,
		"status":  "loading",
	}}
}

func (c *SSEConverter) OnToolResult(stepID, status, result string) StreamEvent {
	return StreamEvent{Type: EventStepUpdate, Data: map[string]any{
		"step_id": stepID,
		"status":  status,
		"result":  strings.TrimSpace(result),
	}}
}

func (c *SSEConverter) OnApprovalRequired(pending *PendingApproval, checkpointID string) []StreamEvent {
	if pending == nil {
		return nil
	}
	return []StreamEvent{
		{Type: EventStageDelta, Data: map[string]any{"stage": "user_action", "status": "loading"}},
		{Type: EventApprovalRequired, Data: map[string]any{
			"id":            pending.ID,
			"plan_id":       pending.PlanID,
			"step_id":       pending.StepID,
			"checkpoint_id": checkpointID,
			"tool_name":     pending.ToolName,
			"risk_level":    pending.Risk,
			"mode":          pending.Mode,
			"summary":       pending.Summary,
			"params":        pending.Params,
		}},
	}
}

func (c *SSEConverter) OnApprovalResult(stepID string, approved bool, reason string) []StreamEvent {
	status := "abort"
	message := "审批未通过，待审批步骤不会继续执行。"
	if approved {
		status = "success"
		message = "审批已通过，待审批步骤会继续执行。"
	}
	if strings.TrimSpace(reason) != "" {
		message = fmt.Sprintf("%s 原因: %s", message, strings.TrimSpace(reason))
	}
	return []StreamEvent{
		{Type: EventStageDelta, Data: map[string]any{"stage": "user_action", "status": status}},
		{Type: EventStepUpdate, Data: map[string]any{"step_id": stepID, "status": status, "message": message}},
	}
}

func (c *SSEConverter) OnTextDelta(chunk string) StreamEvent {
	return StreamEvent{Type: EventDelta, Data: map[string]any{"content_chunk": chunk}}
}

func (c *SSEConverter) OnExecuteComplete() []StreamEvent {
	return []StreamEvent{
		{Type: EventTurnState, Data: map[string]any{"status": "completed"}},
		{Type: EventStageDelta, Data: map[string]any{"stage": "execute", "status": "success"}},
	}
}

func (c *SSEConverter) OnDone(status string) StreamEvent {
	return StreamEvent{Type: EventDone, Data: map[string]any{"status": status}}
}

func (c *SSEConverter) OnError(stage string, err error) StreamEvent {
	message := ""
	if err != nil {
		message = err.Error()
	}
	return StreamEvent{Type: EventError, Data: map[string]any{"stage": stage, "message": message}}
}
