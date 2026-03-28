// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件实现审批恢复 Worker，负责处理审批通过后的 Run 恢复执行。
//
// 工作流程:
//  1. 轮询 Outbox 中待处理的审批决策事件
//  2. 验证审批状态和所有权
//  3. 通过 ADK ResumeWithParams 恢复执行
//  4. 消费恢复后的迭代器事件并持久化
//  5. 更新 Run 状态并发布完成事件
package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/common/approval"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"gorm.io/gorm"
)

// 默认配置常量。
const (
	approvalWorkerDefaultLeaseWindow    = 2 * time.Minute
	approvalWorkerDefaultRetryDelay     = 5 * time.Second
	approvalWorkerDefaultPollInterval   = 2 * time.Second
	approvalWorkerOutboxProcessingLease = 2 * time.Minute
)

const legacyApprovalDecidedEventType = "approval_decided"

func approvalDecidedEventTypes() []string {
	return []string{ApprovalEventTypeDecided, legacyApprovalDecidedEventType}
}

func isApprovalDecidedEventType(eventType string) bool {
	normalized := strings.TrimSpace(eventType)
	for _, candidate := range approvalDecidedEventTypes() {
		if normalized == candidate {
			return true
		}
	}
	return false
}

// approvalResumeFunc 审批恢复函数类型。
type approvalResumeFunc func(context.Context, *model.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error)

// ApprovalWorkerOption 审批 Worker 配置选项。
type ApprovalWorkerOption func(*ApprovalWorker)

// ApprovalWorker 审批恢复 Worker。
//
// 负责处理审批通过后的 Run 恢复执行，支持重试和熔断。
type ApprovalWorker struct {
	logic       *Logic
	leaseWindow time.Duration
	retryDelay  time.Duration
	now         func() time.Time
	resume      approvalResumeFunc
}

// NewApprovalWorker 创建审批恢复 Worker 实例。
func NewApprovalWorker(l *Logic, opts ...ApprovalWorkerOption) *ApprovalWorker {
	worker := &ApprovalWorker{
		logic:       l,
		leaseWindow: approvalWorkerDefaultLeaseWindow,
		retryDelay:  approvalWorkerDefaultRetryDelay,
		now:         time.Now,
	}
	worker.resume = worker.defaultResume
	for _, opt := range opts {
		if opt != nil {
			opt(worker)
		}
	}
	return worker
}

func WithApprovalWorkerResume(fn approvalResumeFunc) ApprovalWorkerOption {
	return func(worker *ApprovalWorker) {
		if fn != nil {
			worker.resume = fn
		}
	}
}

func WithApprovalWorkerClock(now func() time.Time) ApprovalWorkerOption {
	return func(worker *ApprovalWorker) {
		if now != nil {
			worker.now = now
		}
	}
}

func WithApprovalWorkerLeaseWindow(d time.Duration) ApprovalWorkerOption {
	return func(worker *ApprovalWorker) {
		if d > 0 {
			worker.leaseWindow = d
		}
	}
}

func WithApprovalWorkerRetryDelay(d time.Duration) ApprovalWorkerOption {
	return func(worker *ApprovalWorker) {
		if d > 0 {
			worker.retryDelay = d
		}
	}
}

func (w *ApprovalWorker) RunLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = approvalWorkerDefaultPollInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		claimed, _ := w.RunOnce(ctx)
		if claimed {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *ApprovalWorker) RunOnce(ctx context.Context) (bool, error) {
	if w == nil || w.logic == nil || w.logic.svcCtx == nil || w.logic.svcCtx.DB == nil {
		return false, nil
	}

	outboxDAO := aidao.NewAIApprovalOutboxDAO(w.logic.svcCtx.DB)
	event, err := w.claimApprovalDecidedOutboxEvent(ctx)
	if err != nil {
		return false, err
	}
	if event == nil {
		return false, nil
	}

	if err := w.processClaimedEvent(ctx, event); err != nil {
		nextRetryAt := w.now().Add(w.retryBackoff(event.RetryCount))
		if markErr := outboxDAO.MarkRetry(ctx, event.ID, nextRetryAt); markErr != nil {
			return true, fmt.Errorf("process approval outbox: %w; mark retry: %v", err, markErr)
		}
		return true, err
	}
	if err := outboxDAO.MarkDone(ctx, event.ID); err != nil {
		nextRetryAt := w.now().Add(w.retryBackoff(event.RetryCount))
		if markErr := outboxDAO.MarkRetry(ctx, event.ID, nextRetryAt); markErr != nil {
			return true, fmt.Errorf("mark outbox done: %w; mark retry: %v", err, markErr)
		}
		return true, err
	}
	return true, nil
}

func (w *ApprovalWorker) processClaimedEvent(ctx context.Context, event *model.AIApprovalOutboxEvent) error {
	if event == nil {
		return nil
	}
	if !isApprovalDecidedEventType(event.EventType) {
		return fmt.Errorf("approval worker claimed unsupported event type %q", strings.TrimSpace(event.EventType))
	}

	task, err := w.logic.approvalTaskDAO().GetByApprovalID(ctx, event.ApprovalID)
	if err != nil {
		return fmt.Errorf("load approval task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("approval task %q not found", event.ApprovalID)
	}
	task, err = w.reconcileApprovalTaskToolCallID(ctx, task, event)
	if err != nil {
		return err
	}
	if err := w.ensureApprovalOwnership(ctx, task); err != nil {
		return err
	}
	if run, converged := w.convergedRun(ctx, task); converged {
		return w.emitConvergedLifecycleEvent(ctx, task, run)
	}

	now := w.now()
	if task.ExpiresAt != nil && task.ExpiresAt.Before(now) {
		if err := w.expireTask(ctx, task, now); err != nil {
			return err
		}
		return w.finalizeWithoutResume(ctx, task, "cancelled", "approval expired")
	}

	switch task.Status {
	case "rejected":
		return w.finalizeWithoutResume(ctx, task, "cancelled", "approval rejected")
	case "expired":
		return w.finalizeWithoutResume(ctx, task, "cancelled", "approval expired")
	case "approved":
		if task.LockExpiresAt == nil || task.LockExpiresAt.Before(now) {
			updated, err := w.logic.approvalTaskDAO().AcquireOrStealLease(ctx, task.ApprovalID, now.Add(w.leaseWindow))
			if err != nil {
				return fmt.Errorf("acquire approval lease: %w", err)
			}
			if !updated {
				return fmt.Errorf("approval %q is locked by another worker", task.ApprovalID)
			}
			task, err = w.logic.approvalTaskDAO().GetByApprovalID(ctx, task.ApprovalID)
			if err != nil {
				return fmt.Errorf("reload approval task: %w", err)
			}
		}
		return w.resumeApprovedTask(ctx, task)
	default:
		return fmt.Errorf("approval %q is not resumable from status %q", task.ApprovalID, task.Status)
	}
}

func (w *ApprovalWorker) claimApprovalDecidedOutboxEvent(ctx context.Context) (*model.AIApprovalOutboxEvent, error) {
	if w == nil || w.logic == nil || w.logic.svcCtx == nil || w.logic.svcCtx.DB == nil {
		return nil, nil
	}

	db := w.logic.svcCtx.DB
	var claimed *model.AIApprovalOutboxEvent

	for {
		var (
			candidate    model.AIApprovalOutboxEvent
			hadCandidate bool
		)

		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			now := w.now()
			staleBefore := now.Add(-approvalWorkerOutboxProcessingLease)
			decisionTypes := approvalDecidedEventTypes()
			query := tx.Where(
				"event_type IN ? AND ((status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)) OR (status = ? AND updated_at <= ?))",
				decisionTypes, "pending", now, "processing", staleBefore,
			).
				Order("next_retry_at ASC").
				Order("created_at ASC").
				Order("id ASC")
			if err := query.First(&candidate).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			hadCandidate = true

			result := tx.Model(&model.AIApprovalOutboxEvent{}).
				Where(
					"id = ? AND event_type IN ? AND ((status = ?) OR (status = ? AND updated_at <= ?))",
					candidate.ID, decisionTypes, "pending", "processing", staleBefore,
				).
				Updates(map[string]any{
					"status":     "processing",
					"updated_at": now,
				})
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
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
}

func (w *ApprovalWorker) resumeApprovedTask(ctx context.Context, task *model.AIApprovalTask) error {
	shell, seqCounter, err := w.logic.loadApprovalShell(ctx, task)
	if err != nil {
		return err
	}
	if err := w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
		Status: "resuming",
	}); err != nil {
		return fmt.Errorf("set run resuming: %w", err)
	}
	if writeModel := w.logic.approvalWriteModel(); writeModel != nil {
		if err := writeModel.EmitRunResuming(ctx, task.ApprovalID); err != nil {
			return fmt.Errorf("emit run resuming event: %w", err)
		}
	}

	params := buildApprovalResumeParams(task)
	iter, err := w.resume(ctx, task, params)
	if err != nil {
		return w.persistRetryableResumeFailure(ctx, task, shell, &seqCounter, fmt.Errorf("resume approval: %w", err))
	}
	if writeModel := w.logic.approvalWriteModel(); writeModel != nil {
		if err := writeModel.EmitRunResumed(ctx, task.ApprovalID); err != nil {
			return fmt.Errorf("emit run resumed event: %w", err)
		}
	}

	var (
		projector = airuntime.NewStreamProjector()
	)

	emit := func(string, any) {}
	result, err := processAgentIterator(ctx, iteratorProcessInput{
		Iterator:  iter,
		Projector: projector,
		Emit:      emit,
		ConsumeProjected: func(_ iteratorConsumeKind, events []airuntime.PublicStreamEvent) error {
			_, consumeErr := w.logic.consumeProjectedEvents(ctx, shell.Run.ID, shell.SessionID, &seqCounter, events, emit, nil)
			return consumeErr
		},
		HandleRunUpdate: func(update projectedRunUpdate) {
			if update.AssistantType != "" || update.IntentType != "" {
				_ = w.logic.RunDAO.UpdateRunStatus(ctx, shell.Run.ID, aidao.AIRunStatusUpdate{
					IntentType:    update.IntentType,
					AssistantType: update.AssistantType,
				})
			}
		},
	})
	if err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return err
	}
	if result.FatalErr != nil {
		cause := result.FatalErr
		if strings.HasPrefix(cause.Error(), "iterator event:") {
			cause = fmt.Errorf("resume %w", cause)
		} else {
			cause = fmt.Errorf("resume stream recv: %w", cause)
		}
		return w.persistFatalResumeFailure(ctx, task, shell, &seqCounter, cause)
	}
	if result.Interrupted {
		runStatus := aidao.AIRunStatusUpdate{
			Status:             "waiting_approval",
			AssistantMessageID: shell.AssistantMessage.ID,
		}
		if err := w.logic.finalizeRunCritical(ctx, shell, runStatus, result.SummaryText); err != nil {
			_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
				Status:       "resume_failed_retryable",
				ErrorMessage: err.Error(),
			})
			return fmt.Errorf("persist waiting approval state after resume interrupt: %w", err)
		}
		if err := w.logic.persistRunEnhancementsBestEffort(ctx, shell.Run.ID, shell.SessionID, runStatus.Status, result.SummaryText); err != nil {
			_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
				Status:       "resume_failed_retryable",
				ErrorMessage: err.Error(),
			})
			return fmt.Errorf("persist waiting approval convergence after resume interrupt: %w", err)
		}
		return nil
	}

	done := projector.Finish(shell.Run.ID)
	if payload, ok := done.Data.(map[string]any); ok {
		ensureDoneSummary(payload, result.SummaryText, result.HasToolErrors)
		done.Data = payload
	}
	if err := w.appendRunStateEvent(ctx, shell, &seqCounter, runStatePayload(shell.Run.ID, runFinalStatus(result.HasToolErrors), result.SummaryText)); err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return fmt.Errorf("append resume terminal run_state event: %w", err)
	}
	if err := w.logic.appendRunEvent(ctx, shell.Run.ID, shell.SessionID, &seqCounter, done.Event, done.Data); err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return fmt.Errorf("append resume done event: %w", err)
	}

	runStatus := aidao.AIRunStatusUpdate{
		Status:             runFinalStatus(result.HasToolErrors),
		AssistantMessageID: shell.AssistantMessage.ID,
	}
	if err := w.logic.finalizeRunCritical(ctx, shell, runStatus, ""); err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return fmt.Errorf("finalize resumed run: %w", err)
	}
	if err := w.logic.persistRunEnhancementsBestEffort(ctx, shell.Run.ID, shell.SessionID, runStatus.Status, result.SummaryText); err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return fmt.Errorf("persist resumed run convergence: %w", err)
	}
	if writeModel := w.logic.approvalWriteModel(); writeModel != nil {
		if err := writeModel.EmitRunCompleted(ctx, task.ApprovalID, runStatus.Status); err != nil {
			return fmt.Errorf("emit run completed event: %w", err)
		}
	}
	return nil
}

func (w *ApprovalWorker) finalizeWithoutResume(ctx context.Context, task *model.AIApprovalTask, runStatus, errorMessage string) error {
	shell, seqCounter, err := w.logic.loadApprovalShell(ctx, task)
	if err != nil {
		return err
	}
	update := aidao.AIRunStatusUpdate{
		Status:             runStatus,
		AssistantMessageID: shell.AssistantMessage.ID,
		ErrorMessage:       errorMessage,
	}
	if err := w.logic.finalizeRunCritical(ctx, shell, update, ""); err != nil {
		return fmt.Errorf("finalize non-resumable approval: %w", err)
	}
	if w.logic.RunEventDAO != nil {
		events, err := w.logic.RunEventDAO.ListByRun(ctx, shell.Run.ID)
		if err != nil {
			return fmt.Errorf("load existing run events: %w", err)
		}
		if len(events) == 0 {
			return nil
		}
	}
	if err := w.logic.persistRunEnhancementsBestEffort(ctx, shell.Run.ID, shell.SessionID, runStatus, ""); err != nil {
		return fmt.Errorf("persist non-resumable approval convergence: %w", err)
	}
	if err := w.appendRunStateEvent(ctx, shell, &seqCounter, runStatePayload(shell.Run.ID, runStatus, "")); err != nil {
		return fmt.Errorf("append non-resumable run_state event: %w", err)
	}
	if writeModel := w.logic.approvalWriteModel(); writeModel != nil {
		if err := writeModel.EmitRunCompleted(ctx, task.ApprovalID, runStatus); err != nil {
			return fmt.Errorf("emit non-resumable run completed event: %w", err)
		}
	}
	return nil
}

func (w *ApprovalWorker) expireTask(ctx context.Context, task *model.AIApprovalTask, now time.Time) error {
	return w.logic.svcCtx.DB.WithContext(ctx).
		Model(&model.AIApprovalTask{}).
		Where("approval_id = ?", task.ApprovalID).
		Updates(map[string]any{
			"status":     "expired",
			"updated_at": now,
		}).Error
}

func (w *ApprovalWorker) ensureApprovalOwnership(ctx context.Context, task *model.AIApprovalTask) error {
	if task == nil || w.logic.ChatDAO == nil || strings.TrimSpace(task.SessionID) == "" || task.UserID == 0 {
		return nil
	}
	session, err := w.logic.ChatDAO.GetSession(ctx, task.SessionID, task.UserID, "")
	if err != nil {
		return fmt.Errorf("verify approval session ownership: %w", err)
	}
	if session == nil {
		return fmt.Errorf("approval session ownership check failed")
	}
	return nil
}

func (w *ApprovalWorker) convergedRun(ctx context.Context, task *model.AIApprovalTask) (*model.AIRun, bool) {
	if task == nil || w.logic.RunDAO == nil {
		return nil, false
	}
	run, err := w.logic.RunDAO.GetRun(ctx, task.RunID)
	if err != nil || run == nil {
		return nil, false
	}
	switch run.Status {
	case "completed", "completed_with_tool_errors", "cancelled", "failed", "failed_runtime":
		return run, true
	default:
		return run, false
	}
}

func (w *ApprovalWorker) emitConvergedLifecycleEvent(ctx context.Context, task *model.AIApprovalTask, run *model.AIRun) error {
	if task == nil || run == nil {
		return nil
	}
	writeModel := w.logic.approvalWriteModel()
	if writeModel == nil {
		return nil
	}

	switch run.Status {
	case "completed", "completed_with_tool_errors", "cancelled":
		return writeModel.EmitRunCompleted(ctx, task.ApprovalID, run.Status)
	case "failed", "failed_runtime":
		return writeModel.EmitRunResumeFailed(ctx, task.ApprovalID, false, errors.New(run.ErrorMessage))
	default:
		return nil
	}
}

func (w *ApprovalWorker) persistRetryableResumeFailure(ctx context.Context, task *model.AIApprovalTask, shell chatShell, seqCounter *int, cause error) error {
	if err := w.appendRunStateEvent(ctx, shell, seqCounter, runStatePayload(shell.Run.ID, "resume_failed_retryable", "")); err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: cause.Error(),
		})
		return fmt.Errorf("%w; append retryable run_state event: %v", cause, err)
	}
	if writeModel := w.logic.approvalWriteModel(); writeModel != nil {
		_ = writeModel.EmitRunResumeFailed(ctx, task.ApprovalID, true, cause)
	}
	_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
		Status:       "resume_failed_retryable",
		ErrorMessage: cause.Error(),
	})
	return cause
}

func (w *ApprovalWorker) persistFatalResumeFailure(ctx context.Context, task *model.AIApprovalTask, shell chatShell, seqCounter *int, cause error) error {
	if err := w.appendRunStateEvent(ctx, shell, seqCounter, runStatePayload(shell.Run.ID, "failed", "")); err != nil {
		return fmt.Errorf("%w; append failed run_state event: %v", cause, err)
	}
	errorEvent := airuntime.NewErrorEvent(shell.Run.ID, cause)
	if err := w.logic.appendRunEvent(ctx, shell.Run.ID, shell.SessionID, seqCounter, errorEvent.Event, errorEvent.Data); err != nil {
		return fmt.Errorf("%w; append failed error event: %v", cause, err)
	}
	if err := w.logic.finalizeRunCritical(ctx, shell, aidao.AIRunStatusUpdate{
		Status:             "failed_runtime",
		AssistantMessageID: shell.AssistantMessage.ID,
		ErrorMessage:       cause.Error(),
	}, ""); err != nil {
		return fmt.Errorf("%w; finalize failed run: %v", cause, err)
	}
	if err := w.logic.persistRunEnhancementsBestEffort(ctx, shell.Run.ID, shell.SessionID, "failed_runtime", ""); err != nil {
		return fmt.Errorf("%w; persist failed run convergence: %v", cause, err)
	}
	if writeModel := w.logic.approvalWriteModel(); writeModel != nil {
		if err := writeModel.EmitRunResumeFailed(ctx, task.ApprovalID, false, cause); err != nil {
			return fmt.Errorf("%w; emit failed resume event: %v", cause, err)
		}
	}
	return nil
}

func (w *ApprovalWorker) appendRunStateEvent(ctx context.Context, shell chatShell, seqCounter *int, payload map[string]any) error {
	if payload == nil {
		return nil
	}
	return w.logic.appendRunEvent(ctx, shell.Run.ID, shell.SessionID, seqCounter, "run_state", payload)
}

func runStatePayload(runID, status, summary string) map[string]any {
	payload := map[string]any{
		"run_id": runID,
		"status": status,
	}
	if strings.TrimSpace(summary) != "" {
		payload["summary"] = summary
	}
	return payload
}

func runFinalStatus(hasToolErrors bool) string {
	if hasToolErrors {
		return "completed_with_tool_errors"
	}
	return "completed"
}

func (w *ApprovalWorker) reconcileApprovalTaskToolCallID(ctx context.Context, task *model.AIApprovalTask, outboxEvent *model.AIApprovalOutboxEvent) (*model.AIApprovalTask, error) {
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
		Model(&model.AIApprovalTask{}).
		Where("approval_id = ?", task.ApprovalID).
		Update("tool_call_id", candidate).Error; err != nil {
		return nil, fmt.Errorf("backfill approval task tool_call_id: %w", err)
	}
	return task, nil
}

func (w *ApprovalWorker) findApprovalResumeTargetFromRunEvents(ctx context.Context, task *model.AIApprovalTask) string {
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

func (w *ApprovalWorker) defaultResume(ctx context.Context, task *model.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
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

func buildApprovalResumeParams(task *model.AIApprovalTask) *adk.ResumeParams {
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

func (l *Logic) approvalTaskDAO() *aidao.AIApprovalTaskDAO {
	if l == nil {
		return nil
	}
	if l.ApprovalDAO != nil {
		return l.ApprovalDAO
	}
	if l.svcCtx == nil || l.svcCtx.DB == nil {
		return nil
	}
	return aidao.NewAIApprovalTaskDAO(l.svcCtx.DB)
}

func (l *Logic) loadApprovalShell(ctx context.Context, task *model.AIApprovalTask) (chatShell, int, error) {
	shell := chatShell{}
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
