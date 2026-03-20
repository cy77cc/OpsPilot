package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/google/uuid"
)

type ProjectionSummary struct {
	Title       string `json:"title"`
	ContentMode string `json:"content_mode"`
	Content     string `json:"content"`
}

type ProjectionToolResult struct {
	EventID         string `json:"event_id"`
	Status          string `json:"status"`
	Preview         string `json:"preview,omitempty"`
	ResultContentID string `json:"result_content_id,omitempty"`
}

type ProjectionExecutorItem struct {
	ID                 string                `json:"id"`
	Type               string                `json:"type"`
	ContentID          string                `json:"content_id,omitempty"`
	StartEventID       string                `json:"start_event_id,omitempty"`
	EndEventID         string                `json:"end_event_id,omitempty"`
	ToolCallID         string                `json:"tool_call_id,omitempty"`
	ToolName           string                `json:"tool_name,omitempty"`
	EventID            string                `json:"event_id,omitempty"`
	ArgumentsContentID string                `json:"arguments_content_id,omitempty"`
	Result             *ProjectionToolResult `json:"result,omitempty"`
}

type ProjectionBlock struct {
	ID           string                   `json:"id"`
	Type         string                   `json:"type"`
	Title        string                   `json:"title"`
	Agent        string                   `json:"agent,omitempty"`
	EventIDs     []string                 `json:"event_ids,omitempty"`
	Steps        []string                 `json:"steps,omitempty"`
	Data         map[string]any           `json:"data,omitempty"`
	StartEventID string                   `json:"start_event_id,omitempty"`
	EndEventID   string                   `json:"end_event_id,omitempty"`
	Lazy         bool                     `json:"lazy,omitempty"`
	Items        []ProjectionExecutorItem `json:"items,omitempty"`
}

type RunProjection struct {
	Version   int                `json:"version"`
	RunID     string             `json:"run_id"`
	SessionID string             `json:"session_id"`
	Status    string             `json:"status"`
	Summary   *ProjectionSummary `json:"summary,omitempty"`
	Blocks    []ProjectionBlock  `json:"blocks"`
}

func BuildProjection(events []model.AIRunEvent) (*RunProjection, []*model.AIRunContent, error) {
	if len(events) == 0 {
		return nil, nil, fmt.Errorf("no events")
	}

	projection := &RunProjection{
		Version:   1,
		RunID:     events[0].RunID,
		SessionID: events[0].SessionID,
		Status:    "running",
		Blocks:    make([]ProjectionBlock, 0, 4),
	}
	contents := make([]*model.AIRunContent, 0, 8)

	var currentExecutor *ProjectionBlock
	var textBuffer strings.Builder
	var textStartID string
	var textEndID string

	flushText := func() {
		if currentExecutor == nil || textBuffer.Len() == 0 {
			return
		}
		contentID := uuid.NewString()
		body := textBuffer.String()
		currentExecutor.Items = append(currentExecutor.Items, ProjectionExecutorItem{
			ID:           uuid.NewString(),
			Type:         "content",
			ContentID:    contentID,
			StartEventID: textStartID,
			EndEventID:   textEndID,
		})
		currentExecutor.EndEventID = textEndID
		contents = append(contents, &model.AIRunContent{
			ID:          contentID,
			RunID:       projection.RunID,
			SessionID:   projection.SessionID,
			ContentKind: "executor_content",
			Encoding:    "text",
			SummaryText: truncate(body, 200),
			BodyText:    body,
			SizeBytes:   int64(len(body)),
		})
		textBuffer.Reset()
		textStartID = ""
		textEndID = ""
	}

	for _, event := range events {
		switch EventType(event.EventType) {
		case EventTypeAgentHandoff:
			flushText()
			currentExecutor = nil
			payload, err := UnmarshalEventPayload(EventTypeAgentHandoff, event.PayloadJSON)
			if err != nil {
				return nil, nil, err
			}
			handoff := payload.(*AgentHandoffPayload)
			projection.Blocks = append(projection.Blocks, ProjectionBlock{
				ID:       blockID("handoff", len(projection.Blocks)+1),
				Type:     "agent_handoff",
				Title:    "任务转交",
				EventIDs: []string{event.ID},
				Data: map[string]any{
					"from":   handoff.From,
					"to":     handoff.To,
					"intent": handoff.Intent,
				},
			})
		case EventTypePlan:
			flushText()
			currentExecutor = nil
			payload, err := UnmarshalEventPayload(EventTypePlan, event.PayloadJSON)
			if err != nil {
				return nil, nil, err
			}
			plan := payload.(*PlanPayload)
			projection.Blocks = append(projection.Blocks, ProjectionBlock{
				ID:       blockID("plan", len(projection.Blocks)+1),
				Type:     "plan",
				Title:    "处理计划",
				EventIDs: []string{event.ID},
				Steps:    append([]string(nil), plan.Steps...),
			})
		case EventTypeReplan:
			flushText()
			currentExecutor = nil
			payload, err := UnmarshalEventPayload(EventTypeReplan, event.PayloadJSON)
			if err != nil {
				return nil, nil, err
			}
			replan := payload.(*ReplanPayload)
			projection.Blocks = append(projection.Blocks, ProjectionBlock{
				ID:       blockID("replan", len(projection.Blocks)+1),
				Type:     "replan",
				Title:    "重新规划",
				EventIDs: []string{event.ID},
				Steps:    append([]string(nil), replan.Steps...),
				Data: map[string]any{
					"iteration": replan.Iteration,
					"completed": replan.Completed,
					"is_final":  replan.IsFinal,
				},
			})
		case EventTypeDelta:
			payload, err := UnmarshalEventPayload(EventTypeDelta, event.PayloadJSON)
			if err != nil {
				return nil, nil, err
			}
			delta := payload.(*DeltaPayload)
			if strings.TrimSpace(delta.Agent) == "executor" {
				if currentExecutor == nil {
					currentExecutor = &ProjectionBlock{
						ID:           blockID("executor", len(projection.Blocks)+1),
						Type:         "executor",
						Title:        "执行过程",
						Agent:        "executor",
						StartEventID: event.ID,
						EndEventID:   event.ID,
						Lazy:         true,
						Items:        make([]ProjectionExecutorItem, 0, 4),
					}
					projection.Blocks = append(projection.Blocks, *currentExecutor)
					currentExecutor = &projection.Blocks[len(projection.Blocks)-1]
				}
				if textBuffer.Len() == 0 {
					textStartID = event.ID
				}
				textBuffer.WriteString(delta.Content)
				textEndID = event.ID
				currentExecutor.EndEventID = event.ID
			}
		case EventTypeToolCall:
			payload, err := UnmarshalEventPayload(EventTypeToolCall, event.PayloadJSON)
			if err != nil {
				return nil, nil, err
			}
			call := payload.(*ToolCallPayload)
			if strings.TrimSpace(call.Agent) != "executor" {
				continue
			}
			if currentExecutor == nil {
				currentExecutor = &ProjectionBlock{
					ID:           blockID("executor", len(projection.Blocks)+1),
					Type:         "executor",
					Title:        "执行过程",
					Agent:        "executor",
					StartEventID: event.ID,
					EndEventID:   event.ID,
					Lazy:         true,
					Items:        make([]ProjectionExecutorItem, 0, 4),
				}
				projection.Blocks = append(projection.Blocks, *currentExecutor)
				currentExecutor = &projection.Blocks[len(projection.Blocks)-1]
			}
			flushText()
			argsContentID := uuid.NewString()
			argsJSON, _ := json.Marshal(call.Arguments)
			contents = append(contents, &model.AIRunContent{
				ID:          argsContentID,
				RunID:       projection.RunID,
				SessionID:   projection.SessionID,
				ContentKind: "tool_arguments",
				Encoding:    "json",
				SummaryText: call.ToolName,
				BodyJSON:    string(argsJSON),
				SizeBytes:   int64(len(argsJSON)),
			})
			currentExecutor.Items = append(currentExecutor.Items, ProjectionExecutorItem{
				ID:                 uuid.NewString(),
				Type:               "tool_call",
				ToolCallID:         call.CallID,
				ToolName:           call.ToolName,
				EventID:            event.ID,
				ArgumentsContentID: argsContentID,
			})
			currentExecutor.EndEventID = event.ID
		case EventTypeToolResult:
			payload, err := UnmarshalEventPayload(EventTypeToolResult, event.PayloadJSON)
			if err != nil {
				return nil, nil, err
			}
			result := payload.(*ToolResultPayload)
			if currentExecutor == nil {
				continue
			}
			resultContentID := uuid.NewString()
			contents = append(contents, &model.AIRunContent{
				ID:          resultContentID,
				RunID:       projection.RunID,
				SessionID:   projection.SessionID,
				ContentKind: "tool_result",
				Encoding:    "text",
				SummaryText: truncate(result.Content, 200),
				BodyText:    result.Content,
				SizeBytes:   int64(len(result.Content)),
			})
			for i := len(currentExecutor.Items) - 1; i >= 0; i-- {
				if currentExecutor.Items[i].Type == "tool_call" && currentExecutor.Items[i].ToolCallID == result.CallID {
					currentExecutor.Items[i].Result = &ProjectionToolResult{
						EventID:         event.ID,
						Status:          result.Status,
						Preview:         truncate(result.Content, 200),
						ResultContentID: resultContentID,
					}
					break
				}
			}
			currentExecutor.EndEventID = event.ID
		case EventTypeDone:
			flushText()
			currentExecutor = nil
			payload, err := UnmarshalEventPayload(EventTypeDone, event.PayloadJSON)
			if err != nil {
				return nil, nil, err
			}
			done := payload.(*DonePayload)
			projection.Status = done.Status
			if strings.TrimSpace(done.Summary) != "" {
				projection.Summary = &ProjectionSummary{
					Title:       "结论",
					ContentMode: "inline",
					Content:     done.Summary,
				}
			}
		case EventTypeError:
			flushText()
			currentExecutor = nil
			payload, err := UnmarshalEventPayload(EventTypeError, event.PayloadJSON)
			if err != nil {
				return nil, nil, err
			}
			errPayload := payload.(*ErrorPayload)
			projection.Status = "failed_runtime"
			projection.Blocks = append(projection.Blocks, ProjectionBlock{
				ID:       blockID("error", len(projection.Blocks)+1),
				Type:     "error",
				Title:    "执行错误",
				EventIDs: []string{event.ID},
				Data: map[string]any{
					"message": errPayload.Message,
					"code":    errPayload.Code,
				},
			})
		}
	}

	flushText()
	return projection, contents, nil
}

func blockID(prefix string, index int) string {
	return fmt.Sprintf("block_%s_%d", prefix, index)
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
