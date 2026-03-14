package runtime

import (
	"fmt"
	"strings"
)

type SSEConverter struct{}

func NewSSEConverter() *SSEConverter {
	return &SSEConverter{}
}

func (c *SSEConverter) OnChainStarted(turnID string) StreamEvent {
	return StreamEvent{Type: EventChainStarted, Data: compactMap(map[string]any{
		"turn_id": turnID,
	})}
}

func (c *SSEConverter) OnChainNodeOpen(payload ChainNodeInfo) StreamEvent {
	return StreamEvent{Type: EventChainNodeOpen, Data: chainNodeData(payload)}
}

func (c *SSEConverter) OnChainNodePatch(payload ChainNodeInfo) StreamEvent {
	return StreamEvent{Type: EventChainNodePatch, Data: chainNodeData(payload)}
}

func (c *SSEConverter) OnChainNodeClose(payload ChainNodeInfo) StreamEvent {
	return StreamEvent{Type: EventChainNodeClose, Data: chainNodeData(payload)}
}

func (c *SSEConverter) OnChainCollapsed(turnID string) StreamEvent {
	return StreamEvent{Type: EventChainCollapsed, Data: compactMap(map[string]any{
		"turn_id": turnID,
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
		{Type: EventTurnStarted, Data: map[string]any{"turn_id": turnID, "session_id": sessionID}},
		c.OnPhaseStarted(PhaseEvent{
			Phase:  PhasePlanning,
			PlanID: planID,
			TurnID: turnID,
			Status: "loading",
			Title:  "整理执行步骤",
		}),
		{Type: EventTurnState, Data: map[string]any{
			"turn_id": turnID,
			"plan_id": planID,
			"status":  "running",
		}},
	}
}

func (c *SSEConverter) OnPhaseStarted(payload PhaseEvent) StreamEvent {
	return StreamEvent{Type: EventPhaseStarted, Data: compactMap(map[string]any{
		"phase":    string(payload.Phase),
		"plan_id":  payload.PlanID,
		"turn_id":  payload.TurnID,
		"status":   payload.Status,
		"title":    payload.Title,
		"summary":  payload.Summary,
		"reason":   payload.Reason,
		"message":  payload.Message,
		"metadata": payload.Metadata,
	})}
}

func (c *SSEConverter) OnPhaseComplete(payload PhaseEvent) StreamEvent {
	return StreamEvent{Type: EventPhaseComplete, Data: compactMap(map[string]any{
		"phase":    string(payload.Phase),
		"plan_id":  payload.PlanID,
		"turn_id":  payload.TurnID,
		"status":   payload.Status,
		"title":    payload.Title,
		"summary":  payload.Summary,
		"reason":   payload.Reason,
		"message":  payload.Message,
		"metadata": payload.Metadata,
	})}
}

func (c *SSEConverter) OnPlanGenerated(payload PlanEvent) StreamEvent {
	steps := make([]map[string]any, 0, len(payload.Steps))
	for _, step := range payload.Steps {
		stepData := compactMap(map[string]any{
			"id":        step.ID,
			"title":     step.Title,
			"content":   step.Content,
			"status":    step.Status,
			"tool_hint": step.ToolHint,
			"metadata":  step.Metadata,
		})
		if step.Tool != nil {
			stepData["tool"] = compactMap(map[string]any{
				"name":         step.Tool.Name,
				"display_name": step.Tool.DisplayName,
				"args":         step.Tool.Args,
				"mode":         step.Tool.Mode,
				"risk":         step.Tool.Risk,
			})
		}
		steps = append(steps, stepData)
	}
	return StreamEvent{Type: EventPlanGenerated, Data: compactMap(map[string]any{
		"plan_id":  payload.PlanID,
		"turn_id":  payload.TurnID,
		"source":   payload.Source,
		"summary":  payload.Summary,
		"raw":      payload.Raw,
		"steps":    steps,
		"total":    len(steps),
		"metadata": payload.Metadata,
	})}
}

func (c *SSEConverter) OnStepStarted(payload StepEvent) StreamEvent {
	return StreamEvent{Type: EventStepStarted, Data: stepEventData(payload)}
}

func (c *SSEConverter) OnStepComplete(payload StepEvent) StreamEvent {
	return StreamEvent{Type: EventStepComplete, Data: stepEventData(payload)}
}

func (c *SSEConverter) OnReplanTriggered(payload ReplanEvent) StreamEvent {
	return StreamEvent{Type: EventReplanTriggered, Data: compactMap(map[string]any{
		"plan_id":          payload.PlanID,
		"turn_id":          payload.TurnID,
		"previous_plan_id": payload.PreviousPlanID,
		"reason":           payload.Reason,
		"summary":          payload.Summary,
		"metadata":         payload.Metadata,
	})}
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
		c.OnPhaseComplete(PhaseEvent{
			Phase:  PhaseExecuting,
			Status: "success",
			Title:  "执行步骤",
		}),
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

func stepEventData(payload StepEvent) map[string]any {
	data := compactMap(map[string]any{
		"plan_id":  payload.PlanID,
		"turn_id":  payload.TurnID,
		"step_id":  payload.StepID,
		"title":    payload.Title,
		"content":  payload.Content,
		"status":   payload.Status,
		"expert":   payload.Expert,
		"summary":  payload.Summary,
		"result":   payload.Result,
		"error":    payload.Error,
		"metadata": payload.Metadata,
	})
	if payload.Tool != nil {
		data["tool_name"] = payload.Tool.Name
		data["params"] = payload.Tool.Args
		data["tool"] = compactMap(map[string]any{
			"name":         payload.Tool.Name,
			"display_name": payload.Tool.DisplayName,
			"args":         payload.Tool.Args,
			"mode":         payload.Tool.Mode,
			"risk":         payload.Tool.Risk,
		})
	}
	return data
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

func chainNodeData(payload ChainNodeInfo) map[string]any {
	return compactMap(map[string]any{
		"turn_id":     payload.TurnID,
		"node_id":     payload.NodeID,
		"kind":        string(payload.Kind),
		"title":       payload.Title,
		"status":      payload.Status,
		"summary":     payload.Summary,
		"details":     payload.Details,
		"approval":    payload.Approval,
		"started_at":  payload.StartedAt,
		"finished_at": payload.FinishedAt,
	})
}
