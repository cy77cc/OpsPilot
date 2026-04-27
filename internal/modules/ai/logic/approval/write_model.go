// Package approval 实现审批写入模型。
package approval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/event"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"gorm.io/gorm"
)

// WriteModel 审批写入模型。
type WriteModel struct {
	db *gorm.DB
}

// NewWriteModel 创建审批写入模型。
func NewWriteModel(db *gorm.DB) *WriteModel {
	if db == nil {
		return &WriteModel{}
	}
	return &WriteModel{db: db}
}

// SubmitApproval 提交审批结果。
func (m *WriteModel) SubmitApproval(ctx context.Context, input SubmitApprovalInput) (*SubmitApprovalOutput, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("approval write model not initialized")
	}
	if strings.TrimSpace(input.ApprovalID) == "" {
		return nil, fmt.Errorf("approval_id is required")
	}
	idempotencyKey := ApprovalSubmitIdempotencyKeyFromContext(ctx)
	payloadHash, err := ApprovalSubmitPayloadHash(input)
	if err != nil {
		return nil, fmt.Errorf("hash approval submit payload: %w", err)
	}
	var result *SubmitApprovalOutput
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		approvalDAO := aidaoapproval.NewAIApprovalTaskDAO(tx)
		outboxDAO := aidaoapproval.NewAIApprovalOutboxDAO(tx)
		task, err := approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
		if err != nil {
			return fmt.Errorf("get approval task: %w", err)
		}
		if task == nil {
			return &ApprovalNotFoundError{ApprovalID: input.ApprovalID}
		}
		if task.UserID != 0 && task.UserID != input.UserID {
			return &ApprovalForbiddenError{ApprovalID: input.ApprovalID, UserID: input.UserID}
		}
		now := time.Now().UTC()
		if task.Status != "pending" {
			replay, err := findSubmitIdempotencyResult(ctx, tx, input.ApprovalID, idempotencyKey, payloadHash)
			if err != nil {
				return err
			}
			if replay != nil {
				result = replay
				return nil
			}
			return AlreadyHandledError(task)
		}
		if task.LockExpiresAt != nil && task.LockExpiresAt.After(now) {
			result = &SubmitApprovalOutput{ApprovalID: input.ApprovalID, Status: task.Status, Message: "approval is currently being processed"}
			return nil
		}
		if task.ExpiresAt != nil && task.ExpiresAt.Before(now) {
			update := tx.WithContext(ctx).Model(&ai.AIApprovalTask{}).
				Where("approval_id = ? AND status = ?", input.ApprovalID, "pending").
				Updates(map[string]any{"status": "expired", "updated_at": now})
			if update.Error != nil {
				return fmt.Errorf("mark approval expired: %w", update.Error)
			}
			if update.RowsAffected == 0 {
				task, _ = approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
				if task != nil && task.Status == "pending" {
					return AlreadyHandledError(task)
				}
				result = &SubmitApprovalOutput{ApprovalID: input.ApprovalID, Status: task.Status, Message: "approval is currently being processed"}
				return nil
			}
			task, _ = approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
			result = &SubmitApprovalOutput{ApprovalID: input.ApprovalID, Status: "expired", Message: "approval has expired"}
			return nil
		}
		var updated bool
		if input.Approved {
			updated, err = approvalDAO.ApproveWithLease(ctx, input.ApprovalID, input.UserID, input.Comment, now.Add(DefaultLeaseWindow))
		} else {
			updated, err = approvalDAO.RejectPending(ctx, input.ApprovalID, input.UserID, input.DisapproveReason, input.Comment)
		}
		if err != nil {
			return err
		}
		if !updated {
			task, _ = approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
			if task != nil && task.Status == "pending" {
				result = &SubmitApprovalOutput{ApprovalID: input.ApprovalID, Status: task.Status, Message: "approval is currently being processed"}
				return nil
			}
			return AlreadyHandledError(task)
		}
		task, _ = approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
		result = &SubmitApprovalOutput{ApprovalID: input.ApprovalID, Status: task.Status, Message: fmt.Sprintf("approval %s successfully", task.Status)}
		payload := TaskDecisionPayloadWithIdempotency(task, idempotencyKey, payloadHash, result)
		return m.writeEvent(ctx, tx, outboxDAO, task, event.ApprovalEventTypeDecided, payload)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// RetryResumeApproval 重试恢复审批。
func (m *WriteModel) RetryResumeApproval(ctx context.Context, input RetryResumeApprovalInput) (*RetryResumeApprovalOutput, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("approval write model not initialized")
	}
	if strings.TrimSpace(input.ApprovalID) == "" {
		return nil, fmt.Errorf("approval_id is required")
	}
	if strings.TrimSpace(input.TriggerID) == "" {
		return nil, fmt.Errorf("trigger_id is required")
	}
	payloadHash, err := ApprovalRetryResumePayloadHash(input)
	if err != nil {
		return nil, fmt.Errorf("hash retry resume payload: %w", err)
	}
	var result *RetryResumeApprovalOutput
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		approvalDAO := aidaoapproval.NewAIApprovalTaskDAO(tx)
		runDAO := aidao.NewAIRunDAO(tx)
		task, err := approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
		if err != nil {
			return fmt.Errorf("get approval task: %w", err)
		}
		if task == nil {
			return &ApprovalNotFoundError{ApprovalID: input.ApprovalID}
		}
		if task.UserID != 0 && task.UserID != input.UserID {
			return &ApprovalForbiddenError{ApprovalID: input.ApprovalID, UserID: input.UserID}
		}
		run, err := runDAO.GetRun(ctx, task.RunID)
		if err != nil || run == nil {
			return fmt.Errorf("run not found")
		}
		decidedEvent, err := findApprovalDecidedOutbox(ctx, tx, input.ApprovalID)
		if err != nil {
			return fmt.Errorf("load approval decided outbox: %w", err)
		}
		if replay := findRetryResumeIdempotencyResult(decidedEvent, input.TriggerID, payloadHash); replay != nil {
			result = replay
			return nil
		}
		if task.Status != "approved" {
			return &ApprovalConflictError{ApprovalID: input.ApprovalID, Message: fmt.Sprintf("approval not retryable from status %q", task.Status)}
		}
		switch strings.TrimSpace(run.Status) {
		case "resume_failed_retryable":
		case "resuming", "running":
			return &ApprovalConflictError{ApprovalID: input.ApprovalID, Message: fmt.Sprintf("run already %s", run.Status)}
		default:
			return &ApprovalConflictError{ApprovalID: input.ApprovalID, Message: fmt.Sprintf("run not retryable from status %q", run.Status)}
		}
		result = &RetryResumeApprovalOutput{ApprovalID: input.ApprovalID, Status: "queued", Message: "resume retry queued"}
		payloadBase := TaskDecisionPayload(task)
		if decidedEvent != nil && strings.TrimSpace(decidedEvent.PayloadJSON) != "" {
			existingPayload, err := DecodeApprovalEventPayload(decidedEvent.PayloadJSON)
			if err != nil {
				return fmt.Errorf("decode existing approval decided payload: %w", err)
			}
			payloadBase = existingPayload
		}
		payload := AttachApprovalRetryResume(payloadBase, input.TriggerID, payloadHash, result)
		if decidedEvent == nil {
			outboxDAO := aidaoapproval.NewAIApprovalOutboxDAO(tx)
			return m.writeEvent(ctx, tx, outboxDAO, task, event.ApprovalEventTypeDecided, payload)
		}
		raw, err := marshalEventPayload(decidedEvent.EventType, time.Now().UTC(), decidedEvent.Sequence, task, payload)
		if err != nil {
			return fmt.Errorf("marshal retry resume payload: %w", err)
		}
		return tx.WithContext(ctx).Model(&ai.AIApprovalOutboxEvent{}).
			Where("id = ?", decidedEvent.ID).
			Updates(map[string]any{
				"status":        "pending",
				"next_retry_at": nil,
				"payload_json":  raw,
				"updated_at":    time.Now().UTC(),
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ExpireApproval atomically marks a pending approval expired, updates the run state,
// and emits an approval-expired lifecycle event.
func (m *WriteModel) ExpireApproval(ctx context.Context, approvalID string) (bool, error) {
	if m == nil || m.db == nil {
		return false, fmt.Errorf("approval write model not initialized")
	}
	if strings.TrimSpace(approvalID) == "" {
		return false, fmt.Errorf("approval_id is required")
	}

	expired := false
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		approvalDAO := aidaoapproval.NewAIApprovalTaskDAO(tx)
		runDAO := aidao.NewAIRunDAO(tx)
		outboxDAO := aidaoapproval.NewAIApprovalOutboxDAO(tx)

		task, err := approvalDAO.GetByApprovalID(ctx, approvalID)
		if err != nil {
			return fmt.Errorf("load approval task: %w", err)
		}
		if task == nil {
			return &ApprovalNotFoundError{ApprovalID: approvalID}
		}
		if task.Status != "pending" {
			return nil
		}

		now := time.Now().UTC()
		result := tx.WithContext(ctx).Model(&ai.AIApprovalTask{}).
			Where("approval_id = ? AND status = ?", approvalID, "pending").
			Updates(map[string]any{
				"status":     "expired",
				"updated_at": now,
			})
		if result.Error != nil {
			return fmt.Errorf("mark approval expired: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return nil
		}

		task, err = approvalDAO.GetByApprovalID(ctx, approvalID)
		if err != nil {
			return fmt.Errorf("reload expired approval task: %w", err)
		}
		if task == nil {
			return &ApprovalNotFoundError{ApprovalID: approvalID}
		}

		if err := runDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "expired",
			ErrorMessage: "approval expired",
		}); err != nil {
			return fmt.Errorf("update expired run status: %w", err)
		}

		if err := m.writeEvent(ctx, tx, outboxDAO, task, event.ApprovalEventTypeExpired, TaskStatusExpiredPayload(task)); err != nil {
			return err
		}
		expired = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return expired, nil
}

// EmitRunResuming 发布运行恢复中事件。
func (m *WriteModel) EmitRunResuming(ctx context.Context, approvalID string) error {
	return m.emitLifecycle(ctx, approvalID, event.RunEventTypeResuming, "resuming")
}

// EmitRunResumed 发布运行已恢复事件。
func (m *WriteModel) EmitRunResumed(ctx context.Context, approvalID string) error {
	return m.emitLifecycle(ctx, approvalID, event.RunEventTypeResumed, "running")
}

// EmitRunCompleted 发布运行完成事件。
func (m *WriteModel) EmitRunCompleted(ctx context.Context, approvalID, runStatus string) error {
	if strings.TrimSpace(runStatus) == "" {
		runStatus = "completed"
	}
	return m.emitLifecycle(ctx, approvalID, event.RunEventTypeCompleted, runStatus)
}

// EmitRunResumeFailed 发布运行恢复失败事件。
func (m *WriteModel) EmitRunResumeFailed(ctx context.Context, approvalID string, retryable bool, cause error) error {
	status := "failed"
	if retryable {
		status = "resume_failed_retryable"
	}
	return m.emitLifecycle(ctx, approvalID, event.RunEventTypeResumeFailed, status)
}

// RenewApprovalLease 续期审批租约。
func (m *WriteModel) RenewApprovalLease(ctx context.Context, approvalID string, leaseExpiresAt time.Time) (bool, error) {
	if m == nil || m.db == nil {
		return false, fmt.Errorf("approval write model not initialized")
	}
	return aidaoapproval.NewAIApprovalTaskDAO(m.db).AcquireOrStealLease(ctx, approvalID, leaseExpiresAt)
}

// AcquireApprovalLease 获取审批租约。
func (m *WriteModel) AcquireApprovalLease(ctx context.Context, approvalID string, leaseExpiresAt time.Time) (bool, error) {
	return m.RenewApprovalLease(ctx, approvalID, leaseExpiresAt)
}

func (m *WriteModel) emitLifecycle(ctx context.Context, approvalID, eventType, runStatus string) error {
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
		if err != nil || run == nil {
			return fmt.Errorf("run not found")
		}
		payload := map[string]any{"run_id": run.ID, "session_id": task.SessionID, "status": runStatus}
		return m.writeEvent(ctx, tx, outboxDAO, task, eventType, payload)
	})
}

func (m *WriteModel) writeEvent(ctx context.Context, tx *gorm.DB, outboxDAO *aidaoapproval.AIApprovalOutboxDAO, task *ai.AIApprovalTask, eventType string, payload any) error {
	if outboxDAO == nil || task == nil {
		return fmt.Errorf("outbox dao and task are required")
	}
	sequence, err := outboxDAO.NextSequence(ctx, task.RunID)
	if err != nil {
		return fmt.Errorf("allocate approval outbox sequence: %w", err)
	}
	now := time.Now().UTC()
	eventRow := &ai.AIApprovalOutboxEvent{
		ApprovalID: task.ApprovalID, ToolCallID: task.ToolCallID, EventType: eventType,
		RunID: task.RunID, SessionID: task.SessionID, Status: "pending",
	}
	if p, ok := payload.(map[string]any); ok {
		raw, err := marshalEventPayload(eventType, now, sequence, task, p)
		if err != nil {
			return fmt.Errorf("marshal approval outbox payload: %w", err)
		}
		eventRow.PayloadJSON = raw
	}
	eventRow.Sequence = sequence
	eventRow.OccurredAt = now
	return outboxDAO.EnqueueOrTouch(ctx, eventRow)
}

func marshalEventPayload(eventType string, occurredAt time.Time, sequence int64, task *ai.AIApprovalTask, payload map[string]any) (string, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	payload["event_type"] = eventType
	payload["occurred_at"] = occurredAt.Format(time.RFC3339Nano)
	payload["sequence"] = sequence
	if task == nil {
		return "", fmt.Errorf("approval task is required")
	}
	payload["approval_id"] = task.ApprovalID
	payload["run_id"] = task.RunID
	payload["session_id"] = task.SessionID
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func findSubmitIdempotencyResult(ctx context.Context, tx *gorm.DB, approvalID, idempotencyKey, payloadHash string) (*SubmitApprovalOutput, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, nil
	}
	decidedEvent, err := findApprovalDecidedOutbox(ctx, tx, approvalID)
	if err != nil || decidedEvent == nil {
		return nil, err
	}
	payload, err := DecodeApprovalEventPayload(decidedEvent.PayloadJSON)
	if err != nil {
		return nil, nil
	}
	var record approvalSubmitIdempotencyRecord
	if !decodeApprovalRecord(payload["idempotency"], &record) {
		return nil, nil
	}
	if strings.TrimSpace(record.Key) != idempotencyKey {
		return nil, nil
	}
	if existing := strings.TrimSpace(record.PayloadHash); existing != "" && existing != strings.TrimSpace(payloadHash) {
		return nil, nil
	}
	if record.ResultSnapshot == nil {
		return nil, nil
	}
	snapshot := *record.ResultSnapshot
	return &snapshot, nil
}

func findRetryResumeIdempotencyResult(decidedEvent *ai.AIApprovalOutboxEvent, triggerID, payloadHash string) *RetryResumeApprovalOutput {
	if decidedEvent == nil || strings.TrimSpace(triggerID) == "" {
		return nil
	}
	payload, err := DecodeApprovalEventPayload(decidedEvent.PayloadJSON)
	if err != nil {
		return nil
	}
	var record approvalRetryResumeRecord
	if !decodeApprovalRecord(payload["retry_resume"], &record) {
		return nil
	}
	if strings.TrimSpace(record.TriggerID) != strings.TrimSpace(triggerID) {
		return nil
	}
	if existing := strings.TrimSpace(record.PayloadHash); existing != "" && existing != strings.TrimSpace(payloadHash) {
		return nil
	}
	if record.ResultSnapshot == nil {
		return nil
	}
	snapshot := *record.ResultSnapshot
	return &snapshot
}

func findApprovalDecidedOutbox(ctx context.Context, tx *gorm.DB, approvalID string) (*ai.AIApprovalOutboxEvent, error) {
	if tx == nil || strings.TrimSpace(approvalID) == "" {
		return nil, nil
	}
	var outbox ai.AIApprovalOutboxEvent
	err := tx.WithContext(ctx).
		Where("approval_id = ? AND event_type IN ?", approvalID, ApprovalDecidedEventTypes()).
		Order("id DESC").
		First(&outbox).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &outbox, nil
}

func decodeApprovalRecord(raw any, target any) bool {
	if raw == nil || target == nil {
		return false
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return false
	}
	return json.Unmarshal(b, target) == nil
}
