package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

func (w *ApprovalWorker) reconcileApprovalTaskToolCallID(ctx context.Context, task *ai.AIApprovalTask, outboxEvent *ai.AIApprovalOutboxEvent) (*ai.AIApprovalTask, error) {
	if task == nil {
		return nil, nil
	}
	// Prefer persisted run events because tool_approval now carries target_id (interrupt context id),
	// while outbox payload can still contain legacy call_id only.
	candidate := w.findApprovalResumeTargetFromRunEvents(ctx, task)
	if strings.TrimSpace(candidate) == "" && outboxEvent != nil {
		candidate = approvalResumeTargetFromPayloadJSON(outboxEvent.PayloadJSON)
	}
	candidate = strings.TrimSpace(candidate)
	if candidate == "" || candidate == strings.TrimSpace(task.ToolCallID) {
		return task, nil
	}

	task.ToolCallID = candidate
	if w == nil || w.logic == nil || w.logic.svcCtx == nil || w.logic.svcCtx.DB == nil {
		return task, nil
	}
	if err := w.logic.svcCtx.DB.WithContext(ctx).
		Model(&ai.AIApprovalTask{}).
		Where("approval_id = ?", task.ApprovalID).
		Update("tool_call_id", candidate).Error; err != nil {
		return nil, fmt.Errorf("backfill approval task tool_call_id: %w", err)
	}
	return task, nil
}

func (w *ApprovalWorker) findApprovalResumeTargetFromRunEvents(ctx context.Context, task *ai.AIApprovalTask) string {
	if task == nil || w == nil || w.logic == nil || w.logic.RunEventDAO == nil || strings.TrimSpace(task.RunID) == "" || strings.TrimSpace(task.ApprovalID) == "" {
		return ""
	}
	events, err := w.logic.RunEventDAO.ListByRun(ctx, task.RunID)
	if err != nil || len(events) == 0 {
		return ""
	}
	for idx := len(events) - 1; idx >= 0; idx-- {
		event := events[idx]
		if strings.TrimSpace(event.EventType) != "tool_approval" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(event.PayloadJSON), &payload); err != nil {
			continue
		}
		if strings.TrimSpace(stringValue(payload, "approval_id")) != strings.TrimSpace(task.ApprovalID) {
			continue
		}
		targetID := strings.TrimSpace(stringValue(payload, "target_id"))
		if targetID != "" {
			return targetID
		}
		callID := strings.TrimSpace(stringValue(payload, "call_id"))
		if callID != "" {
			return callID
		}
	}
	return ""
}

func approvalResumeTargetFromPayloadJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	if targetID := strings.TrimSpace(stringValue(payload, "target_id")); targetID != "" {
		return targetID
	}
	return strings.TrimSpace(stringValue(payload, "call_id"))
}

func (w *ApprovalWorker) retryBackoff(retryCount int) time.Duration {
	if retryCount < 0 {
		retryCount = 0
	}
	delay := w.retryDelay
	for i := 0; i < retryCount; i++ {
		delay *= 2
		if delay >= time.Minute {
			return time.Minute
		}
	}
	if delay > time.Minute {
		return time.Minute
	}
	return delay
}

func (w *ApprovalWorker) defaultResume(ctx context.Context, task *ai.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	if w.logic == nil || w.logic.AIRouter == nil || w.logic.CheckpointStore == nil {
		return nil, fmt.Errorf("AI service not initialized")
	}

	resumeScene := ""
	if w.logic.ChatDAO != nil {
		session, err := w.logic.ChatDAO.GetSession(ctx, task.SessionID, task.UserID, "")
		if err != nil {
			return nil, fmt.Errorf("load session for resume: %w", err)
		}
		if session != nil {
			resumeScene = normalizeScene(session.Scene)
		}
	}

	ctx = w.logic.runtimeContext(ctx)
	ctx = runtimectx.WithAIMetadata(ctx, runtimectx.AIMetadata{
		SessionID:    task.SessionID,
		RunID:        task.RunID,
		CheckpointID: task.CheckpointID,
		UserID:       task.UserID,
		Scene:        resumeScene,
	})
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           w.logic.AIRouter,
		EnableStreaming: true,
		CheckPointStore: w.logic.CheckpointStore,
	})
	return runner.ResumeWithParams(ctx, task.CheckpointID, params)
}

func buildApprovalResumeParams(task *ai.AIApprovalTask) *adk.ResumeParams {
	payload := &approval.ApprovalResult{
		Approved: task != nil && task.Status == "approved",
		Comment:  "",
	}
	if task != nil {
		if reason := strings.TrimSpace(task.DisapproveReason); reason != "" {
			payload.DisapproveReason = &reason
		}
		payload.Comment = task.Comment
		if task.ApprovedBy > 0 {
			payload.ApprovedBy = fmt.Sprintf("%d", task.ApprovedBy)
		}
		if task.DecidedAt != nil {
			decidedAt := task.DecidedAt.UTC()
			payload.ApprovedAt = &decidedAt
		}
	}

	targets := map[string]any{}
	if task != nil {
		targetID := strings.TrimSpace(task.ToolCallID)
		if targetID == "" {
			// fallback_static 场景下 decision.ApprovalID 可能直接等于 call_id
			approvalID := strings.TrimSpace(task.ApprovalID)
			if strings.HasPrefix(approvalID, "call_") {
				targetID = approvalID
			}
		}
		if targetID == "" {
			targetID = strings.TrimSpace(task.CheckpointID)
		}
		if targetID != "" {
			targets[targetID] = payload
		}
	}
	return &adk.ResumeParams{Targets: targets}
}

func (l *Logic) approvalTaskDAO() *aidaoapproval.AIApprovalTaskDAO {
	if l == nil {
		return nil
	}
	if l.ApprovalDAO != nil {
		return l.ApprovalDAO
	}
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil
	}
	return aidaoapproval.NewAIApprovalTaskDAO(l.svcCtx.DB)
}

func (l *Logic) loadApprovalShell(ctx context.Context, task *ai.AIApprovalTask) (ChatShell, int, error) {
	shell := ChatShell{}
	if l == nil || task == nil || l.RunDAO == nil || l.ChatDAO == nil {
		return shell, 0, fmt.Errorf("approval runtime not initialized")
	}

	run, err := l.RunDAO.GetRun(ctx, task.RunID)
	if err != nil {
		return shell, 0, fmt.Errorf("load run: %w", err)
	}
	if run == nil {
		return shell, 0, fmt.Errorf("run not found")
	}

	assistant, err := l.ChatDAO.GetMessage(ctx, run.AssistantMessageID)
	if err != nil {
		return shell, 0, fmt.Errorf("load assistant shell: %w", err)
	}
	if assistant == nil {
		return shell, 0, fmt.Errorf("assistant shell not found")
	}

	seqCounter := 0
	if l.RunEventDAO != nil {
		events, err := l.RunEventDAO.ListByRun(ctx, task.RunID)
		if err != nil {
			return shell, 0, fmt.Errorf("load run events: %w", err)
		}
		if len(events) > 0 {
			seqCounter = events[len(events)-1].Seq
		}
	}

	shell.SessionID = task.SessionID
	shell.Run = run
	shell.AssistantMessage = assistant
	return shell, seqCounter, nil
}
