package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (l *Logic) ensureChatShell(ctx context.Context, input ChatInput) (ChatShell, error) {
	shell := ChatShell{}
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	session, err := l.ChatDAO.GetSession(ctx, sessionID, input.UserID, input.Scene)
	if err != nil {
		return shell, fmt.Errorf("get session: %w", err)
	}
	scene := resolveChatScene(input.Scene, session)
	if session == nil {
		session = &ai.AIChatSession{
			ID:     sessionID,
			UserID: input.UserID,
			Scene:  scene,
			Title:  buildSessionTitle(input.Message),
		}
		if err := l.ChatDAO.CreateSession(ctx, session); err != nil {
			return shell, fmt.Errorf("create session: %w", err)
		}
	}

	var (
		createdUser      *ai.AIChatMessage
		createdAssistant *ai.AIChatMessage
	)
	run, created, err := l.RunDAO.CreateOrReuseRunShell(ctx, input.UserID, sessionID, input.ClientRequestID, func() (*ai.AIRun, *ai.AIChatMessage, *ai.AIChatMessage) {
		userMessageID := uuid.NewString()
		assistantMessageID := uuid.NewString()
		createdUser = &ai.AIChatMessage{
			ID:        userMessageID,
			SessionID: sessionID,
			Role:      "user",
			Content:   input.Message,
			Status:    "done",
		}
		createdAssistant = &ai.AIChatMessage{
			ID:        assistantMessageID,
			SessionID: sessionID,
			Role:      "assistant",
			Content:   "",
			Status:    "streaming",
		}
		return &ai.AIRun{
			ID:                 uuid.NewString(),
			SessionID:          sessionID,
			ClientRequestID:    strings.TrimSpace(input.ClientRequestID),
			UserMessageID:      userMessageID,
			AssistantMessageID: assistantMessageID,
			Status:             "running",
			TraceJSON:          "{}",
		}, createdUser, createdAssistant
	})
	if err != nil {
		return shell, fmt.Errorf("create run shell: %w", err)
	}

	shell = ChatShell{
		SessionID: sessionID,
		Scene:     scene,
		Run:       run,
		Reused:    !created,
	}
	if created {
		shell.UserMessage = createdUser
		shell.AssistantMessage = createdAssistant
		return shell, nil
	}

	userMessage, err := l.ChatDAO.GetMessage(ctx, run.UserMessageID)
	if err != nil {
		return shell, fmt.Errorf("load user message shell: %w", err)
	}
	assistantMessage, err := l.ChatDAO.GetMessage(ctx, run.AssistantMessageID)
	if err != nil {
		return shell, fmt.Errorf("load assistant message shell: %w", err)
	}
	if userMessage == nil || assistantMessage == nil {
		return shell, fmt.Errorf("load reused shell messages")
	}
	shell.UserMessage = userMessage
	shell.AssistantMessage = assistantMessage
	return shell, nil
}

func (l *Logic) emitTerminalFailure(ctx context.Context, shell ChatShell, seqCounter *int, internalErr error, summaryBody string, assistantBody string, emit EventEmitter) error {
	publicError := sanitizeUserFacingError(internalErr)
	projected := airuntime.NewErrorEvent(shell.Run.ID, errors.New(publicError))
	eventID, err := l.appendRunEventWithID(ctx, shell.Run.ID, shell.SessionID, seqCounter, projected.Event, projected.Data)
	if err != nil {
		return err
	}
	emit(projected.Event, withEventID(projected.Data, eventID))

	runUpdate := aidao.AIRunStatusUpdate{
		AssistantMessageID: shell.AssistantMessage.ID,
		Status:             "failed_runtime",
		ErrorMessage:       internalErr.Error(),
	}
	assistantSnapshot := buildAssistantFailureSnapshot(summaryBody, assistantBody, publicError)
	if err := l.finalizeRunCritical(ctx, shell, runUpdate, assistantSnapshot); err != nil {
		return err
	}
	if err := l.persistRunEnhancementsBestEffort(ctx, shell.Run.ID, shell.SessionID, runUpdate.Status, assistantSnapshot); err != nil {
		// 启动阶段尚未输出任何内容时，允许增强持久化降级为 best-effort。
		if strings.TrimSpace(summaryBody) == "" && strings.TrimSpace(assistantBody) == "" {
			return nil
		}
		return fmt.Errorf("persist run artifacts: %w", err)
	}
	return nil
}

func decodeRunEventPayload(raw string) (any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}

	var payload any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return map[string]any{}, nil
	}
	return payload, nil
}

func (l *Logic) emitExistingShellTerminal(ctx context.Context, shell ChatShell, emit EventEmitter) {
	switch shell.Run.Status {
	case "failed", "failed_runtime":
		emit("error", map[string]any{
			"run_id":  shell.Run.ID,
			"message": sanitizeUserFacingError(errors.New(shell.Run.ErrorMessage)),
		})
	case "waiting_approval":
		if l.RunEventDAO != nil {
			if events, err := l.RunEventDAO.ListByRun(ctx, shell.Run.ID); err == nil {
				type approvalSnapshot struct {
					seq     int
					payload map[string]any
				}
				sort.SliceStable(events, func(i, j int) bool {
					if events[i].Seq == events[j].Seq {
						return events[i].ID < events[j].ID
					}
					return events[i].Seq < events[j].Seq
				})
				resolvedCalls := make(map[string]struct{}, len(events))
				latestUnresolvedApprovals := make(map[string]approvalSnapshot, len(events))
				for _, event := range events {
					switch strings.TrimSpace(event.EventType) {
					case "tool_result":
						var payload map[string]any
						if unmarshalErr := json.Unmarshal([]byte(event.PayloadJSON), &payload); unmarshalErr != nil {
							continue
						}
						callID := strings.TrimSpace(stringValue(payload, "call_id"))
						if callID == "" {
							callID = strings.TrimSpace(event.ToolCallID)
						}
						if callID == "" {
							continue
						}
						resolvedCalls[callID] = struct{}{}
						delete(latestUnresolvedApprovals, callID)
					case "tool_approval":
						var payload map[string]any
						if unmarshalErr := json.Unmarshal([]byte(event.PayloadJSON), &payload); unmarshalErr != nil {
							continue
						}
						callID := strings.TrimSpace(stringValue(payload, "call_id"))
						if callID == "" {
							callID = strings.TrimSpace(event.ToolCallID)
						}
						if callID == "" {
							continue
						}
						delete(resolvedCalls, callID)
						latestUnresolvedApprovals[callID] = approvalSnapshot{
							seq:     event.Seq,
							payload: payload,
						}
					}
				}
				pendingApprovals := make([]approvalSnapshot, 0, len(latestUnresolvedApprovals))
				for _, snapshot := range latestUnresolvedApprovals {
					pendingApprovals = append(pendingApprovals, snapshot)
				}
				sort.SliceStable(pendingApprovals, func(i, j int) bool {
					return pendingApprovals[i].seq < pendingApprovals[j].seq
				})
				for _, snapshot := range pendingApprovals {
					emit("tool_approval", snapshot.payload)
				}
			}
		}
		emit("run_state", map[string]any{
			"run_id":  shell.Run.ID,
			"status":  "waiting_approval",
			"agent":   "executor",
			"summary": shell.AssistantMessage.Content,
		})
	case "cancelled", "expired":
		emit("run_state", map[string]any{
			"run_id":  shell.Run.ID,
			"status":  shell.Run.Status,
			"agent":   "executor",
			"summary": shell.AssistantMessage.Content,
		})
	case "completed", "completed_with_tool_errors":
		emit("done", map[string]any{
			"run_id":  shell.Run.ID,
			"status":  shell.Run.Status,
			"summary": shell.AssistantMessage.Content,
		})
	}
}

func (l *Logic) finalizeRunCritical(ctx context.Context, shell ChatShell, runUpdate aidao.AIRunStatusUpdate, assistantContent string) error {
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil
	}

	return l.svcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chatDAO := aidaochat.NewAIChatDAO(tx)
		runDAO := aidao.NewAIRunDAO(tx)
		if err := chatDAO.UpdateMessage(ctx, shell.AssistantMessage.ID, map[string]any{
			"content": assistantContent,
			"status":  assistantStatusFromRunStatus(runUpdate.Status),
		}); err != nil {
			return err
		}

		runUpdate.AssistantMessageID = shell.AssistantMessage.ID
		runUpdate.ProgressSummary = truncateString(assistantContent, 500)
		return runDAO.UpdateRunStatus(ctx, shell.Run.ID, runUpdate)
	})
}

func (l *Logic) persistRunEnhancementsBestEffort(ctx context.Context, runID, sessionID, status, _ string) error {
	if l.RunEventDAO == nil || l.RunProjectionDAO == nil || l.RunContentDAO == nil {
		return nil
	}

	events, err := l.RunEventDAO.ListByRun(ctx, runID)
	if err != nil {
		return err
	}
	projection, contents, err := airuntime.BuildProjection(events)
	if err != nil {
		return err
	}
	projection.Status = status
	projectionJSON, err := json.Marshal(projection)
	if err != nil {
		return err
	}

	for _, content := range contents {
		if err := l.RunContentDAO.Create(ctx, content); err != nil {
			return err
		}
	}

	return l.RunProjectionDAO.Upsert(ctx, &ai.AIRunProjection{
		ID:             uuid.NewString(),
		RunID:          runID,
		SessionID:      sessionID,
		Version:        projection.Version,
		Status:         projection.Status,
		ProjectionJSON: string(projectionJSON),
	})
}

func buildAssistantFailureSnapshot(summaryBody, assistantBody, publicError string) string {
	if strings.TrimSpace(assistantBody) != "" {
		return assistantBody
	}
	if strings.TrimSpace(summaryBody) != "" {
		return ""
	}
	return publicError
}

func ensureDoneSummary(payload map[string]any, summary string, hasToolErrors bool) {
	if payload == nil {
		return
	}
	resolved := strings.TrimSpace(stringValue(payload, "summary"))
	if resolved == "" {
		resolved = strings.TrimSpace(summary)
	}
	if resolved == "" && hasToolErrors {
		resolved = "工具调用失败，未生成可用结论。请调整参数后重试。"
	}
	if resolved != "" {
		payload["summary"] = resolved
	}
}

func sanitizeUserFacingError(err error) string {
	if err == nil {
		return "生成中断，请稍后重试。"
	}
	return "生成中断，请稍后重试。"
}

func shouldRetainPartialStreamSnapshot(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "timeout")
}

func accumulateProjectedEvents(events []airuntime.PublicStreamEvent, assistantContent *strings.Builder) projectedRunUpdate {
	update := projectedRunUpdate{}
	for _, projected := range events {
		if projected.Event == "delta" {
			if data, ok := projected.Data.(map[string]any); ok {
				agent := strings.TrimSpace(stringValue(data, "agent"))
				if content, ok := data["content"].(string); ok && agent != "executor" && assistantContent != nil {
					assistantContent.WriteString(content)
				}
			}
		}
		if projected.Event == "agent_handoff" {
			if data, ok := projected.Data.(map[string]any); ok {
				if assistantType, ok := data["to"].(string); ok {
					update.AssistantType = assistantType
				}
				if intentType, ok := data["intent"].(string); ok {
					update.IntentType = intentType
				}
			}
		}
	}
	return update
}

func (l *Logic) consumeProjectedEvents(ctx context.Context, runID, sessionID string, seqCounter *int, events []airuntime.PublicStreamEvent, emit EventEmitter, assistantContent *strings.Builder) (projectedRunUpdate, error) {
	update := accumulateProjectedEvents(events, assistantContent)
	for _, projected := range events {
		eventID, err := l.appendRunEventWithID(ctx, runID, sessionID, seqCounter, projected.Event, projected.Data)
		if err != nil {
			return update, err
		}
		emit(projected.Event, withEventID(projected.Data, eventID))
	}
	return update, nil
}

func (l *Logic) appendRunEvent(ctx context.Context, runID, sessionID string, seqCounter *int, eventName string, payload any) error {
	_, err := l.appendRunEventWithID(ctx, runID, sessionID, seqCounter, eventName, payload)
	return err
}

func (l *Logic) appendRunEventWithID(ctx context.Context, runID, sessionID string, seqCounter *int, eventName string, payload any) (string, error) {
	if l.RunEventDAO == nil || seqCounter == nil {
		return "", nil
	}

	eventType, raw, err := marshalProjectedEvent(eventName, payload)
	if err != nil {
		return "", err
	}
	if eventType == "" {
		return "", nil
	}
	eventID := uuid.NewString()
	*seqCounter = *seqCounter + 1
	return eventID, l.RunEventDAO.Create(ctx, &ai.AIRunEvent{
		ID:          eventID,
		RunID:       runID,
		SessionID:   sessionID,
		Seq:         *seqCounter,
		EventType:   string(eventType),
		AgentName:   eventAgentName(payload),
		ToolCallID:  eventToolCallID(payload),
		PayloadJSON: raw,
	})
}

func withEventID(payload any, eventID string) any {
	if strings.TrimSpace(eventID) == "" {
		return payload
	}

	data, ok := payload.(map[string]any)
	if !ok {
		return payload
	}

	copyPayload := make(map[string]any, len(data)+1)
	for key, value := range data {
		copyPayload[key] = value
	}
	copyPayload["event_id"] = eventID
	return copyPayload
}

func assistantStatusFromRunStatus(status string) string {
	switch status {
	case "failed_runtime":
		return "error"
	case "waiting_approval":
		return "streaming"
	default:
		return "done"
	}
}

func marshalProjectedEvent(eventName string, payload any) (airuntime.EventType, string, error) {
	data, _ := payload.(map[string]any)
	switch eventName {
	case "meta":
		return marshalTypedEvent(airuntime.EventTypeMeta, &airuntime.MetaPayload{
			RunID:     stringValue(data, "run_id"),
			SessionID: stringValue(data, "session_id"),
			Turn:      intValue(data, "turn"),
		})
	case "agent_handoff":
		return marshalTypedEvent(airuntime.EventTypeAgentHandoff, &airuntime.AgentHandoffPayload{
			From:   stringValue(data, "from"),
			To:     stringValue(data, "to"),
			Intent: stringValue(data, "intent"),
		})
	case "plan":
		return marshalTypedEvent(airuntime.EventTypePlan, &airuntime.PlanPayload{
			Iteration: intValue(data, "iteration"),
			Steps:     stringSliceValue(data, "steps"),
		})
	case "replan":
		return marshalTypedEvent(airuntime.EventTypeReplan, &airuntime.ReplanPayload{
			Iteration: intValue(data, "iteration"),
			Completed: intValue(data, "completed"),
			IsFinal:   boolValue(data, "is_final"),
			Steps:     stringSliceValue(data, "steps"),
		})
	case "delta":
		return marshalTypedEvent(airuntime.EventTypeDelta, &airuntime.DeltaPayload{
			Agent:   stringValue(data, "agent"),
			Content: stringValue(data, "content"),
		})
	case "tool_call":
		if strings.TrimSpace(stringValue(data, "call_id")) == "" || strings.TrimSpace(stringValue(data, "tool_name")) == "" {
			return "", "", nil
		}
		return marshalTypedEvent(airuntime.EventTypeToolCall, &airuntime.ToolCallPayload{
			Agent:     stringValue(data, "agent"),
			CallID:    stringValue(data, "call_id"),
			ToolName:  stringValue(data, "tool_name"),
			Arguments: mapValue(data, "arguments"),
		})
	case "tool_approval":
		if strings.TrimSpace(stringValue(data, "approval_id")) == "" || strings.TrimSpace(stringValue(data, "call_id")) == "" || strings.TrimSpace(stringValue(data, "tool_name")) == "" {
			return "", "", nil
		}
		return marshalTypedEvent(airuntime.EventTypeToolApproval, &airuntime.ToolApprovalPayload{
			ApprovalID:     stringValue(data, "approval_id"),
			TargetID:       stringValue(data, "target_id"),
			CallID:         stringValue(data, "call_id"),
			ToolName:       stringValue(data, "tool_name"),
			Preview:        mapValue(data, "preview"),
			TimeoutSeconds: intValue(data, "timeout_seconds"),
		})
	case "tool_result":
		return marshalTypedEvent(airuntime.EventTypeToolResult, &airuntime.ToolResultPayload{
			Agent:    stringValue(data, "agent"),
			CallID:   stringValue(data, "call_id"),
			ToolName: stringValue(data, "tool_name"),
			Content:  stringValue(data, "content"),
			Status:   stringValue(data, "status"),
		})
	case "run_state":
		if strings.TrimSpace(stringValue(data, "status")) == "" {
			return "", "", nil
		}
		return marshalTypedEvent(airuntime.EventTypeRunState, &airuntime.RunStatePayload{
			Status: stringValue(data, "status"),
			Agent:  stringValue(data, "agent"),
		})
	case "done":
		return marshalTypedEvent(airuntime.EventTypeDone, &airuntime.DonePayload{
			RunID:      stringValue(data, "run_id"),
			Status:     stringValue(data, "status"),
			Summary:    stringValue(data, "summary"),
			Iterations: intValue(data, "iterations"),
		})
	case "error":
		return marshalTypedEvent(airuntime.EventTypeError, &airuntime.ErrorPayload{
			RunID:   stringValue(data, "run_id"),
			Message: stringValue(data, "message"),
			Code:    stringValue(data, "code"),
		})
	default:
		return "", "", nil
	}
}

func marshalTypedEvent(eventType airuntime.EventType, payload any) (airuntime.EventType, string, error) {
	raw, err := airuntime.MarshalEventPayload(eventType, payload)
	return eventType, raw, err
}

func eventAgentName(payload any) string {
	data, _ := payload.(map[string]any)
	return stringValue(data, "agent")
}

func eventToolCallID(payload any) string {
	data, _ := payload.(map[string]any)
	return stringValue(data, "call_id")
}

func stringValue(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, _ := data[key].(string)
	return value
}

func intValue(data map[string]any, key string) int {
	if data == nil {
		return 0
	}
	switch value := data[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}

func boolValue(data map[string]any, key string) bool {
	if data == nil {
		return false
	}
	value, _ := data[key].(bool)
	return value
}

func stringSliceValue(data map[string]any, key string) []string {
	if data == nil {
		return nil
	}
	raw, ok := data[key].([]any)
	if !ok {
		if direct, ok := data[key].([]string); ok {
			return direct
		}
		return nil
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func mapValue(data map[string]any, key string) map[string]any {
	if data == nil {
		return nil
	}
	value, _ := data[key].(map[string]any)
	return value
}

func (l *Logic) runtimeContext(ctx context.Context) context.Context {
	if l == nil || l.svcCtx == nil {
		return ctx
	}
	return runtimectx.WithServices(ctx, l.svcCtx)
}
