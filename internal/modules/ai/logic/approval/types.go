// Package approval 实现 AI 审批流程相关的业务逻辑。
package approval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
)

// SubmitApprovalInput 提交审批结果的输入参数。
type SubmitApprovalInput struct {
	ApprovalID       string
	Approved         bool
	DisapproveReason string
	Comment          string
	UserID           uint64
}

// SubmitApprovalOutput 提交审批结果的输出。
type SubmitApprovalOutput struct {
	ApprovalID string `json:"approval_id"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

// RetryResumeApprovalInput 重试恢复审批的输入参数。
type RetryResumeApprovalInput struct {
	ApprovalID string
	TriggerID  string
	UserID     uint64
}

// RetryResumeApprovalOutput 重试恢复审批的输出。
type RetryResumeApprovalOutput struct {
	ApprovalID string `json:"approval_id"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

// ResumeApprovalInput 恢复审批执行的输入参数。
type ResumeApprovalInput struct {
	SessionID  string
	ApprovalID string
	Approved   bool
	Reason     string
	Comment    string
	UserID     uint64
}

// ApprovalNotFoundError 审批任务不存在错误。
type ApprovalNotFoundError struct{ ApprovalID string }

func (e *ApprovalNotFoundError) Error() string {
	if e == nil || strings.TrimSpace(e.ApprovalID) == "" {
		return "approval not found"
	}
	return fmt.Sprintf("approval %q not found", e.ApprovalID)
}

// ApprovalForbiddenError 审批权限不足错误。
type ApprovalForbiddenError struct{ ApprovalID string; UserID uint64 }

func (e *ApprovalForbiddenError) Error() string {
	if e == nil || strings.TrimSpace(e.ApprovalID) == "" {
		return "approval does not belong to current user"
	}
	return fmt.Sprintf("approval %q does not belong to current user", e.ApprovalID)
}

// ApprovalConflictError 审批冲突错误。
type ApprovalConflictError struct{ ApprovalID, Message string }

func (e *ApprovalConflictError) Error() string {
	if e == nil {
		return "approval already handled"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.ApprovalID) == "" {
		return "approval already handled"
	}
	return fmt.Sprintf("approval %q already handled", e.ApprovalID)
}

type approvalSubmitIdempotencyRecord struct {
	Key, ApprovalID, PayloadHash string
	ResultSnapshot               *SubmitApprovalOutput `json:"result_snapshot,omitempty"`
}

type approvalRetryResumeRecord struct {
	TriggerID, ApprovalID, PayloadHash string
	ResultSnapshot                     *RetryResumeApprovalOutput `json:"result_snapshot,omitempty"`
}

type approvalSubmitIdempotencyKeyContextKey struct{}

// WithApprovalSubmitIdempotencyKey 将幂等键存入上下文。
func WithApprovalSubmitIdempotencyKey(ctx context.Context, key string) context.Context {
	trimmed := strings.TrimSpace(key)
	if ctx == nil || trimmed == "" {
		return ctx
	}
	return context.WithValue(ctx, approvalSubmitIdempotencyKeyContextKey{}, trimmed)
}

// ApprovalSubmitIdempotencyKeyFromContext 从上下文获取幂等键。
func ApprovalSubmitIdempotencyKeyFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	key, _ := ctx.Value(approvalSubmitIdempotencyKeyContextKey{}).(string)
	return strings.TrimSpace(key)
}

// ApprovalSubmitPayloadHash 计算审批提交负载哈希。
func ApprovalSubmitPayloadHash(input SubmitApprovalInput) (string, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// ApprovalRetryResumePayloadHash 计算重试恢复负载哈希。
func ApprovalRetryResumePayloadHash(input RetryResumeApprovalInput) (string, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

// DecodeApprovalEventPayload 解码审批事件负载。
func DecodeApprovalEventPayload(payloadJSON string) (map[string]any, error) {
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

// AttachApprovalSubmitIdempotency 附加幂等性记录。
func AttachApprovalSubmitIdempotency(payload map[string]any, idempotencyKey, payloadHash string, result *SubmitApprovalOutput) map[string]any {
	if payload == nil {
		payload = map[string]any{}
	}
	if strings.TrimSpace(idempotencyKey) == "" || result == nil {
		return payload
	}
	snapshot := *result
	payload["idempotency"] = approvalSubmitIdempotencyRecord{Key: idempotencyKey, ApprovalID: snapshot.ApprovalID, PayloadHash: payloadHash, ResultSnapshot: &snapshot}
	return payload
}

// AttachApprovalRetryResume 附加重试恢复记录。
func AttachApprovalRetryResume(payload map[string]any, triggerID, payloadHash string, result *RetryResumeApprovalOutput) map[string]any {
	if payload == nil {
		payload = map[string]any{}
	}
	if strings.TrimSpace(triggerID) == "" || result == nil {
		return payload
	}
	snapshot := *result
	payload["retry_resume"] = approvalRetryResumeRecord{TriggerID: triggerID, ApprovalID: snapshot.ApprovalID, PayloadHash: payloadHash, ResultSnapshot: &snapshot}
	return payload
}

// TaskDecisionPayload 构建审批决策负载。
func TaskDecisionPayload(task *ai.AIApprovalTask) map[string]any {
	payload := map[string]any{
		"approval_id": task.ApprovalID, "run_id": task.RunID, "session_id": task.SessionID,
		"status": task.Status, "approved": task.Status == "approved", "approved_by": task.ApprovedBy,
		"comment": task.Comment, "disapprove_reason": task.DisapproveReason,
	}
	if task.DecidedAt != nil {
		payload["decided_at"] = task.DecidedAt.UTC().Format(time.RFC3339)
	}
	return payload
}

// TaskStatusExpiredPayload 构建审批过期负载。
func TaskStatusExpiredPayload(task *ai.AIApprovalTask) map[string]any {
	payload := map[string]any{
		"approval_id": task.ApprovalID, "run_id": task.RunID, "session_id": task.SessionID,
		"status": task.Status, "expired": true,
	}
	if task.ExpiresAt != nil {
		payload["expires_at"] = task.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return payload
}

// TaskDecisionPayloadWithIdempotency 构建带幂等性的审批决策负载。
func TaskDecisionPayloadWithIdempotency(task *ai.AIApprovalTask, idempotencyKey, payloadHash string, result *SubmitApprovalOutput) map[string]any {
	return AttachApprovalSubmitIdempotency(TaskDecisionPayload(task), idempotencyKey, payloadHash, result)
}

// TaskStatusExpiredPayloadWithIdempotency 构建带幂等性的审批过期负载。
func TaskStatusExpiredPayloadWithIdempotency(task *ai.AIApprovalTask, idempotencyKey, payloadHash string, result *SubmitApprovalOutput) map[string]any {
	return AttachApprovalSubmitIdempotency(TaskStatusExpiredPayload(task), idempotencyKey, payloadHash, result)
}

// AlreadyHandledError 构建已处理错误。
func AlreadyHandledError(task *ai.AIApprovalTask) error {
	if task == nil {
		return &ApprovalConflictError{}
	}
	return &ApprovalConflictError{ApprovalID: task.ApprovalID, Message: fmt.Sprintf("approval already %s", task.Status)}
}
