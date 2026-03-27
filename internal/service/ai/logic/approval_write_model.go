// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件实现审批写入模型，封装审批提交的幂等性、事务性和事件发布逻辑。
//
// 核心功能:
//   - 幂等性审批提交（防止重复提交）
//   - 乐观锁和租约机制（防止并发冲突）
//   - 审批过期自动处理
//   - 审批事件发布
package logic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// ApprovalWriteModel 审批写入模型。
//
// 封装审批相关的数据库写操作和事件发布，确保操作的原子性和幂等性。
type ApprovalWriteModel struct {
	db *gorm.DB
}

// NewApprovalWriteModel 创建审批写入模型实例。
func NewApprovalWriteModel(db *gorm.DB) *ApprovalWriteModel {
	if db == nil {
		return &ApprovalWriteModel{}
	}
	return &ApprovalWriteModel{db: db}
}

func (l *Logic) approvalWriteModel() *ApprovalWriteModel {
	if l == nil || l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil
	}
	return NewApprovalWriteModel(l.svcCtx.DB)
}

// SubmitApproval 提交审批结果。
//
// 实现幂等的审批提交，支持以下场景:
//   - 首次提交：记录审批结果并发布事件
//   - 重复提交：返回已缓存的结果（幂等）
//   - 并发冲突：返回当前状态
//   - 审批过期：自动标记过期并发布事件
//
// 参数:
//   - ctx: 上下文
//   - input: 审批提交输入
//
// 返回: 审批结果输出
func (m *ApprovalWriteModel) SubmitApproval(ctx context.Context, input SubmitApprovalInput) (*SubmitApprovalOutput, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("approval write model not initialized")
	}
	if strings.TrimSpace(input.ApprovalID) == "" {
		return nil, fmt.Errorf("approval_id is required")
	}

	idempotencyKey := ApprovalSubmitIdempotencyKeyFromContext(ctx)
	payloadHash, err := approvalSubmitPayloadHash(input)
	if err != nil {
		return nil, fmt.Errorf("hash approval submit payload: %w", err)
	}

	var result *SubmitApprovalOutput
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		approvalDAO := aidao.NewAIApprovalTaskDAO(tx)
		outboxDAO := aidao.NewAIApprovalOutboxDAO(tx)

		cachedResult, err := loadApprovalSubmitIdempotencyResult(ctx, tx, input.ApprovalID, idempotencyKey, payloadHash)
		if err != nil {
			return err
		}
		if cachedResult != nil {
			result = cachedResult
			return nil
		}

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
			replayed, ok, err := loadDuplicateSubmitReplayResult(ctx, tx, task, input)
			if err != nil {
				return err
			}
			if ok {
				result = replayed
				return nil
			}
			return approvalAlreadyHandledError(task)
		}
		if task.LockExpiresAt != nil && task.LockExpiresAt.After(now) {
			result = &SubmitApprovalOutput{
				ApprovalID: input.ApprovalID,
				Status:     task.Status,
				Message:    "approval is currently being processed",
			}
			return nil
		}

		if task.ExpiresAt != nil && task.ExpiresAt.Before(now) {
			update := tx.WithContext(ctx).
				Model(&model.AIApprovalTask{}).
				Where("approval_id = ? AND status = ?", input.ApprovalID, "pending").
				Updates(map[string]any{
					"status":     "expired",
					"updated_at": now,
				})
			if update.Error != nil {
				return fmt.Errorf("mark approval expired: %w", update.Error)
			}
			if update.RowsAffected == 0 {
				task, err = approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
				if err != nil {
					return fmt.Errorf("reload approval task: %w", err)
				}
				if task == nil {
					return &ApprovalNotFoundError{ApprovalID: input.ApprovalID}
				}
				if task.Status != "pending" {
					return approvalAlreadyHandledError(task)
				}
				result = &SubmitApprovalOutput{
					ApprovalID: input.ApprovalID,
					Status:     task.Status,
					Message:    "approval is currently being processed",
				}
				return nil
			}
			task, err = approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
			if err != nil {
				return fmt.Errorf("reload expired task: %w", err)
			}
			if task == nil {
				return &ApprovalNotFoundError{ApprovalID: input.ApprovalID}
			}
			result = &SubmitApprovalOutput{
				ApprovalID: input.ApprovalID,
				Status:     "expired",
				Message:    "approval has expired",
			}
			if err := m.writeApprovalEvent(ctx, tx, outboxDAO, task, ApprovalEventTypeExpired, taskStatusExpiredPayloadWithIdempotency(task, idempotencyKey, payloadHash, result)); err != nil {
				return fmt.Errorf("enqueue approval_expired outbox: %w", err)
			}
			return nil
		}

		if input.Approved {
			updated, err := approvalDAO.ApproveWithLease(ctx, input.ApprovalID, input.UserID, input.Comment, now.Add(approvalWorkerDefaultLeaseWindow))
			if err != nil {
				return fmt.Errorf("approve approval task: %w", err)
			}
			if !updated {
				task, err = approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
				if err != nil {
					return fmt.Errorf("reload approval task: %w", err)
				}
				if task == nil {
					return &ApprovalNotFoundError{ApprovalID: input.ApprovalID}
				}
				if task.Status != "pending" {
					return approvalAlreadyHandledError(task)
				}
				result = &SubmitApprovalOutput{
					ApprovalID: input.ApprovalID,
					Status:     task.Status,
					Message:    "approval is currently being processed",
				}
				return nil
			}
		} else {
			updated, err := approvalDAO.RejectPending(ctx, input.ApprovalID, input.UserID, input.DisapproveReason, input.Comment)
			if err != nil {
				return fmt.Errorf("reject approval task: %w", err)
			}
			if !updated {
				task, err = approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
				if err != nil {
					return fmt.Errorf("reload approval task: %w", err)
				}
				if task == nil {
					return &ApprovalNotFoundError{ApprovalID: input.ApprovalID}
				}
				if task.Status != "pending" {
					return approvalAlreadyHandledError(task)
				}
				result = &SubmitApprovalOutput{
					ApprovalID: input.ApprovalID,
					Status:     task.Status,
					Message:    "approval is currently being processed",
				}
				return nil
			}
		}

		task, err = approvalDAO.GetByApprovalID(ctx, input.ApprovalID)
		if err != nil {
			return fmt.Errorf("reload updated approval task: %w", err)
		}
		if task == nil {
			return &ApprovalNotFoundError{ApprovalID: input.ApprovalID}
		}

		result = &SubmitApprovalOutput{
			ApprovalID: input.ApprovalID,
			Status:     task.Status,
			Message:    fmt.Sprintf("approval %s successfully", task.Status),
		}
		if err := m.writeApprovalEvent(ctx, tx, outboxDAO, task, ApprovalEventTypeDecided, taskDecisionPayloadWithIdempotency(task, idempotencyKey, payloadHash, result)); err != nil {
			return fmt.Errorf("enqueue approval_decided outbox: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type approvalSubmitIdempotencyRecord struct {
	Key            string                `json:"key"`
	ApprovalID     string                `json:"approval_id"`
	PayloadHash    string                `json:"payload_hash"`
	ResultSnapshot *SubmitApprovalOutput `json:"result_snapshot,omitempty"`
}

type approvalRetryResumeRecord struct {
	TriggerID      string                     `json:"trigger_id"`
	ApprovalID     string                     `json:"approval_id"`
	PayloadHash    string                     `json:"payload_hash"`
	ResultSnapshot *RetryResumeApprovalOutput `json:"result_snapshot,omitempty"`
}

func approvalSubmitPayloadHash(input SubmitApprovalInput) (string, error) {
	payload, err := json.Marshal(struct {
		ApprovalID       string `json:"approval_id"`
		Approved         bool   `json:"approved"`
		DisapproveReason string `json:"disapprove_reason,omitempty"`
		Comment          string `json:"comment,omitempty"`
		UserID           uint64 `json:"user_id"`
	}{
		ApprovalID:       input.ApprovalID,
		Approved:         input.Approved,
		DisapproveReason: input.DisapproveReason,
		Comment:          input.Comment,
		UserID:           input.UserID,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func (m *ApprovalWriteModel) RetryResumeApproval(ctx context.Context, input RetryResumeApprovalInput) (*RetryResumeApprovalOutput, error) {
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("approval write model not initialized")
	}
	if strings.TrimSpace(input.ApprovalID) == "" {
		return nil, fmt.Errorf("approval_id is required")
	}
	if strings.TrimSpace(input.TriggerID) == "" {
		return nil, fmt.Errorf("trigger_id is required")
	}

	payloadHash, err := approvalRetryResumePayloadHash(input)
	if err != nil {
		return nil, fmt.Errorf("hash retry resume payload: %w", err)
	}

	var result *RetryResumeApprovalOutput
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		approvalDAO := aidao.NewAIApprovalTaskDAO(tx)
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
		if err != nil {
			return fmt.Errorf("get run: %w", err)
		}
		if run == nil {
			return fmt.Errorf("run not found")
		}

		var outbox model.AIApprovalOutboxEvent
		hasOutbox := true
		if err := tx.WithContext(ctx).
			Where("approval_id = ? AND event_type = ?", input.ApprovalID, ApprovalEventTypeDecided).
			First(&outbox).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				hasOutbox = false
			} else {
				return fmt.Errorf("load approval decided outbox: %w", err)
			}
		}

		if hasOutbox {
			record, ok, err := decodeApprovalRetryResumeRecord(outbox.PayloadJSON)
			if err != nil {
				return fmt.Errorf("decode retry resume idempotency record: %w", err)
			}
			if ok && record != nil {
				if record.TriggerID == input.TriggerID {
					if record.PayloadHash != payloadHash {
						return &ApprovalConflictError{
							ApprovalID: input.ApprovalID,
							Message:    "trigger_id already used with different retry payload",
						}
					}
					if record.ResultSnapshot != nil {
						snapshot := *record.ResultSnapshot
						result = &snapshot
						return nil
					}
				}
				if record.TriggerID != "" && record.TriggerID != input.TriggerID && (outbox.Status == "pending" || outbox.Status == "processing") {
					return &ApprovalConflictError{
						ApprovalID: input.ApprovalID,
						Message:    "resume retry already queued",
					}
				}
			}
		}

		if task.Status != "approved" {
			return &ApprovalConflictError{
				ApprovalID: input.ApprovalID,
				Message:    fmt.Sprintf("approval %q is not retryable from status %q", input.ApprovalID, task.Status),
			}
		}

		switch strings.TrimSpace(run.Status) {
		case "resume_failed_retryable":
		case "resuming", "running":
			return &ApprovalConflictError{
				ApprovalID: input.ApprovalID,
				Message:    fmt.Sprintf("run %q is already %s", run.ID, run.Status),
			}
		default:
			return &ApprovalConflictError{
				ApprovalID: input.ApprovalID,
				Message:    fmt.Sprintf("run %q is not retryable from status %q", run.ID, run.Status),
			}
		}

		result = &RetryResumeApprovalOutput{
			ApprovalID: input.ApprovalID,
			Status:     "queued",
			Message:    "resume retry queued",
		}

		payload, err := decodeApprovalEventPayload(outbox.PayloadJSON)
		if err != nil {
			return fmt.Errorf("decode approval decided payload: %w", err)
		}
		payload = attachApprovalRetryResume(payload, input.TriggerID, payloadHash, result)

		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal retry resume payload: %w", err)
		}

		now := time.Now().UTC()
		if hasOutbox {
			if err := tx.WithContext(ctx).
				Model(&model.AIApprovalOutboxEvent{}).
				Where("id = ?", outbox.ID).
				Updates(map[string]any{
					"status":        "pending",
					"payload_json":  string(payloadJSON),
					"next_retry_at": nil,
					"updated_at":    now,
				}).Error; err != nil {
				return fmt.Errorf("requeue approval decided outbox: %w", err)
			}
			return nil
		}

		return m.writeApprovalEvent(ctx, tx, aidao.NewAIApprovalOutboxDAO(tx), task, ApprovalEventTypeDecided, payload)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func approvalAlreadyHandledError(task *model.AIApprovalTask) error {
	if task == nil {
		return &ApprovalConflictError{}
	}
	return &ApprovalConflictError{
		ApprovalID: task.ApprovalID,
		Message:    fmt.Sprintf("approval already %s", task.Status),
	}
}

func loadApprovalSubmitIdempotencyResult(ctx context.Context, tx *gorm.DB, approvalID, idempotencyKey, payloadHash string) (*SubmitApprovalOutput, error) {
	if tx == nil || strings.TrimSpace(idempotencyKey) == "" {
		return nil, nil
	}

	var events []model.AIApprovalOutboxEvent
	if err := tx.WithContext(ctx).
		Where("approval_id = ? AND event_type IN ?", approvalID, []string{ApprovalEventTypeDecided, ApprovalEventTypeExpired}).
		Order("sequence DESC, id DESC").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("load approval submit idempotency records: %w", err)
	}

	for _, event := range events {
		record, ok, err := decodeApprovalSubmitIdempotencyRecord(event.PayloadJSON)
		if err != nil {
			return nil, fmt.Errorf("decode approval submit idempotency record: %w", err)
		}
		if !ok || record == nil || record.Key != idempotencyKey {
			continue
		}
		if record.PayloadHash != payloadHash {
			return nil, &ApprovalConflictError{
				ApprovalID: approvalID,
				Message:    "idempotency key already used with different approval payload",
			}
		}
		if record.ResultSnapshot == nil {
			return nil, nil
		}
		snapshot := *record.ResultSnapshot
		return &snapshot, nil
	}
	return nil, nil
}

func loadDuplicateSubmitReplayResult(ctx context.Context, tx *gorm.DB, task *model.AIApprovalTask, input SubmitApprovalInput) (*SubmitApprovalOutput, bool, error) {
	if tx == nil || task == nil {
		return nil, false, nil
	}
	if !matchesDuplicateSubmitTask(task, input) {
		return nil, false, nil
	}

	var event model.AIApprovalOutboxEvent
	if err := tx.WithContext(ctx).
		Where("approval_id = ? AND event_type = ?", input.ApprovalID, ApprovalEventTypeDecided).
		Order("sequence DESC, id DESC").
		First(&event).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("load duplicate submit replay event: %w", err)
	}

	return &SubmitApprovalOutput{
		ApprovalID: input.ApprovalID,
		Status:     task.Status,
		Message:    fmt.Sprintf("approval %s successfully", task.Status),
	}, true, nil
}

func matchesDuplicateSubmitTask(task *model.AIApprovalTask, input SubmitApprovalInput) bool {
	if task == nil {
		return false
	}
	expectedStatus := "rejected"
	if input.Approved {
		expectedStatus = "approved"
	}
	if strings.TrimSpace(task.Status) != expectedStatus {
		return false
	}
	if task.ApprovedBy != input.UserID {
		return false
	}
	if strings.TrimSpace(task.Comment) != strings.TrimSpace(input.Comment) {
		return false
	}
	if strings.TrimSpace(task.DisapproveReason) != strings.TrimSpace(input.DisapproveReason) {
		return false
	}
	return true
}

func decodeApprovalSubmitIdempotencyRecord(payloadJSON string) (*approvalSubmitIdempotencyRecord, bool, error) {
	if strings.TrimSpace(payloadJSON) == "" {
		return nil, false, nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return nil, false, err
	}
	rawRecord, ok := payload["idempotency"]
	if !ok || rawRecord == nil {
		return nil, false, nil
	}
	encoded, err := json.Marshal(rawRecord)
	if err != nil {
		return nil, false, err
	}
	var record approvalSubmitIdempotencyRecord
	if err := json.Unmarshal(encoded, &record); err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

func decodeApprovalRetryResumeRecord(payloadJSON string) (*approvalRetryResumeRecord, bool, error) {
	payload, err := decodeApprovalEventPayload(payloadJSON)
	if err != nil {
		return nil, false, err
	}
	rawRecord, ok := payload["retry_resume"]
	if !ok || rawRecord == nil {
		return nil, false, nil
	}
	encoded, err := json.Marshal(rawRecord)
	if err != nil {
		return nil, false, err
	}
	var record approvalRetryResumeRecord
	if err := json.Unmarshal(encoded, &record); err != nil {
		return nil, false, err
	}
	return &record, true, nil
}

func attachApprovalSubmitIdempotency(payload map[string]any, idempotencyKey, payloadHash string, result *SubmitApprovalOutput) map[string]any {
	if payload == nil {
		payload = map[string]any{}
	}
	if strings.TrimSpace(idempotencyKey) == "" || result == nil {
		return payload
	}
	snapshot := *result
	payload["idempotency"] = approvalSubmitIdempotencyRecord{
		Key:            idempotencyKey,
		ApprovalID:     snapshot.ApprovalID,
		PayloadHash:    payloadHash,
		ResultSnapshot: &snapshot,
	}
	return payload
}

func attachApprovalRetryResume(payload map[string]any, triggerID, payloadHash string, result *RetryResumeApprovalOutput) map[string]any {
	if payload == nil {
		payload = map[string]any{}
	}
	if strings.TrimSpace(triggerID) == "" || result == nil {
		return payload
	}
	snapshot := *result
	payload["retry_resume"] = approvalRetryResumeRecord{
		TriggerID:      triggerID,
		ApprovalID:     snapshot.ApprovalID,
		PayloadHash:    payloadHash,
		ResultSnapshot: &snapshot,
	}
	return payload
}

func decodeApprovalEventPayload(payloadJSON string) (map[string]any, error) {
	if strings.TrimSpace(payloadJSON) == "" {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return map[string]any{}, nil
	}
	return payload, nil
}

func approvalRetryResumePayloadHash(input RetryResumeApprovalInput) (string, error) {
	payload, err := json.Marshal(struct {
		ApprovalID string `json:"approval_id"`
		TriggerID  string `json:"trigger_id"`
		UserID     uint64 `json:"user_id"`
	}{
		ApprovalID: input.ApprovalID,
		TriggerID:  input.TriggerID,
		UserID:     input.UserID,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func (m *ApprovalWriteModel) EmitRunResuming(ctx context.Context, approvalID string) error {
	return m.emitRunLifecycleEvent(ctx, approvalID, RunEventTypeResuming, "resuming", map[string]any{
		"status": "resuming",
	})
}

func (m *ApprovalWriteModel) EmitRunResumed(ctx context.Context, approvalID string) error {
	return m.emitRunLifecycleEvent(ctx, approvalID, RunEventTypeResumed, "running", map[string]any{
		"status": "running",
	})
}

func (m *ApprovalWriteModel) EmitRunCompleted(ctx context.Context, approvalID, runStatus string) error {
	if strings.TrimSpace(runStatus) == "" {
		runStatus = "completed"
	}
	return m.emitRunLifecycleEvent(ctx, approvalID, RunEventTypeCompleted, runStatus, map[string]any{
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
	return m.emitRunLifecycleEvent(ctx, approvalID, RunEventTypeResumeFailed, status, payload)
}

func (m *ApprovalWriteModel) RenewApprovalLease(ctx context.Context, approvalID string, leaseExpiresAt time.Time) (bool, error) {
	if m == nil || m.db == nil {
		return false, fmt.Errorf("approval write model not initialized")
	}
	dao := aidao.NewAIApprovalTaskDAO(m.db)
	return dao.AcquireOrStealLease(ctx, approvalID, leaseExpiresAt)
}

func (m *ApprovalWriteModel) AcquireApprovalLease(ctx context.Context, approvalID string, leaseExpiresAt time.Time) (bool, error) {
	if m == nil || m.db == nil {
		return false, fmt.Errorf("approval write model not initialized")
	}
	dao := aidao.NewAIApprovalTaskDAO(m.db)
	return dao.AcquireOrStealLease(ctx, approvalID, leaseExpiresAt)
}

func (m *ApprovalWriteModel) emitRunLifecycleEvent(ctx context.Context, approvalID, eventType, runStatus string, payload map[string]any) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("approval write model not initialized")
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		approvalDAO := aidao.NewAIApprovalTaskDAO(tx)
		runDAO := aidao.NewAIRunDAO(tx)
		outboxDAO := aidao.NewAIApprovalOutboxDAO(tx)

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
		case RunEventTypeResuming:
			payload["run_id"] = run.ID
			payload["session_id"] = task.SessionID
		case RunEventTypeResumed:
			payload["run_id"] = run.ID
			payload["session_id"] = task.SessionID
		case RunEventTypeCompleted:
			payload["run_id"] = run.ID
			payload["session_id"] = task.SessionID
		case RunEventTypeResumeFailed:
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
	case RunEventTypeResumeFailed:
		return runStatus == "resume_failed_retryable"
	default:
		return true
	}
}

func (m *ApprovalWriteModel) writeApprovalEvent(ctx context.Context, tx *gorm.DB, outboxDAO *aidao.AIApprovalOutboxDAO, task *model.AIApprovalTask, eventType string, payload any) error {
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
	event := &model.AIApprovalOutboxEvent{
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
	return outboxDAO.EnqueueOrTouch(ctx, event)
}

func buildApprovalEventEnvelope(eventType string, sequence int64, occurredAt time.Time, task *model.AIApprovalTask, payload any) (*ApprovalEventEnvelope, error) {
	switch eventType {
	case ApprovalEventTypeRequested:
		return NewApprovalRequestedEnvelope(ApprovalRequestedInput{
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
	case ApprovalEventTypeDecided:
		return NewApprovalDecidedEnvelope(ApprovalDecidedInput{
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
	case ApprovalEventTypeExpired:
		return NewApprovalExpiredEnvelope(ApprovalExpiredInput{
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
	case RunEventTypeResuming:
		return NewRunResumingEnvelope(RunResumingInput{
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
	case RunEventTypeResumed:
		return NewRunResumedEnvelope(RunResumedInput{
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
	case RunEventTypeResumeFailed:
		return NewRunResumeFailedEnvelope(RunResumeFailedInput{
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
	case RunEventTypeCompleted:
		return NewRunCompletedEnvelope(RunCompletedInput{
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

func taskDecisionPayload(task *model.AIApprovalTask) map[string]any {
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

func taskDecisionPayloadWithIdempotency(task *model.AIApprovalTask, idempotencyKey, payloadHash string, result *SubmitApprovalOutput) map[string]any {
	return attachApprovalSubmitIdempotency(taskDecisionPayload(task), idempotencyKey, payloadHash, result)
}

func taskStatusExpiredPayload(task *model.AIApprovalTask) map[string]any {
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

func taskStatusExpiredPayloadWithIdempotency(task *model.AIApprovalTask, idempotencyKey, payloadHash string, result *SubmitApprovalOutput) map[string]any {
	return attachApprovalSubmitIdempotency(taskStatusExpiredPayload(task), idempotencyKey, payloadHash, result)
}
