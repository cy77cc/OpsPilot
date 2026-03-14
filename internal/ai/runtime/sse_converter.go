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
		{Type: EventTurnState, Data: map[string]any{
			"turn_id": turnID,
			"plan_id": planID,
			"status":  "running",
		}},
	}
}

func (c *SSEConverter) OnApprovalRequired(pending *PendingApproval, checkpointID string) []StreamEvent {
	if pending == nil {
		return nil
	}
	return []StreamEvent{
		{Type: EventTurnState, Data: map[string]any{
			"plan_id": pending.PlanID,
			"step_id": pending.StepID,
			"status":  "waiting_approval",
		}},
		{Type: EventApprovalRequired, Data: map[string]any{
			"id":            pending.ID,
			"plan_id":       pending.PlanID,
			"step_id":       pending.StepID,
			"checkpoint_id": checkpointID,
			"title":         pending.Title,
			"tool_name":     pending.ToolName,
			"risk_level":    pending.Risk,
			"mode":          pending.Mode,
			"summary":       pending.Summary,
			"params":        pending.Params,
		}},
	}
}

func (c *SSEConverter) OnApprovalResult(stepID string, approved bool, reason string) []StreamEvent {
	status := "rejected"
	message := "审批未通过，待审批步骤不会继续执行。"
	if approved {
		status = "running"
		message = "审批已通过，待审批步骤会继续执行。"
	}
	if strings.TrimSpace(reason) != "" {
		message = fmt.Sprintf("%s 原因: %s", message, strings.TrimSpace(reason))
	}
	return []StreamEvent{
		{Type: EventTurnState, Data: map[string]any{
			"step_id":  stepID,
			"status":   status,
			"decision": map[bool]string{true: "approved", false: "rejected"}[approved],
			"message":  message,
		}},
	}
}

func (c *SSEConverter) OnTextDelta(chunk string) StreamEvent {
	return StreamEvent{Type: EventDelta, Data: map[string]any{"content_chunk": chunk}}
}

func (c *SSEConverter) OnExecuteComplete() []StreamEvent {
	return []StreamEvent{
		{Type: EventTurnState, Data: map[string]any{"status": "completed"}},
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
	return StreamEvent{Type: EventError, Data: map[string]any{"phase": stage, "message": message}}
}
