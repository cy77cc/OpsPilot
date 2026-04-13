package logic

import (
	"context"
	"fmt"
	"strings"
	"time"

	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/event"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"gorm.io/gorm"
)

func (m *ApprovalWriteModel) EmitRunResuming(ctx context.Context, approvalID string) error {
	return m.emitRunLifecycleEvent(ctx, approvalID, event.RunEventTypeResuming, "resuming", map[string]any{
		"status": "resuming",
	})
}

func (m *ApprovalWriteModel) EmitRunResumed(ctx context.Context, approvalID string) error {
	return m.emitRunLifecycleEvent(ctx, approvalID, event.RunEventTypeResumed, "running", map[string]any{
		"status": "running",
	})
}

func (m *ApprovalWriteModel) EmitRunCompleted(ctx context.Context, approvalID, runStatus string) error {
	if strings.TrimSpace(runStatus) == "" {
		runStatus = "completed"
	}
	return m.emitRunLifecycleEvent(ctx, approvalID, event.RunEventTypeCompleted, runStatus, map[string]any{
		"status": runStatus,
	})
}

func (m *ApprovalWriteModel) EmitRunResumeFailed(ctx context.Context, approvalID string, retryable bool, cause error) error {
	status := "failed"
	if retryable {
		status = "resume_failed_retryable"
	}
	payload := map[string]any{
		"approval_id": approvalID,
		"retryable":   retryable,
	}
	if cause != nil {
		payload["message"] = cause.Error()
	}
	return m.emitRunLifecycleEvent(ctx, approvalID, event.RunEventTypeResumeFailed, status, payload)
}

func (m *ApprovalWriteModel) RenewApprovalLease(ctx context.Context, approvalID string, leaseExpiresAt time.Time) (bool, error) {
	if m == nil || m.db == nil {
		return false, fmt.Errorf("approval write model not initialized")
	}
	dao := aidaoapproval.NewAIApprovalTaskDAO(m.db)
	return dao.AcquireOrStealLease(ctx, approvalID, leaseExpiresAt)
}

func (m *ApprovalWriteModel) AcquireApprovalLease(ctx context.Context, approvalID string, leaseExpiresAt time.Time) (bool, error) {
	if m == nil || m.db == nil {
		return false, fmt.Errorf("approval write model not initialized")
	}
	dao := aidaoapproval.NewAIApprovalTaskDAO(m.db)
	return dao.AcquireOrStealLease(ctx, approvalID, leaseExpiresAt)
}

func (m *ApprovalWriteModel) emitRunLifecycleEvent(ctx context.Context, approvalID, eventType, runStatus string, payload map[string]any) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("approval write model not initialized")
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		approvalDAO := aidaoapproval.NewAIApprovalTaskDAO(tx)
		runDAO := aidao.NewAIRunDAO(tx)
		outboxDAO := aidaoapproval.NewAIApprovalOutboxDAO(tx)

		task, err := approvalDAO.GetByApprovalID(ctx, approvalID)
		if err != nil {
			return fmt.Errorf("load approval task: %w", err)
		}
		if task == nil {
			return fmt.Errorf("approval task not found")
		}
		run, err := runDAO.GetRun(ctx, task.RunID)
		if err != nil {
			return fmt.Errorf("load run: %w", err)
		}
		if run == nil {
			return fmt.Errorf("run not found")
		}

		if shouldSyncRunStatusFromLifecycleEvent(eventType, runStatus) {
			if err := runDAO.UpdateRunStatus(ctx, run.ID, aidao.AIRunStatusUpdate{Status: runStatus}); err != nil {
				return fmt.Errorf("update run status: %w", err)
			}
		}

		switch eventType {
		case event.RunEventTypeResuming:
			payload["run_id"] = run.ID
			payload["session_id"] = task.SessionID
		case event.RunEventTypeResumed:
			payload["run_id"] = run.ID
			payload["session_id"] = task.SessionID
		case event.RunEventTypeCompleted:
			payload["run_id"] = run.ID
			payload["session_id"] = task.SessionID
		case event.RunEventTypeResumeFailed:
			payload["run_id"] = run.ID
			payload["session_id"] = task.SessionID
			payload["approval_id"] = task.ApprovalID
		}

		if err := m.writeApprovalEvent(ctx, tx, outboxDAO, task, eventType, payload); err != nil {
			return err
		}
		return nil
	})
}

func shouldSyncRunStatusFromLifecycleEvent(eventType, runStatus string) bool {
	if strings.TrimSpace(runStatus) == "" {
		return false
	}
	switch eventType {
	case event.RunEventTypeResumeFailed:
		return runStatus == "resume_failed_retryable"
	default:
		return true
	}
}

func (m *ApprovalWriteModel) writeApprovalEvent(ctx context.Context, tx *gorm.DB, outboxDAO *aidaoapproval.AIApprovalOutboxDAO, task *ai.AIApprovalTask, eventType string, payload any) error {
	if outboxDAO == nil {
		return fmt.Errorf("outbox dao is required")
	}
	if task == nil {
		return fmt.Errorf("approval task is required")
	}
	sequence, err := outboxDAO.NextSequence(ctx, task.RunID)
	if err != nil {
		return fmt.Errorf("allocate approval outbox sequence: %w", err)
	}

	now := time.Now().UTC()
	envelope, err := buildApprovalEventEnvelope(eventType, sequence, now, task, payload)
	if err != nil {
		return err
	}
	eventRow := &ai.AIApprovalOutboxEvent{
		EventID:     envelope.EventID,
		Sequence:    envelope.Sequence,
		AggregateID: envelope.AggregateID,
		OccurredAt:  envelope.OccurredAt,
		Version:     envelope.Version,
		ApprovalID:  task.ApprovalID,
		ToolCallID:  task.ToolCallID,
		EventType:   envelope.EventType,
		RunID:       task.RunID,
		SessionID:   task.SessionID,
		PayloadJSON: envelope.PayloadJSON,
		Status:      "pending",
	}
	return outboxDAO.EnqueueOrTouch(ctx, eventRow)
}

func buildApprovalEventEnvelope(eventType string, sequence int64, occurredAt time.Time, task *ai.AIApprovalTask, payload any) (*event.ApprovalEventEnvelope, error) {
	switch eventType {
	case event.ApprovalEventTypeRequested:
		return event.NewApprovalRequestedEnvelope(event.ApprovalRequestedInput{
			EventID:     "",
			OccurredAt:  occurredAt,
			Sequence:    sequence,
			Version:     1,
			RunID:       task.RunID,
			SessionID:   task.SessionID,
			ApprovalID:  task.ApprovalID,
			ToolCallID:  task.ToolCallID,
			AggregateID: task.RunID,
			Payload:     payload,
		})
	case event.ApprovalEventTypeDecided:
		return event.NewApprovalDecidedEnvelope(event.ApprovalDecidedInput{
			EventID:     "",
			OccurredAt:  occurredAt,
			Sequence:    sequence,
			Version:     1,
			RunID:       task.RunID,
			SessionID:   task.SessionID,
			ApprovalID:  task.ApprovalID,
			ToolCallID:  task.ToolCallID,
			AggregateID: task.RunID,
			Payload:     payload,
		})
	case event.ApprovalEventTypeExpired:
		return event.NewApprovalExpiredEnvelope(event.ApprovalExpiredInput{
			EventID:     "",
			OccurredAt:  occurredAt,
			Sequence:    sequence,
			Version:     1,
			RunID:       task.RunID,
			SessionID:   task.SessionID,
			ApprovalID:  task.ApprovalID,
			ToolCallID:  task.ToolCallID,
			AggregateID: task.RunID,
			Payload:     payload,
		})
	case event.RunEventTypeResuming:
		return event.NewRunResumingEnvelope(event.RunResumingInput{
			EventID:     "",
			OccurredAt:  occurredAt,
			Sequence:    sequence,
			Version:     1,
			RunID:       task.RunID,
			SessionID:   task.SessionID,
			ApprovalID:  task.ApprovalID,
			ToolCallID:  task.ToolCallID,
			AggregateID: task.RunID,
			Payload:     payload,
		})
	case event.RunEventTypeResumed:
		return event.NewRunResumedEnvelope(event.RunResumedInput{
			EventID:     "",
			OccurredAt:  occurredAt,
			Sequence:    sequence,
			Version:     1,
			RunID:       task.RunID,
			SessionID:   task.SessionID,
			ApprovalID:  task.ApprovalID,
			ToolCallID:  task.ToolCallID,
			AggregateID: task.RunID,
			Payload:     payload,
		})
	case event.RunEventTypeResumeFailed:
		return event.NewRunResumeFailedEnvelope(event.RunResumeFailedInput{
			EventID:     "",
			OccurredAt:  occurredAt,
			Sequence:    sequence,
			Version:     1,
			RunID:       task.RunID,
			SessionID:   task.SessionID,
			ApprovalID:  task.ApprovalID,
			ToolCallID:  task.ToolCallID,
			AggregateID: task.RunID,
			Payload:     payload,
		})
	case event.RunEventTypeCompleted:
		return event.NewRunCompletedEnvelope(event.RunCompletedInput{
			EventID:     "",
			OccurredAt:  occurredAt,
			Sequence:    sequence,
			Version:     1,
			RunID:       task.RunID,
			SessionID:   task.SessionID,
			ApprovalID:  task.ApprovalID,
			ToolCallID:  task.ToolCallID,
			AggregateID: task.RunID,
			Payload:     payload,
		})
	default:
		return nil, fmt.Errorf("unsupported approval event type %q", eventType)
	}
}

func taskDecisionPayload(task *ai.AIApprovalTask) map[string]any {
	payload := map[string]any{
		"approval_id":       task.ApprovalID,
		"run_id":            task.RunID,
		"session_id":        task.SessionID,
		"status":            task.Status,
		"approved":          task.Status == "approved",
		"approved_by":       task.ApprovedBy,
		"comment":           task.Comment,
		"disapprove_reason": task.DisapproveReason,
	}
	if task.DecidedAt != nil {
		payload["decided_at"] = task.DecidedAt.UTC().Format(time.RFC3339)
	}
	return payload
}

func taskDecisionPayloadWithIdempotency(task *ai.AIApprovalTask, idempotencyKey, payloadHash string, result *SubmitApprovalOutput) map[string]any {
	return attachApprovalSubmitIdempotency(taskDecisionPayload(task), idempotencyKey, payloadHash, result)
}

func taskStatusExpiredPayload(task *ai.AIApprovalTask) map[string]any {
	payload := map[string]any{
		"approval_id": task.ApprovalID,
		"run_id":      task.RunID,
		"session_id":  task.SessionID,
		"status":      task.Status,
		"expired":     true,
	}
	if task.ExpiresAt != nil {
		payload["expires_at"] = task.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return payload
}

func taskStatusExpiredPayloadWithIdempotency(task *ai.AIApprovalTask, idempotencyKey, payloadHash string, result *SubmitApprovalOutput) map[string]any {
	return attachApprovalSubmitIdempotency(taskStatusExpiredPayload(task), idempotencyKey, payloadHash, result)
}
