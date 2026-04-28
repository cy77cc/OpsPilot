package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/stream"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	runtimecontext "github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/context"
	projectionruntime "github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/projection"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EmitTerminalFailure 发送终端失败事件并持久化。
func EmitTerminalFailure(ctx context.Context, l *Logic, shell ChatShell, seq *int, internalErr error, summaryBody, assistantBody string, emit EventEmitter) error {
	publicErr := stream.SanitizeUserFacingError(internalErr)
	projected := airuntime.NewErrorEvent(shell.Run.ID, errors.New(publicErr))
	eid, err := AppendRunEventWithID(ctx, l, shell.Run.ID, shell.SessionID, seq, projected.Event, projected.Data)
	if err != nil {
		return err
	}
	if err := persistTerminalProjectionEvent(ctx, l, shell.Run.ID, shell.SessionID, eid, projected); err != nil {
		return err
	}
	emit(projected.Event, withEventID(projected.Data, eid))
	runUpdate := aidao.AIRunStatusUpdate{AssistantMessageID: shell.AssistantMessage.ID, Status: "failed_runtime", ErrorMessage: stream.SanitizeUserFacingError(internalErr)}
	snapshot := stream.BuildAssistantFailureSnapshot(summaryBody, assistantBody, publicErr)
	if err := FinalizeRunCritical(ctx, l, shell, runUpdate, snapshot); err != nil {
		return err
	}
	if err := PersistRunEnhancementsBestEffort(ctx, l, shell.Run.ID, shell.SessionID, runUpdate.Status, snapshot); err != nil {
		if strings.TrimSpace(summaryBody) == "" && strings.TrimSpace(assistantBody) == "" {
			return nil
		}
		return fmt.Errorf("persist run artifacts: %w", err)
	}
	return nil
}

// FinalizeRunCritical 事务性更新消息和运行状态。
func FinalizeRunCritical(ctx context.Context, l *Logic, shell ChatShell, runUpdate aidao.AIRunStatusUpdate, assistantContent string) error {
	if l.SvcCtx == nil || l.SvcCtx.DB == nil {
		return nil
	}
	return l.SvcCtx.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chatDAO := aidaochat.NewAIChatDAO(tx)
		runDAO := aidao.NewAIRunDAO(tx)
		if err := chatDAO.UpdateMessage(ctx, shell.AssistantMessage.ID, map[string]any{"content": assistantContent, "status": assistantStatusFromRunStatus(runUpdate.Status)}); err != nil {
			return err
		}
		runUpdate.AssistantMessageID = shell.AssistantMessage.ID
		runUpdate.ProgressSummary = TruncateString(assistantContent, maxProgressSummaryLen)
		return runDAO.UpdateRunStatus(ctx, shell.Run.ID, runUpdate)
	})
}

// PersistRunEnhancementsBestEffort 持久化投影和内容。
func PersistRunEnhancementsBestEffort(ctx context.Context, l *Logic, runID, sessionID, status string, _ string) error {
	if l.RunProjectionDAO == nil {
		return nil
	}
	current, err := l.RunProjectionDAO.GetByRunID(ctx, runID)
	if err != nil {
		return err
	}
	if current == nil || strings.TrimSpace(current.ProjectionJSON) == "" {
		return nil
	}
	var projection airuntime.RunProjection
	if err := json.Unmarshal([]byte(current.ProjectionJSON), &projection); err != nil {
		return err
	}
	if strings.TrimSpace(status) != "" {
		projection.Status = status
	}
	projectionJSON, err := json.Marshal(&projection)
	if err != nil {
		return err
	}
	return l.RunProjectionDAO.Upsert(ctx, &ai.AIRunProjection{
		ID:             current.ID,
		RunID:          runID,
		SessionID:      sessionID,
		Version:        current.Version,
		Status:         projection.Status,
		ProjectionJSON: string(projectionJSON),
	})
}

// EmitExistingShellTerminal 对重用 shell 发送终态事件。
func EmitExistingShellTerminal(ctx context.Context, l *Logic, shell ChatShell, emit EventEmitter) {
	switch shell.Run.Status {
	case "failed", "failed_runtime":
		emit("error", map[string]any{"run_id": shell.Run.ID, "message": stream.SanitizeUserFacingError(errors.New(shell.Run.ErrorMessage))})
	case "cancelled", "expired":
		emit("run_state", map[string]any{"run_id": shell.Run.ID, "status": shell.Run.Status, "agent": "executor", "summary": shell.AssistantMessage.Content})
	case "completed", "completed_with_tool_errors":
		emit("done", map[string]any{"run_id": shell.Run.ID, "status": shell.Run.Status, "summary": shell.AssistantMessage.Content})
	default:
		if ai.IsOpenRunStatus(shell.Run.Status) {
			emit("run_state", map[string]any{"run_id": shell.Run.ID, "status": shell.Run.Status, "agent": "executor", "summary": shell.AssistantMessage.Content})
		}
	}
}

// AppendRunEventWithID 追加运行事件并返回事件 ID。
func AppendRunEventWithID(ctx context.Context, l *Logic, runID, sessionID string, seq *int, eventName string, payload any) (string, error) {
	if l.RunEventDAO == nil || seq == nil {
		return "", nil
	}
	eventType, raw, err := marshalRuntimeEvent(eventName, payload)
	if err != nil {
		return "", err
	}
	if eventType == "" {
		return "", nil
	}
	eventID := uuid.NewString()
	*seq++
	agentName := stream.EventAgentName(payload)
	if eventType == airuntime.EventTypeDelegationNode {
		data, _ := payload.(map[string]any)
		agentName = strings.TrimSpace(stream.StringValue(data, "agent_name"))
	}
	return eventID, l.RunEventDAO.Create(ctx, &ai.AIRunEvent{
		ID: eventID, RunID: runID, SessionID: sessionID, Seq: *seq,
		EventType: string(eventType), AgentName: agentName,
		ToolCallID: stream.EventToolCallID(payload), PayloadJSON: raw,
	})
}

func marshalRuntimeEvent(eventName string, payload any) (airuntime.EventType, string, error) {
	eventType, raw, err := stream.MarshalProjectedEvent(eventName, payload)
	if err != nil || eventType != "" || eventName != "delegation_node" {
		return eventType, raw, err
	}
	data, _ := payload.(map[string]any)
	node := &airuntime.DelegationNodePayload{
		DelegationID: strings.TrimSpace(stream.StringValue(data, "delegation_id")),
		AgentName:    strings.TrimSpace(stream.StringValue(data, "agent_name")),
		Intent:       strings.TrimSpace(stream.StringValue(data, "intent")),
		Status:       strings.TrimSpace(stream.StringValue(data, "status")),
		Title:        strings.TrimSpace(stream.StringValue(data, "title")),
		Summary:      strings.TrimSpace(stream.StringValue(data, "summary")),
		RiskLevel:    strings.TrimSpace(stream.StringValue(data, "risk_level")),
	}
	raw, err = airuntime.MarshalEventPayload(airuntime.EventTypeDelegationNode, node)
	return airuntime.EventTypeDelegationNode, raw, err
}

// ConsumeProjectedEvents 消费投影事件并持久化。
func ConsumeProjectedEvents(ctx context.Context, l *Logic, runID, sessionID string, seq *int, events []airuntime.PublicStreamEvent, emit EventEmitter) (stream.RunUpdate, error) {
	update := stream.AccumulateProjectedEvents(events, nil)
	state, current, err := loadIncrementalProjectionState(ctx, l, runID)
	if err != nil {
		return update, err
	}
	for _, projected := range events {
		if err := persistApprovalResumeTarget(ctx, l, projected); err != nil {
			return update, err
		}
		eid, err := AppendRunEventWithID(ctx, l, runID, sessionID, seq, projected.Event, projected.Data)
		if err != nil {
			return update, err
		}
		state = projectionruntime.ApplyEvent(state, projectionruntime.Event{
			ID:   eid,
			Type: projected.Event,
			Text: projectionEventText(projected),
			Data: projected.Data,
		})
		emit(projected.Event, withEventID(projected.Data, eid))
	}
	if err := persistIncrementalProjection(ctx, l, sessionID, state, current); err != nil {
		return update, err
	}
	return update, nil
}

func persistApprovalResumeTarget(ctx context.Context, l *Logic, event airuntime.PublicStreamEvent) error {
	if l == nil || l.SvcCtx == nil || l.SvcCtx.DB == nil || event.Event != "tool_approval" {
		return nil
	}
	data, _ := event.Data.(map[string]any)
	if data == nil {
		return nil
	}
	approvalID := strings.TrimSpace(stream.StringValue(data, "approval_id"))
	targetID := strings.TrimSpace(stream.StringValue(data, "target_id"))
	if approvalID == "" || targetID == "" {
		return nil
	}
	return aidaoapproval.NewAIApprovalTaskDAO(l.SvcCtx.DB).UpdateResumeTarget(ctx, approvalID, targetID)
}

func assistantStatusFromRunStatus(status string) string {
	switch status {
	case "failed_runtime":
		return "error"
	case "waiting_approval", "running", "delegating", "waiting_subagent", "resuming", "resume_failed_retryable":
		return "streaming"
	default:
		return "done"
	}
}

func withEventID(payload any, eventID string) any {
	if strings.TrimSpace(eventID) == "" {
		return payload
	}
	data, ok := payload.(map[string]any)
	if !ok {
		return payload
	}
	cp := make(map[string]any, len(data)+1)
	for k, v := range data {
		cp[k] = v
	}
	cp["event_id"] = eventID
	return cp
}

func projectionEventText(event airuntime.PublicStreamEvent) string {
	if event.Event != "delta" {
		return ""
	}
	data, _ := event.Data.(map[string]any)
	if data == nil {
		return ""
	}
	content, _ := data["content"].(string)
	return content
}

func persistTerminalProjectionEvent(ctx context.Context, l *Logic, runID, sessionID, eventID string, event airuntime.PublicStreamEvent) error {
	state, current, err := loadIncrementalProjectionState(ctx, l, runID)
	if err != nil {
		return err
	}
	state = projectionruntime.ApplyEvent(state, projectionruntime.Event{
		ID:   eventID,
		Type: event.Event,
		Text: projectionEventText(event),
		Data: event.Data,
	})
	return persistIncrementalProjection(ctx, l, sessionID, state, current)
}

func buildSessionAgentInput(ctx context.Context, l *Logic, shell ChatShell, input ChatInput) []*schema.Message {
	history := loadSessionHistoryMessages(ctx, l, shell, input.Budget)
	current := schema.UserMessage(BuildAugmentedMessage(ctx, l, shell.Scene, input.Context, input.Message))
	return append(history, current)
}

func loadSessionHistoryMessages(ctx context.Context, l *Logic, shell ChatShell, budget runtimecontext.Budget) []*schema.Message {
	if l == nil || l.ChatDAO == nil || strings.TrimSpace(shell.SessionID) == "" {
		return nil
	}
	rows, err := l.ChatDAO.ListMessagesBySession(ctx, shell.SessionID)
	if err != nil || len(rows) == 0 {
		return nil
	}

	history := make([]runtimecontext.Message, 0, len(rows))
	for _, row := range rows {
		if row.ID == shell.UserMessage.ID || row.ID == shell.AssistantMessage.ID {
			continue
		}
		content := strings.TrimSpace(row.Content)
		if content == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(row.Role)) {
		case "user":
			history = append(history, runtimecontext.Message{Role: "user", Content: content})
		case "assistant":
			history = append(history, runtimecontext.Message{Role: "assistant", Content: content})
		case "system":
			history = append(history, runtimecontext.Message{Role: "system", Content: content, Pinned: true})
		}
	}

	selected := runtimecontext.SelectBudgeted(history, budget)
	if len(selected) < len(history) {
		selected = runtimecontext.CompressOverflow(history, budget)
	}

	result := make([]*schema.Message, 0, len(selected))
	for _, msg := range selected {
		switch strings.ToLower(strings.TrimSpace(msg.Role)) {
		case "assistant":
			result = append(result, schema.AssistantMessage(msg.Content, nil))
		case "system":
			result = append(result, schema.SystemMessage(msg.Content))
		default:
			result = append(result, schema.UserMessage(msg.Content))
		}
	}
	return result
}
