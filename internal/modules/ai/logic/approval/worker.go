// Package approval 实现 AI 审批流程相关的业务逻辑。
package approval

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	sharedapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/shared/approval"
	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/event"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

const (
	DefaultLeaseWindow    = 2 * time.Minute
	DefaultRetryDelay     = 5 * time.Second
	DefaultPollInterval   = 2 * time.Second
	OutboxProcessingLease = 2 * time.Minute
)

const legacyApprovalDecidedEventType = "approval_decided"

// ResumeFunc 定义审批恢复函数类型。
type ResumeFunc func(context.Context, *ai.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error)

// ApprovalDecidedEventTypes 返回所有审批决策事件类型。
func ApprovalDecidedEventTypes() []string {
	return []string{event.ApprovalEventTypeDecided, legacyApprovalDecidedEventType}
}

// IsApprovalDecidedEventType 判断事件类型是否为审批决策事件。
func IsApprovalDecidedEventType(eventType string) bool {
	normalized := strings.TrimSpace(eventType)
	for _, candidate := range ApprovalDecidedEventTypes() {
		if normalized == candidate {
			return true
		}
	}
	return false
}

// Logic 定义审批 Worker 所需的最小 Logic 依赖。
type Logic struct {
	SvcCtx          *svc.ServiceContext
	ChatDAO         *aidaochat.AIChatDAO
	RunDAO          *aidao.AIRunDAO
	RunEventDAO     *aidao.AIRunEventDAO
	ApprovalDAO     *aidaoapproval.AIApprovalTaskDAO
	AIRouter        adk.ResumableAgent
	CheckpointStore adk.CheckPointStore
}

// Worker 审批恢复 Worker。
type Worker struct {
	logic       *Logic
	leaseWindow time.Duration
	retryDelay  time.Duration
	now         func() time.Time
	resume      ResumeFunc
}

// WorkerOption 配置选项。
type WorkerOption func(*Worker)

// WithWorkerResume 设置自定义恢复函数。
func WithWorkerResume(fn ResumeFunc) WorkerOption {
	return func(w *Worker) {
		if fn != nil {
			w.resume = fn
		}
	}
}

// WithWorkerClock 设置自定义时钟。
func WithWorkerClock(now func() time.Time) WorkerOption {
	return func(w *Worker) {
		if now != nil {
			w.now = now
		}
	}
}

// WithWorkerLeaseWindow 设置租约窗口。
func WithWorkerLeaseWindow(d time.Duration) WorkerOption {
	return func(w *Worker) {
		if d > 0 {
			w.leaseWindow = d
		}
	}
}

// WithWorkerRetryDelay 设置重试延迟。
func WithWorkerRetryDelay(d time.Duration) WorkerOption {
	return func(w *Worker) {
		if d > 0 {
			w.retryDelay = d
		}
	}
}

// NewWorker 创建审批恢复 Worker 实例。
func NewWorker(l *Logic, opts ...WorkerOption) *Worker {
	w := &Worker{
		logic:       l,
		leaseWindow: DefaultLeaseWindow,
		retryDelay:  DefaultRetryDelay,
		now:         time.Now,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(w)
		}
	}
	return w
}

// WithClock 设置自定义时钟。
func (w *Worker) WithClock(now func() time.Time) *Worker {
	if now != nil {
		w.now = now
	}
	return w
}

// WithLeaseWindow 设置租约窗口。
func (w *Worker) WithLeaseWindow(d time.Duration) *Worker {
	if d > 0 {
		w.leaseWindow = d
	}
	return w
}

// WithRetryDelay 设置重试延迟。
func (w *Worker) WithRetryDelay(d time.Duration) *Worker {
	if d > 0 {
		w.retryDelay = d
	}
	return w
}

// RunLoop 运行主循环。
func (w *Worker) RunLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		claimed, _ := w.RunOnce(ctx)
		if claimed {
			continue
		}
	}
}

// RunOnce 执行一次轮询。
func (w *Worker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil || w.logic == nil || w.logic.SvcCtx == nil || w.logic.SvcCtx.DB == nil {
		return false, nil
	}
	outboxDAO := aidaoapproval.NewAIApprovalOutboxDAO(w.logic.SvcCtx.DB)
	e, err := w.claimOutboxEvent(ctx)
	if err != nil {
		return false, err
	}
	if e == nil {
		return false, nil
	}
	if err := w.processEvent(ctx, e); err != nil {
		nextRetryAt := w.now().Add(w.retryBackoff(e.RetryCount))
		if markErr := outboxDAO.MarkRetry(ctx, e.ID, nextRetryAt); markErr != nil {
			return true, err
		}
		return true, err
	}
	if err := outboxDAO.MarkDone(ctx, e.ID); err != nil {
		// Log the MarkDone failure but do not retry the event processing.
		// Retrying after successful processEvent risks double-resume (e.g., resuming the run twice).
		// A separate reconciliation job should handle stuck "processing" events.
		logger.L().Infof("approval worker: failed to mark outbox event %d as done: %v (event was processed successfully)", []any{e.ID, err})
	}
	return true, nil
}

const maxClaimOutboxAttempts = 20

func (w *Worker) claimOutboxEvent(ctx context.Context) (*ai.AIApprovalOutboxEvent, error) {
	db := w.logic.SvcCtx.DB
	var claimed *ai.AIApprovalOutboxEvent
	for attempt := 0; attempt < maxClaimOutboxAttempts; attempt++ {
		var candidate ai.AIApprovalOutboxEvent
		hadCandidate := false
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			now := w.now()
			staleBefore := now.Add(-OutboxProcessingLease)
			decisionTypes := ApprovalDecidedEventTypes()
			query := tx.Where(
				"event_type IN ? AND ((status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND updated_at <= ?))",
				decisionTypes, "pending", now, "processing", staleBefore,
			).Order("next_retry_at ASC").Order("created_at ASC").Order("id ASC")
			if err := query.First(&candidate).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			hadCandidate = true
			result := tx.Model(&ai.AIApprovalOutboxEvent{}).
				Where("id = ? AND event_type IN ? AND ((status = ?) OR (status = ? AND updated_at <= ?))",
					candidate.ID, decisionTypes, "pending", "processing", staleBefore).
				Updates(map[string]any{"status": "processing", "updated_at": now})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}
			candidate.Status = "processing"
			candidate.UpdatedAt = now
			claimed = &candidate
			return nil
		})
		if err != nil {
			return nil, err
		}
		if claimed != nil {
			return claimed, nil
		}
		if !hadCandidate {
			return nil, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		delay := time.Duration(10<<min(attempt, 5)) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, nil
}

func (w *Worker) processEvent(ctx context.Context, e *ai.AIApprovalOutboxEvent) error {
	if e == nil || !IsApprovalDecidedEventType(e.EventType) {
		return nil
	}
	task, err := w.logic.ApprovalDAO.GetByApprovalID(ctx, e.ApprovalID)
	if err != nil {
		return err
	}
	if task == nil {
		return &ApprovalNotFoundError{ApprovalID: e.ApprovalID}
	}
	now := w.now()
	if task.ExpiresAt != nil && task.ExpiresAt.Before(now) {
		return w.expireAndFinalize(ctx, task)
	}
	switch task.Status {
	case "approved":
		return w.resumeApproved(ctx, task)
	case "rejected", "expired":
		return w.finalize(ctx, task, "cancelled", "approval "+task.Status)
	default:
		return &ApprovalConflictError{ApprovalID: task.ApprovalID, Message: "approval not in resumable state"}
	}
}

func (w *Worker) retryBackoff(retryCount int) time.Duration {
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

func (w *Worker) expireAndFinalize(ctx context.Context, task *ai.AIApprovalTask) error {
	if w == nil || w.logic == nil || w.logic.SvcCtx == nil || w.logic.SvcCtx.DB == nil {
		return fmt.Errorf("approval worker not initialized")
	}
	_, err := NewWriteModel(w.logic.SvcCtx.DB).ExpireApproval(ctx, task.ApprovalID)
	return err
}

func (w *Worker) finalize(ctx context.Context, task *ai.AIApprovalTask, runStatus, errMsg string) error {
	if task == nil || w.logic.RunDAO == nil {
		return fmt.Errorf("run finalizer not initialized")
	}
	return w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{Status: runStatus, ErrorMessage: errMsg})
}

func (w *Worker) resumeApproved(ctx context.Context, task *ai.AIApprovalTask) error {
	if task == nil {
		return fmt.Errorf("approval task is nil")
	}
	if strings.TrimSpace(task.ResumeTargetID) == "" {
		return &ApprovalConflictError{ApprovalID: task.ApprovalID, Message: "approval resume target is missing"}
	}
	if w.resume == nil {
		return fmt.Errorf("approval resume handler not configured")
	}
	_, err := w.resume(ctx, task, buildResumeParams(task))
	return err
}

func buildResumeParams(task *ai.AIApprovalTask) *adk.ResumeParams {
	if task == nil || strings.TrimSpace(task.ResumeTargetID) == "" {
		return nil
	}

	result := &sharedapproval.ApprovalResult{
		Approved: true,
		Comment:  task.Comment,
	}
	if task.ApprovedBy != 0 {
		result.ApprovedBy = strconv.FormatUint(task.ApprovedBy, 10)
	}
	if task.DecidedAt != nil {
		approvedAt := task.DecidedAt.UTC()
		result.ApprovedAt = &approvedAt
	}
	if reason := strings.TrimSpace(task.DisapproveReason); reason != "" {
		result.DisapproveReason = &reason
	}

	return &adk.ResumeParams{
		Targets: map[string]any{
			task.ResumeTargetID: result,
		},
	}
}
