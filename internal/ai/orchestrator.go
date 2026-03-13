package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/google/uuid"

	"github.com/cy77cc/OpsPilot/internal/ai/agents"
	aiobs "github.com/cy77cc/OpsPilot/internal/ai/observability"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aitools "github.com/cy77cc/OpsPilot/internal/ai/tools"
	approvaltools "github.com/cy77cc/OpsPilot/internal/ai/tools/approval"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

type Orchestrator struct {
	runner           *adk.Runner
	checkpoints      *airuntime.CheckpointStore
	executions       *airuntime.ExecutionStore
	contextProcessor *airuntime.ContextProcessor
	sceneResolver    *airuntime.SceneConfigResolver
	converter        *airuntime.SSEConverter
	approvals        *airuntime.ApprovalDecisionMaker
	summaries        *approvaltools.SummaryRenderer
}

func NewOrchestrator(_ any, executionStore *airuntime.ExecutionStore, deps common.PlatformDeps) *Orchestrator {
	ctx := context.Background()
	sceneResolver := airuntime.NewSceneConfigResolver(nil)
	contextProcessor := airuntime.NewContextProcessor(sceneResolver)
	checkpointStore := airuntime.NewCheckpointStore(nil, "")
	registry := aitools.NewRegistry(deps)
	approvals := airuntime.NewApprovalDecisionMaker(airuntime.ApprovalDecisionMakerOptions{
		ResolveScene: sceneResolver.Resolve,
		LookupTool: func(name string) (airuntime.ApprovalToolSpec, bool) {
			spec, ok := registry.Get(name)
			if !ok {
				return airuntime.ApprovalToolSpec{}, false
			}
			return airuntime.ApprovalToolSpec{
				Name:        spec.Name,
				DisplayName: spec.DisplayName,
				Description: spec.Description,
				Mode:        string(spec.Mode),
				Risk:        string(spec.Risk),
				Category:    spec.Category,
			}, true
		},
	})
	summaries := approvaltools.NewSummaryRenderer()
	if executionStore == nil {
		executionStore = airuntime.NewExecutionStore(nil, "")
	}

	agent, err := agents.NewAgent(ctx, agents.Deps{
		PlatformDeps:     deps,
		ContextProcessor: contextProcessor,
	})
	if err != nil {
		return &Orchestrator{
			executions:       executionStore,
			checkpoints:      checkpointStore,
			contextProcessor: contextProcessor,
			sceneResolver:    sceneResolver,
			converter:        airuntime.NewSSEConverter(),
			approvals:        approvals,
			summaries:        summaries,
		}
	}

	return &Orchestrator{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{
			Agent:           agent,
			CheckPointStore: checkpointStore,
			EnableStreaming: true,
		}),
		checkpoints:      checkpointStore,
		executions:       executionStore,
		contextProcessor: contextProcessor,
		sceneResolver:    sceneResolver,
		converter:        airuntime.NewSSEConverter(),
		approvals:        approvals,
		summaries:        summaries,
	}
}

func (o *Orchestrator) Run(ctx context.Context, req airuntime.RunRequest, emit airuntime.StreamEmitter) error {
	startedAt := time.Now().UTC()
	if o == nil || o.runner == nil {
		return fmt.Errorf("orchestrator runner is nil")
	}
	if strings.TrimSpace(req.Message) == "" {
		return fmt.Errorf("message is empty")
	}
	if emit == nil {
		emit = func(airuntime.StreamEvent) bool { return true }
	}

	sessionID := firstNonEmpty(req.SessionID, uuid.NewString())
	planID := uuid.NewString()
	turnID := uuid.NewString()
	checkpointID := uuid.NewString()
	scene := o.sceneResolver.Resolve(req.RuntimeContext.Scene)
	adkValues := map[string]any{
		airuntime.SessionKeyRuntimeContext: req.RuntimeContext,
		airuntime.SessionKeyResolvedScene:  scene,
		airuntime.SessionKeySessionID:      sessionID,
		airuntime.SessionKeyPlanID:         planID,
		airuntime.SessionKeyTurnID:         turnID,
	}

	state := airuntime.ExecutionState{
		TraceID:        uuid.NewString(),
		SessionID:      sessionID,
		PlanID:         planID,
		TurnID:         turnID,
		Message:        req.Message,
		Scene:          req.RuntimeContext.Scene,
		Status:         airuntime.ExecutionStatusRunning,
		Phase:          "plan",
		RuntimeContext: req.RuntimeContext,
		CheckpointID:   checkpointID,
		Steps:          map[string]airuntime.StepState{},
		Metadata: map[string]any{
			"token_accounting_status": "unavailable",
			"token_accounting_source": "runtime_api_unavailable",
		},
	}
	_ = o.executions.Save(ctx, state)

	emit(airuntime.StreamEvent{Type: airuntime.EventMeta, Data: map[string]any{
		"session_id": sessionID,
		"plan_id":    planID,
		"turn_id":    turnID,
	}})
	for _, evt := range o.converter.OnPlannerStart(sessionID, planID, turnID) {
		if !emit(evt) {
			return nil
		}
	}
	emit(airuntime.StreamEvent{Type: airuntime.EventTurnState, Data: map[string]any{"turn_id": turnID, "status": "running"}})

	iter := o.runner.Query(ctx, strings.TrimSpace(req.Message),
		adk.WithCheckPointID(checkpointID),
		adk.WithSessionValues(adkValues),
	)
	_, err := o.streamExecution(ctx, iter, &state, emit)
	aiobs.ObserveAgentExecution(aiobs.ExecutionRecord{
		Operation: "run",
		Scene:     req.RuntimeContext.Scene,
		Status:    statusFromExecutionState(state.Status, err),
		Duration:  time.Since(startedAt),
	})
	return err
}

func (o *Orchestrator) Resume(ctx context.Context, req airuntime.ResumeRequest) (*airuntime.ResumeResult, error) {
	return o.resume(ctx, req, nil)
}

func (o *Orchestrator) ResumeStream(ctx context.Context, req airuntime.ResumeRequest, emit airuntime.StreamEmitter) (*airuntime.ResumeResult, error) {
	return o.resume(ctx, req, emit)
}

func (o *Orchestrator) resume(ctx context.Context, req airuntime.ResumeRequest, emit airuntime.StreamEmitter) (*airuntime.ResumeResult, error) {
	startedAt := time.Now().UTC()
	if o == nil {
		return nil, fmt.Errorf("orchestrator is nil")
	}

	state, ok, err := o.loadExecution(ctx, req)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("execution state not found")
	}

	stepID := firstNonEmpty(req.StepID, state.InterruptTarget, pendingStepID(state.PendingApproval))
	if emit != nil {
		emit(airuntime.StreamEvent{Type: airuntime.EventMeta, Data: map[string]any{
			"session_id": state.SessionID,
			"plan_id":    state.PlanID,
			"turn_id":    state.TurnID,
		}})
		emit(airuntime.StreamEvent{Type: airuntime.EventTurnStarted, Data: map[string]any{
			"turn_id":    state.TurnID,
			"session_id": state.SessionID,
		}})
		emit(airuntime.StreamEvent{Type: airuntime.EventTurnState, Data: map[string]any{
			"turn_id": state.TurnID,
			"status":  "running",
		}})
	}

	if !req.Approved {
		state.Status = airuntime.ExecutionStatusRejected
		if step := state.Steps[stepID]; step.StepID != "" {
			step.Status = airuntime.StepRejected
			step.UserVisibleSummary = "审批已拒绝，当前步骤不会执行。"
			state.Steps[stepID] = step
		}
		if state.PendingApproval != nil {
			state.PendingApproval.Status = "rejected"
		}
		_ = o.executions.Save(ctx, state)
		if emit != nil {
			for _, evt := range o.converter.OnApprovalResult(stepID, false, req.Reason) {
				emit(evt)
			}
			emit(o.converter.OnDone(string(state.Status)))
		}
		aiobs.ObserveAgentExecution(aiobs.ExecutionRecord{
			Operation: "resume",
			Scene:     state.Scene,
			Status:    string(state.Status),
			Duration:  time.Since(startedAt),
		})
		return &airuntime.ResumeResult{
			Resumed:   true,
			SessionID: state.SessionID,
			PlanID:    state.PlanID,
			StepID:    stepID,
			TurnID:    state.TurnID,
			Status:    string(state.Status),
			Message:   "审批已拒绝，待审批步骤不会继续执行。",
		}, nil
	}

	checkpointID, target, found, err := o.checkpoints.Resolve(ctx, state.SessionID, state.PlanID, stepID, firstNonEmpty(req.CheckpointID, state.CheckpointID))
	if err != nil {
		return nil, err
	}
	if !found || o.runner == nil {
		state.Status = airuntime.ExecutionStatusCompleted
		if step := state.Steps[stepID]; step.StepID != "" {
			step.Status = airuntime.StepSucceeded
			state.Steps[stepID] = step
		}
		if state.PendingApproval != nil {
			state.PendingApproval.Status = "approved"
		}
		_ = o.executions.Save(ctx, state)
		if emit != nil {
			for _, evt := range o.converter.OnApprovalResult(stepID, true, req.Reason) {
				emit(evt)
			}
			for _, evt := range o.converter.OnExecuteComplete() {
				emit(evt)
			}
			emit(o.converter.OnDone(string(state.Status)))
		}
		aiobs.ObserveAgentExecution(aiobs.ExecutionRecord{
			Operation: "resume",
			Scene:     state.Scene,
			Status:    string(state.Status),
			Duration:  time.Since(startedAt),
		})
		return &airuntime.ResumeResult{
			Resumed:   true,
			SessionID: state.SessionID,
			PlanID:    state.PlanID,
			StepID:    stepID,
			TurnID:    state.TurnID,
			Status:    string(state.Status),
			Message:   "审批已通过，待审批步骤会继续执行。",
		}, nil
	}

	params := &adk.ResumeParams{}
	if strings.TrimSpace(target) != "" {
		params.Targets = map[string]any{
			target: map[string]any{
				"approved": true,
				"reason":   strings.TrimSpace(req.Reason),
			},
		}
	}
	iter, err := o.runner.ResumeWithParams(ctx, checkpointID, params)
	if err != nil {
		aiobs.ObserveAgentExecution(aiobs.ExecutionRecord{
			Operation: "resume",
			Scene:     state.Scene,
			Status:    "failed",
			Duration:  time.Since(startedAt),
		})
		return nil, err
	}
	res, streamErr := o.streamExecution(ctx, iter, &state, emit)
	aiobs.ObserveAgentExecution(aiobs.ExecutionRecord{
		Operation: "resume",
		Scene:     state.Scene,
		Status:    statusFromExecutionState(state.Status, streamErr),
		Duration:  time.Since(startedAt),
	})
	return res, streamErr
}

func (o *Orchestrator) streamExecution(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent], state *airuntime.ExecutionState, emit airuntime.StreamEmitter) (*airuntime.ResumeResult, error) {
	if iter == nil {
		return nil, fmt.Errorf("event iterator is nil")
	}
	if emit == nil {
		emit = func(airuntime.StreamEvent) bool { return true }
	}

	var lastText string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			emit(o.converter.OnError(state.Phase, event.Err))
			return nil, event.Err
		}
		msg, _, err := adk.GetMessage(event)
		if err != nil {
			return nil, fmt.Errorf("failed to parse agent message: %w", err)
		}
		if msg != nil && strings.TrimSpace(msg.Content) != "" {
			text := strings.TrimSpace(msg.Content)
			if text != lastText {
				lastText = text
				emit(o.converter.OnTextDelta(text))
			}
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			stepID := interruptStepID(event)
			pending := o.pendingApprovalFromInterrupt(state, stepID, event)
			state.Status = airuntime.ExecutionStatusWaitingApproval
			state.Phase = "waiting_approval"
			state.InterruptTarget = stepID
			state.PendingApproval = pending
			state.Steps[stepID] = airuntime.StepState{
				StepID:             stepID,
				Title:              pending.Title,
				Status:             airuntime.StepWaitingApproval,
				Mode:               pending.Mode,
				Risk:               pending.Risk,
				ToolName:           pending.ToolName,
				ToolArgs:           pending.Params,
				UserVisibleSummary: pending.Summary,
			}
			_ = o.checkpoints.BindIdentity(ctx, state.SessionID, state.PlanID, stepID, state.CheckpointID, stepID)
			_ = o.executions.Save(ctx, *state)
			for _, evt := range o.converter.OnApprovalRequired(state.PendingApproval, state.CheckpointID) {
				emit(evt)
			}
			emit(o.converter.OnDone(string(state.Status)))
			return &airuntime.ResumeResult{
				Interrupted: true,
				SessionID:   state.SessionID,
				PlanID:      state.PlanID,
				StepID:      stepID,
				TurnID:      state.TurnID,
				Status:      string(state.Status),
				Message:     "执行已中断，等待审批。",
			}, nil
		}
	}

	state.Status = airuntime.ExecutionStatusCompleted
	state.Phase = "completed"
	_ = o.executions.Save(ctx, *state)
	for _, evt := range o.converter.OnExecuteComplete() {
		emit(evt)
	}
	emit(o.converter.OnDone(string(state.Status)))
	return &airuntime.ResumeResult{
		Resumed:   true,
		SessionID: state.SessionID,
		PlanID:    state.PlanID,
		StepID:    state.InterruptTarget,
		TurnID:    state.TurnID,
		Status:    string(state.Status),
		Message:   lastNonEmpty(lastText, "执行完成。"),
	}, nil
}

func (o *Orchestrator) loadExecution(ctx context.Context, req airuntime.ResumeRequest) (airuntime.ExecutionState, bool, error) {
	if o.executions == nil {
		return airuntime.ExecutionState{}, false, nil
	}
	if strings.TrimSpace(req.SessionID) != "" && strings.TrimSpace(req.PlanID) != "" {
		return o.executions.Load(ctx, req.SessionID, req.PlanID)
	}
	if strings.TrimSpace(req.SessionID) != "" {
		return o.executions.LoadLatestBySession(ctx, req.SessionID)
	}
	return airuntime.ExecutionState{}, false, nil
}

func (o *Orchestrator) pendingApprovalFromInterrupt(state *airuntime.ExecutionState, stepID string, event *adk.AgentEvent) *airuntime.PendingApproval {
	info := interruptApprovalInfo(event)
	decision := airuntime.ApprovalDecision{
		Environment: info.Environment,
		Tool: airuntime.ApprovalToolSpec{
			Name:        info.ToolName,
			DisplayName: info.ToolDisplayName,
			Mode:        info.Mode,
			Risk:        info.RiskLevel,
		},
	}
	summary := strings.TrimSpace(info.Summary)
	if summary == "" && o.summaries != nil {
		summary = o.summaries.Render(decision, info.Params)
	}
	if summary == "" {
		summary = "执行到敏感步骤，需要确认后继续。"
	}
	return &airuntime.PendingApproval{
		ID:          uuid.NewString(),
		PlanID:      state.PlanID,
		StepID:      stepID,
		Status:      "pending",
		Title:       firstNonEmpty(info.ToolDisplayName, info.ToolName, "待确认步骤"),
		Mode:        firstNonEmpty(info.Mode, "mutating"),
		Risk:        firstNonEmpty(info.RiskLevel, "medium"),
		Summary:     summary,
		ApprovalKey: airuntime.ResumeIdentity(state.SessionID, state.PlanID, stepID),
		ToolName:    info.ToolName,
		Params:      info.Params,
		CreatedAt:   timeNowUTC(),
		ExpiresAt:   timeNowUTC().Add(24 * time.Hour),
	}
}

func interruptApprovalInfo(event *adk.AgentEvent) airuntime.ApprovalInterruptInfo {
	if event == nil || event.Action == nil || event.Action.Interrupted == nil {
		return airuntime.ApprovalInterruptInfo{}
	}
	for _, interruptCtx := range event.Action.Interrupted.InterruptContexts {
		if interruptCtx == nil {
			continue
		}
		switch info := interruptCtx.Info.(type) {
		case airuntime.ApprovalInterruptInfo:
			return info
		case *airuntime.ApprovalInterruptInfo:
			if info != nil {
				return *info
			}
		case map[string]any:
			return airuntime.ApprovalInterruptInfo{
				PlanID:          mapString(info["plan_id"]),
				StepID:          mapString(info["step_id"]),
				ToolName:        mapString(info["tool_name"]),
				ToolDisplayName: mapString(info["tool_display_name"]),
				Mode:            mapString(info["mode"]),
				RiskLevel:       firstNonEmpty(mapString(info["risk_level"]), mapString(info["risk"])),
				Summary:         mapString(info["summary"]),
				Params:          mapParams(info["params"]),
				Environment:     mapString(info["environment"]),
				Namespace:       mapString(info["namespace"]),
			}
		}
	}
	return airuntime.ApprovalInterruptInfo{}
}

func interruptStepID(event *adk.AgentEvent) string {
	if event == nil || event.Action == nil || event.Action.Interrupted == nil {
		return uuid.NewString()
	}
	for _, interruptCtx := range event.Action.Interrupted.InterruptContexts {
		if interruptCtx == nil {
			continue
		}
		if strings.TrimSpace(interruptCtx.ID) != "" {
			return strings.TrimSpace(interruptCtx.ID)
		}
	}
	return uuid.NewString()
}

func pendingStepID(pending *airuntime.PendingApproval) string {
	if pending == nil {
		return ""
	}
	return strings.TrimSpace(pending.StepID)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func lastNonEmpty(values ...string) string {
	for i := len(values) - 1; i >= 0; i-- {
		if strings.TrimSpace(values[i]) != "" {
			return strings.TrimSpace(values[i])
		}
	}
	return ""
}

func timeNowUTC() (t time.Time) {
	return time.Now().UTC()
}

func mapString(value any) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func mapParams(value any) map[string]any {
	params, _ := value.(map[string]any)
	if len(params) == 0 {
		return nil
	}
	return params
}

func statusFromExecutionState(status airuntime.ExecutionStatus, err error) string {
	if err != nil {
		return string(airuntime.ExecutionStatusFailed)
	}
	if strings.TrimSpace(string(status)) == "" {
		return string(airuntime.ExecutionStatusCompleted)
	}
	return string(status)
}
