package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

const (
	approvalWorkerDefaultLeaseWindow  = 2 * time.Minute
	approvalWorkerDefaultRetryDelay   = 5 * time.Second
	approvalWorkerDefaultPollInterval = 2 * time.Second
)

type approvalResumeFunc func(context.Context, *model.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error)

type ApprovalWorkerOption func(*ApprovalWorker)

type ApprovalWorker struct {
	logic       *Logic
	leaseWindow time.Duration
	retryDelay  time.Duration
	now         func() time.Time
	resume      approvalResumeFunc
}

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
	event, err := outboxDAO.ClaimPending(ctx)
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
	if strings.TrimSpace(event.EventType) != "approval_decided" {
		return nil
	}

	task, err := w.logic.approvalTaskDAO().GetByApprovalID(ctx, event.ApprovalID)
	if err != nil {
		return fmt.Errorf("load approval task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("approval task %q not found", event.ApprovalID)
	}
	if err := w.ensureApprovalOwnership(ctx, task); err != nil {
		return err
	}
	if w.runAlreadyConverged(ctx, task) {
		return nil
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

	params := buildApprovalResumeParams(task)
	iter, err := w.resume(ctx, task, params)
	if err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return fmt.Errorf("resume approval: %w", err)
	}

	var (
		summaryContent strings.Builder
		projector      = airuntime.NewStreamProjector()
		hasToolErrors  bool
		toolFailures   = newToolFailureTracker()
		circuitBroken  bool
	)

	emit := func(string, any) {}
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			if recoverable, ok := recoverableToolErrorFromEvent(event); ok {
				hasToolErrors = true
				if _, count, tripped := toolFailures.recordFailure(recoverable.Info); tripped && count > 0 {
					circuitBroken = true
				}
				projected := projector.Consume(recoverable.Event)
				toolFailures.recordProjectedEvents(projected)
				update, consumeErr := w.logic.consumeProjectedEvents(ctx, shell.Run.ID, shell.SessionID, &seqCounter, projected, emit, &summaryContent)
				if consumeErr != nil {
					_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
						Status:       "resume_failed_retryable",
						ErrorMessage: consumeErr.Error(),
					})
					return fmt.Errorf("persist recoverable tool error: %w", consumeErr)
				}
				if update.AssistantType != "" || update.IntentType != "" {
					_ = w.logic.RunDAO.UpdateRunStatus(ctx, shell.Run.ID, aidao.AIRunStatusUpdate{
						IntentType:    update.IntentType,
						AssistantType: update.AssistantType,
					})
				}
				if circuitBroken {
					break
				}
				continue
			}
			if isBusinessToolResultEvent(event) {
				hasToolErrors = true
				// Allow the normal projection path to persist the tool_result error.
			} else {
				_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
					Status:       "resume_failed_retryable",
					ErrorMessage: event.Err.Error(),
				})
				return fmt.Errorf("resume iterator event: %w", event.Err)
			}
		}

		if event.Output != nil && event.Output.MessageOutput != nil && event.Output.MessageOutput.IsStreaming && event.Output.MessageOutput.MessageStream != nil {
			for {
				msg, recvErr := event.Output.MessageOutput.MessageStream.Recv()
				if recvErr == io.EOF {
					break
				}
				if recvErr != nil {
					if recoverable, ok := recoverableToolErrorFromErr(recvErr, event.AgentName); ok {
						hasToolErrors = true
						if _, count, tripped := toolFailures.recordFailure(recoverable.Info); tripped && count > 0 {
							circuitBroken = true
						}
						projected := projector.Consume(recoverable.Event)
						toolFailures.recordProjectedEvents(projected)
						update, consumeErr := w.logic.consumeProjectedEvents(ctx, shell.Run.ID, shell.SessionID, &seqCounter, projected, emit, &summaryContent)
						if consumeErr != nil {
							_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
								Status:       "resume_failed_retryable",
								ErrorMessage: consumeErr.Error(),
							})
							return fmt.Errorf("persist recoverable tool error: %w", consumeErr)
						}
						if update.AssistantType != "" || update.IntentType != "" {
							_ = w.logic.RunDAO.UpdateRunStatus(ctx, shell.Run.ID, aidao.AIRunStatusUpdate{
								IntentType:    update.IntentType,
								AssistantType: update.AssistantType,
							})
						}
						if circuitBroken {
							break
						}
						continue
					}
					_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
						Status:       "resume_failed_retryable",
						ErrorMessage: recvErr.Error(),
					})
					return fmt.Errorf("resume stream recv: %w", recvErr)
				}
				if msg == nil {
					continue
				}

				chunkEvent := adk.EventFromMessage(msg, nil, msg.Role, msg.ToolName)
				chunkEvent.AgentName = event.AgentName
				projected := projector.Consume(chunkEvent)
				toolFailures.recordProjectedEvents(projected)
				update, consumeErr := w.logic.consumeProjectedEvents(ctx, shell.Run.ID, shell.SessionID, &seqCounter, projected, emit, &summaryContent)
				if consumeErr != nil {
					_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
						Status:       "resume_failed_retryable",
						ErrorMessage: consumeErr.Error(),
					})
					return fmt.Errorf("persist resume stream chunk: %w", consumeErr)
				}
				if update.AssistantType != "" || update.IntentType != "" {
					_ = w.logic.RunDAO.UpdateRunStatus(ctx, shell.Run.ID, aidao.AIRunStatusUpdate{
						IntentType:    update.IntentType,
						AssistantType: update.AssistantType,
					})
				}
			}
			if circuitBroken {
				break
			}
			continue
		}

		projected := projector.Consume(event)
		toolFailures.recordProjectedEvents(projected)
		update, consumeErr := w.logic.consumeProjectedEvents(ctx, shell.Run.ID, shell.SessionID, &seqCounter, projected, emit, &summaryContent)
		if consumeErr != nil {
			_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
				Status:       "resume_failed_retryable",
				ErrorMessage: consumeErr.Error(),
			})
			return fmt.Errorf("persist resume event: %w", consumeErr)
		}
		if update.AssistantType != "" || update.IntentType != "" {
			_ = w.logic.RunDAO.UpdateRunStatus(ctx, shell.Run.ID, aidao.AIRunStatusUpdate{
				IntentType:    update.IntentType,
				AssistantType: update.AssistantType,
			})
		}
	}

	flushEvents := projector.FlushBuffer()
	toolFailures.recordProjectedEvents(flushEvents)
	if err := w.logic.flushProjectedEvents(ctx, shell.Run.ID, shell.SessionID, &seqCounter, flushEvents, emit, &summaryContent); err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return fmt.Errorf("flush resume projected events: %w", err)
	}

	done := projector.Finish(shell.Run.ID)
	if payload, ok := done.Data.(map[string]any); ok {
		ensureDoneSummary(payload, summaryContent.String(), hasToolErrors)
		done.Data = payload
	}
	if err := w.logic.appendRunEvent(ctx, shell.Run.ID, shell.SessionID, &seqCounter, done.Event, done.Data); err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return fmt.Errorf("append resume done event: %w", err)
	}

	runStatus := aidao.AIRunStatusUpdate{
		Status:             "completed",
		AssistantMessageID: shell.AssistantMessage.ID,
	}
	if hasToolErrors {
		runStatus.Status = "completed_with_tool_errors"
	}
	if err := w.logic.finalizeRunCritical(ctx, shell, runStatus, ""); err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return fmt.Errorf("finalize resumed run: %w", err)
	}
	if err := w.logic.persistRunEnhancementsBestEffort(ctx, shell.Run.ID, shell.SessionID, runStatus.Status, summaryContent.String()); err != nil {
		_ = w.logic.RunDAO.UpdateRunStatus(ctx, task.RunID, aidao.AIRunStatusUpdate{
			Status:       "resume_failed_retryable",
			ErrorMessage: err.Error(),
		})
		return fmt.Errorf("persist resumed run convergence: %w", err)
	}
	return nil
}

func (w *ApprovalWorker) finalizeWithoutResume(ctx context.Context, task *model.AIApprovalTask, runStatus, errorMessage string) error {
	shell, _, err := w.logic.loadApprovalShell(ctx, task)
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

func (w *ApprovalWorker) runAlreadyConverged(ctx context.Context, task *model.AIApprovalTask) bool {
	if task == nil || w.logic.RunDAO == nil {
		return false
	}
	run, err := w.logic.RunDAO.GetRun(ctx, task.RunID)
	if err != nil || run == nil {
		return false
	}
	switch run.Status {
	case "completed", "completed_with_tool_errors", "cancelled":
		return true
	default:
		return false
	}
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
	return runner.ResumeWithParams(ctx, task.CheckpointID, params, adk.WithCheckPointID(task.CheckpointID))
}

func buildApprovalResumeParams(task *model.AIApprovalTask) *adk.ResumeParams {
	payload := map[string]any{
		"approved":          task != nil && task.Status == "approved",
		"disapprove_reason": "",
		"comment":           "",
		"approved_by":       uint64(0),
		"approved_at":       "",
	}
	if task != nil {
		payload["disapprove_reason"] = task.DisapproveReason
		payload["comment"] = task.Comment
		payload["approved_by"] = task.ApprovedBy
		if task.DecidedAt != nil {
			payload["approved_at"] = task.DecidedAt.UTC().Format(time.RFC3339)
		}
	}

	targets := map[string]any{}
	if task != nil && strings.TrimSpace(task.ToolCallID) != "" {
		targets[task.ToolCallID] = payload
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

func buildApprovalDecisionOutboxPayload(task *model.AIApprovalTask) (string, error) {
	if task == nil {
		return "", fmt.Errorf("approval task is required")
	}

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
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
