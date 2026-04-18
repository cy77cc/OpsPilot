package chat

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	aicheckpoint "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/checkpoint"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/stream"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

// ResumeApprovedTask resumes a run from an approval checkpoint and persists
// the resumed event stream through the same projection/finalization pipeline
// used by the interactive chat path.
func ResumeApprovedTask(ctx context.Context, l *Logic, task *ai.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
	if l == nil || l.RunDAO == nil || l.RunEventDAO == nil || l.ChatDAO == nil || l.AIRouter == nil || l.CheckpointStore == nil {
		return nil, fmt.Errorf("chat resume logic not initialized")
	}
	if task == nil {
		return nil, fmt.Errorf("approval task is required")
	}
	if strings.TrimSpace(task.CheckpointID) == "" {
		return nil, fmt.Errorf("checkpoint id is required")
	}
	if params == nil || len(params.Targets) == 0 {
		return nil, fmt.Errorf("resume params are required")
	}

	shell, err := loadResumeShell(ctx, l, task)
	if err != nil {
		return nil, err
	}
	seq, err := currentRunSequence(ctx, l, shell.Run.ID)
	if err != nil {
		return nil, err
	}

	ctx = l.runtimeContext(ctx)
	ctx, runtime := runtimectx.Ensure(ctx)
	if rid := strings.TrimSpace(shell.Run.ClientRequestID); rid != "" {
		runtime.RequestID = rid
	}
	ctx = runtimectx.WithAIMetadata(ctx, runtimectx.AIMetadata{
		SessionID:    shell.SessionID,
		RunID:        shell.Run.ID,
		UserID:       task.UserID,
		Scene:        shell.Scene,
		CheckpointID: task.CheckpointID,
	})
	ctx = aicheckpoint.ContextWithMetadata(ctx, aicheckpoint.Metadata{
		SessionID:    shell.SessionID,
		RunID:        shell.Run.ID,
		UserID:       task.UserID,
		Scene:        shell.Scene,
		CheckpointID: task.CheckpointID,
	})

	if err := persistResumeRunState(ctx, l, shell, &seq, "resuming"); err != nil {
		return nil, err
	}
	if err := l.RunDAO.UpdateRunStatus(ctx, shell.Run.ID, aidao.AIRunStatusUpdate{
		Status:             "resuming",
		AssistantMessageID: shell.AssistantMessage.ID,
	}); err != nil {
		return nil, err
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           l.AIRouter,
		EnableStreaming: true,
		CheckPointStore: l.CheckpointStore,
	})
	iterator, err := runner.ResumeWithParams(ctx, task.CheckpointID, params)
	if err != nil {
		return nil, markResumeRetryableFailure(ctx, l, shell, &seq, err, "", "")
	}

	projector := airuntime.NewStreamProjector()
	delegationState := &delegationStreamState{}
	result, processErr := stream.ProcessAgentIterator(ctx, stream.IteratorProcessInput{
		Iterator:  iterator,
		Projector: projector,
		Emit:      func(string, any) {},
		ConsumeProjected: func(_ stream.IteratorConsumeKind, events []airuntime.PublicStreamEvent) error {
			delegationState.observe(events)
			_, consumeErr := ConsumeProjectedEvents(ctx, l, shell.Run.ID, shell.SessionID, &seq, events, func(string, any) {})
			return consumeErr
		},
		HandleRunUpdate: func(update stream.RunUpdate) {
			if update.AssistantType != "" || update.IntentType != "" {
				_ = l.RunDAO.UpdateRunStatus(ctx, shell.Run.ID, aidao.AIRunStatusUpdate{
					IntentType:    update.IntentType,
					AssistantType: update.AssistantType,
				})
			}
		},
	})
	if processErr != nil {
		return nil, markResumeRetryableFailure(ctx, l, shell, &seq, processErr, "", "")
	}
	if result.FatalErr != nil {
		snapshot := result.AssistantSnapshot
		if !stream.ShouldRetainPartialStreamSnapshot(result.FatalErr) {
			snapshot = ""
		}
		return nil, markResumeRetryableFailure(ctx, l, shell, &seq, result.FatalErr, result.SummaryText, snapshot)
	}
	if err := emitDelegationWindows(ctx, l, shell, delegationState, &seq, func(string, any) {}); err != nil {
		return nil, err
	}
	if persisted := projector.GetPersistedState(); persisted != nil && !persisted.CanFinalizeDone() {
		runStatus := aidao.AIRunStatusUpdate{Status: "waiting_approval", AssistantMessageID: shell.AssistantMessage.ID}
		if err := FinalizeRunCritical(ctx, l, shell, runStatus, result.SummaryText); err != nil {
			return nil, fmt.Errorf("persist waiting approval state: %w", err)
		}
		_ = PersistRunEnhancementsBestEffort(ctx, l, shell.Run.ID, shell.SessionID, runStatus.Status, result.SummaryText)
		return nil, nil
	}

	done := projector.Finish(shell.Run.ID)
	if payload, ok := done.Data.(map[string]any); ok {
		stream.EnsureDoneSummary(payload, result.SummaryText, result.HasToolErrors)
		done.Data = payload
	}
	eid, err := AppendRunEventWithID(ctx, l, shell.Run.ID, shell.SessionID, &seq, done.Event, done.Data)
	if err != nil {
		return nil, fmt.Errorf("append done event: %w", err)
	}
	if err := persistTerminalProjectionEvent(ctx, l, shell.Run.ID, shell.SessionID, eid, done); err != nil {
		return nil, fmt.Errorf("persist terminal projection: %w", err)
	}

	runStatus := aidao.AIRunStatusUpdate{Status: "completed", AssistantMessageID: shell.AssistantMessage.ID}
	if result.HasToolErrors {
		runStatus.Status = "completed_with_tool_errors"
	}
	finalAssistantContent := strings.TrimSpace(result.AssistantSnapshot)
	if finalAssistantContent == "" {
		finalAssistantContent = strings.TrimSpace(result.SummaryText)
	}
	if err := FinalizeRunCritical(ctx, l, shell, runStatus, finalAssistantContent); err != nil {
		return nil, fmt.Errorf("finalize resumed run: %w", err)
	}
	_ = PersistRunEnhancementsBestEffort(ctx, l, shell.Run.ID, shell.SessionID, runStatus.Status, result.SummaryText)
	return nil, nil
}

func loadResumeShell(ctx context.Context, l *Logic, task *ai.AIApprovalTask) (ChatShell, error) {
	shell := ChatShell{}
	run, err := l.RunDAO.GetRun(ctx, task.RunID)
	if err != nil {
		return shell, fmt.Errorf("load run: %w", err)
	}
	if run == nil {
		return shell, fmt.Errorf("run %q not found", task.RunID)
	}
	userMessage, err := l.ChatDAO.GetMessage(ctx, run.UserMessageID)
	if err != nil {
		return shell, fmt.Errorf("load user message: %w", err)
	}
	assistantMessage, err := l.ChatDAO.GetMessage(ctx, run.AssistantMessageID)
	if err != nil {
		return shell, fmt.Errorf("load assistant message: %w", err)
	}
	if userMessage == nil || assistantMessage == nil {
		return shell, fmt.Errorf("resume shell messages not found")
	}
	session, err := l.ChatDAO.GetSession(ctx, run.SessionID, task.UserID, "")
	if err != nil {
		return shell, fmt.Errorf("load session: %w", err)
	}
	scene := ResolveChatScene("", session)
	return ChatShell{
		SessionID:        run.SessionID,
		Scene:            scene,
		Run:              run,
		UserMessage:      userMessage,
		AssistantMessage: assistantMessage,
		Reused:           true,
	}, nil
}

func currentRunSequence(ctx context.Context, l *Logic, runID string) (int, error) {
	if l == nil || l.RunEventDAO == nil {
		return 0, nil
	}
	events, err := l.RunEventDAO.ListByRun(ctx, runID)
	if err != nil {
		return 0, err
	}
	if len(events) == 0 {
		return 0, nil
	}
	return events[len(events)-1].Seq, nil
}

func persistResumeRunState(ctx context.Context, l *Logic, shell ChatShell, seq *int, status string) error {
	event := airuntime.PublicStreamEvent{
		Event: "run_state",
		Data: map[string]any{
			"run_id": shell.Run.ID,
			"status": status,
			"agent":  "executor",
		},
	}
	eid, err := AppendRunEventWithID(ctx, l, shell.Run.ID, shell.SessionID, seq, event.Event, event.Data)
	if err != nil {
		return err
	}
	if strings.TrimSpace(eid) == "" {
		return nil
	}
	return persistTerminalProjectionEvent(ctx, l, shell.Run.ID, shell.SessionID, eid, event)
}

func markResumeRetryableFailure(ctx context.Context, l *Logic, shell ChatShell, seq *int, resumeErr error, summaryBody, assistantBody string) error {
	if err := persistResumeRunState(ctx, l, shell, seq, "resume_failed_retryable"); err != nil {
		return err
	}
	publicErr := stream.SanitizeUserFacingError(resumeErr)
	snapshot := stream.BuildAssistantFailureSnapshot(summaryBody, assistantBody, publicErr)
	runUpdate := aidao.AIRunStatusUpdate{
		Status:             "resume_failed_retryable",
		AssistantMessageID: shell.AssistantMessage.ID,
		ErrorMessage:       resumeErr.Error(),
	}
	if err := FinalizeRunCritical(ctx, l, shell, runUpdate, snapshot); err != nil {
		return err
	}
	if err := PersistRunEnhancementsBestEffort(ctx, l, shell.Run.ID, shell.SessionID, runUpdate.Status, snapshot); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
