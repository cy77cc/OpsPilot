// 本文件实现基于 eino ADK 的 AI 编排器。
//
// Orchestrator 是 AI 模块对外的唯一入口，负责：
//   - 初始化 ADK Runner，并驱动单条后端主路径执行
//   - 接收用户请求，将执行进度收口为 turn lifecycle + delta/approval 事件
//   - 处理人工审批中断：在敏感变更工具调用前暂停执行，等待外部 Resume 信号
//   - 持久化执行状态（ExecutionStore）和断点（CheckpointStore）以支持会话恢复

package ai

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"github.com/cy77cc/OpsPilot/internal/ai/agents"
	aiobs "github.com/cy77cc/OpsPilot/internal/ai/observability"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aitools "github.com/cy77cc/OpsPilot/internal/ai/tools"
	approvaltools "github.com/cy77cc/OpsPilot/internal/ai/tools/approval"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

// Orchestrator 封装 ADK Runner，提供流式执行与审批恢复能力。
type Orchestrator struct {
	runner           *adk.Runner                      // ADK plan-execute 运行器；nil 表示模型不可用
	checkpoints      *airuntime.CheckpointStore       // 保存 Agent 断点，支持审批后续跑
	executions       *airuntime.ExecutionStore        // 保存每次执行的状态快照
	contextProcessor *airuntime.ContextProcessor      // 构建各阶段 LLM 输入的上下文处理器
	sceneResolver    *airuntime.SceneConfigResolver   // 根据场景 key 解析工具白名单和审批策略
	converter        *airuntime.SSEConverter          // 将 Agent 事件转换为标准 SSE StreamEvent
	approvals        *airuntime.ApprovalDecisionMaker // 判断工具调用是否需要人工审批
	summaries        *approvaltools.SummaryRenderer   // 生成审批请求的人类可读摘要
}

// NewOrchestrator 创建并初始化 Orchestrator。
// 若 Agent 构建失败（如模型不可用），返回不含 runner 的降级实例，
// Run 调用时会立即返回错误，Resume 相关能力仍可正常工作。
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

// Run 启动一次新的 AI 对话执行。
// 会初始化执行状态并持久化，然后驱动 ADK Runner 流式处理，
// 将每个 Agent 事件通过 emit 回调推送给调用方。
// 若执行中途遇到审批中断，会保存断点后返回，调用方通过 ResumeStream 继续。
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
		Phase:          "running",
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

// Resume 以非流式方式处理审批结果（通过/拒绝）。
func (o *Orchestrator) Resume(ctx context.Context, req airuntime.ResumeRequest) (*airuntime.ResumeResult, error) {
	return o.resume(ctx, req, nil)
}

// ResumeStream 以流式方式处理审批结果，并将后续执行事件推送给调用方。
func (o *Orchestrator) ResumeStream(ctx context.Context, req airuntime.ResumeRequest, emit airuntime.StreamEmitter) (*airuntime.ResumeResult, error) {
	return o.resume(ctx, req, emit)
}

// resume 是 Resume/ResumeStream 的共享实现。
// 当 req.Approved==false 时直接更新状态为 rejected 并结束；
// 当审批通过但找不到断点时将步骤标记为成功（无需继续执行）；
// 找到断点时通过 ADK ResumeWithParams 将审批结果注入后继续执行。
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
		emit(o.converter.OnChainStarted(state.TurnID))
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

// streamExecution 消费 ADK 事件迭代器，将事件转换为 SSE 推送，并更新执行状态。
// 遇到审批中断时保存断点后提前返回，事件循环结束时更新状态为 completed。
func (o *Orchestrator) streamExecution(ctx context.Context, iter *adk.AsyncIterator[*adk.AgentEvent], state *airuntime.ExecutionState, emit airuntime.StreamEmitter) (*airuntime.ResumeResult, error) {
	if iter == nil {
		return nil, fmt.Errorf("event iterator is nil")
	}
	if emit == nil {
		emit = func(airuntime.StreamEvent) bool { return true }
	}

	var lastText string
	parser := airuntime.NewPlanParser()
	detector := airuntime.NewPhaseDetector()
	planningText := ""
	planningStarted := false
	planningCompleted := false
	executingStarted := false
	chainStarted := false
	chainCollapsed := false
	finalAnswerStarted := false
	activeNodeID := ""
	activeNodeKind := airuntime.ChainNodeKind("")
	planNodeID := fmt.Sprintf("plan:%s", state.PlanID)
	executeNodeID := fmt.Sprintf("execute:%s", state.PlanID)

	emitChainStarted := func() {
		if chainStarted {
			return
		}
		emit(o.converter.OnChainStarted(state.TurnID))
		chainStarted = true
	}
	openChainNode := func(info airuntime.ChainNodeInfo) {
		emitChainStarted()
		if activeNodeID != "" && activeNodeID != info.NodeID {
			emit(o.converter.OnChainNodeClose(airuntime.ChainNodeInfo{
				TurnID: state.TurnID,
				NodeID: activeNodeID,
				Kind:   activeNodeKind,
				Status: "done",
			}))
		}
		emit(o.converter.OnChainNodeOpen(info))
		activeNodeID = info.NodeID
		activeNodeKind = info.Kind
	}
	patchChainNode := func(info airuntime.ChainNodeInfo) {
		emitChainStarted()
		emit(o.converter.OnChainNodePatch(info))
	}
	closeActiveNode := func(status string) {
		if activeNodeID == "" {
			return
		}
		emit(o.converter.OnChainNodeClose(airuntime.ChainNodeInfo{
			TurnID: state.TurnID,
			NodeID: activeNodeID,
			Kind:   activeNodeKind,
			Status: firstNonEmpty(status, "done"),
		}))
		activeNodeID = ""
		activeNodeKind = ""
	}
	startFinalAnswer := func() {
		if !chainCollapsed {
			closeActiveNode("done")
			emit(o.converter.OnChainCollapsed(state.TurnID))
			chainCollapsed = true
		}
		if finalAnswerStarted {
			return
		}
		emit(o.converter.OnFinalAnswerStarted(state.TurnID))
		finalAnswerStarted = true
	}

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
		detectedPhase := detector.Detect(event)
		if detectedPhase == string(airuntime.PhaseReplanning) && state.Phase != string(airuntime.PhaseReplanning) {
			reason := strings.TrimSpace(eventMessageTextPreview(event))
			if reason == "" {
				reason = "执行结果触发重新规划"
			}
			openChainNode(airuntime.ChainNodeInfo{
				TurnID:  state.TurnID,
				NodeID:  fmt.Sprintf("replan:%s", state.PlanID),
				Kind:    airuntime.ChainNodeReplan,
				Title:   "发现新信息，正在调整计划",
				Status:  "loading",
				Summary: reason,
			})
			state.Phase = string(airuntime.PhaseReplanning)
			planningCompleted = false
			planningStarted = false
			planningText = ""
		}
		contents, err := eventTextContents(event)
		if err != nil {
			return nil, fmt.Errorf("failed to parse agent message chunks: %w", err)
		}
		if isToolOutputEvent(event) {
			if !executingStarted {
				openChainNode(airuntime.ChainNodeInfo{
					TurnID:  state.TurnID,
					NodeID:  executeNodeID,
					Kind:    airuntime.ChainNodeExecute,
					Title:   "正在执行步骤",
					Status:  "loading",
					Summary: "开始执行计划步骤",
				})
				planningCompleted = true
				executingStarted = true
			}

			toolName := strings.TrimSpace(event.Output.MessageOutput.ToolName)
			toolResult := strings.TrimSpace(strings.Join(contents, ""))
			stepID, step := claimStepForTool(state, detector, toolName)
			startingStep := step.Status != airuntime.StepRunning
			step.Status = airuntime.StepRunning
			step.ToolName = firstNonEmpty(toolName, step.ToolName)
			if strings.TrimSpace(step.Title) == "" {
				step.Title = firstNonEmpty(step.ToolName, step.StepID, "执行步骤")
			}
			if strings.TrimSpace(toolResult) != "" {
				step.UserVisibleSummary = toolResult
			}
			state.Steps[stepID] = step
			if startingStep {
				openChainNode(airuntime.ChainNodeInfo{
					TurnID:  state.TurnID,
					NodeID:  fmt.Sprintf("tool:%s", stepID),
					Kind:    airuntime.ChainNodeTool,
					Title:   firstNonEmpty(step.Title, step.ToolName, "正在调用工具"),
					Status:  "loading",
					Summary: firstNonEmpty(step.UserVisibleSummary, "正在执行当前步骤"),
				})
			}
			patchChainNode(airuntime.ChainNodeInfo{
				TurnID:  state.TurnID,
				NodeID:  fmt.Sprintf("tool:%s", stepID),
				Kind:    airuntime.ChainNodeTool,
				Summary: firstNonEmpty(step.UserVisibleSummary, toolResult),
				Details: []any{compactEventData(map[string]any{
					"tool_name": step.ToolName,
					"result": map[string]any{
						"ok":   true,
						"data": toolResult,
					},
				})},
			})
			step.Status = airuntime.StepSucceeded
			state.Steps[stepID] = step
			closeActiveNode("done")
			_ = o.executions.Save(ctx, *state)
			continue
		}
		for _, content := range contents {
			if strings.TrimSpace(content) == "" {
				continue
			}
			if !planningStarted {
				title := "整理执行步骤"
				summary := "正在分析并整理执行计划"
				if state.Phase == string(airuntime.PhaseReplanning) {
					title = "动态调整计划"
					summary = "正在根据最新结果调整执行计划"
				}
				openChainNode(airuntime.ChainNodeInfo{
					TurnID:  state.TurnID,
					NodeID:  planNodeID,
					Kind:    airuntime.ChainNodePlan,
					Title:   title,
					Status:  "loading",
					Summary: summary,
				})
				planningStarted = true
			}
			if !planningCompleted {
				planningText = mergeTextProgress(planningText, content)
				patchChainNode(airuntime.ChainNodeInfo{
					TurnID:  state.TurnID,
					NodeID:  planNodeID,
					Kind:    airuntime.ChainNodePlan,
					Summary: strings.TrimSpace(content),
				})
				if plan, ok := parser.Extract(state.PlanID, state.TurnID, planningText); ok {
					patchChainNode(airuntime.ChainNodeInfo{
						TurnID:  state.TurnID,
						NodeID:  planNodeID,
						Kind:    airuntime.ChainNodePlan,
						Summary: firstNonEmpty(plan.Summary, "已生成执行步骤"),
						Details: stepsToChainDetails(plan.Steps),
					})
					for i := range plan.Steps {
						state.Steps[plan.Steps[i].ID] = airuntime.StepState{
							StepID: plan.Steps[i].ID,
							Title:  firstNonEmpty(plan.Steps[i].Title, plan.Steps[i].Content),
							Status: airuntime.StepPending,
						}
					}
					closeActiveNode("done")
					openChainNode(airuntime.ChainNodeInfo{
						TurnID:  state.TurnID,
						NodeID:  executeNodeID,
						Kind:    airuntime.ChainNodeExecute,
						Title:   "正在执行步骤",
						Status:  "loading",
						Summary: "开始执行计划步骤",
					})
					state.Phase = string(airuntime.PhaseExecuting)
					planningCompleted = true
					executingStarted = true
				}
				lastText = mergeTextProgress(lastText, content)
				continue
			}
			startFinalAnswer()
			emit(o.converter.OnFinalAnswerDelta(state.TurnID, content))
			lastText = mergeTextProgress(lastText, content)
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			stepID := interruptStepID(event)
			pending := o.pendingApprovalFromInterrupt(state, stepID, event)
			if !executingStarted {
				planningCompleted = true
				executingStarted = true
			}
			state.Status = airuntime.ExecutionStatusWaitingApproval
			state.Phase = string(airuntime.PhaseExecuting)
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
			openChainNode(airuntime.ChainNodeInfo{
				TurnID:  state.TurnID,
				NodeID:  fmt.Sprintf("approval:%s", stepID),
				Kind:    airuntime.ChainNodeApproval,
				Title:   pending.Title,
				Status:  "waiting",
				Summary: pending.Summary,
				Approval: compactEventData(map[string]any{
					"id":         pending.ID,
					"request_id": pending.ID,
					"title":      pending.Title,
					"risk":       pending.Risk,
					"details": map[string]any{
						"plan_id":   pending.PlanID,
						"step_id":   pending.StepID,
						"tool_name": pending.ToolName,
					},
				}),
			})
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

	if !executingStarted {
		openChainNode(airuntime.ChainNodeInfo{
			TurnID:  state.TurnID,
			NodeID:  executeNodeID,
			Kind:    airuntime.ChainNodeExecute,
			Title:   "正在执行步骤",
			Status:  "loading",
			Summary: "开始执行计划步骤",
		})
	}
	state.Status = airuntime.ExecutionStatusCompleted
	state.Phase = string(airuntime.PhaseExecuting)
	_ = o.executions.Save(ctx, *state)
	closeActiveNode("done")
	if !chainCollapsed {
		emit(o.converter.OnChainCollapsed(state.TurnID))
		chainCollapsed = true
	}
	if finalAnswerStarted {
		emit(o.converter.OnFinalAnswerDone(state.TurnID))
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

func stepsToChainDetails(steps []airuntime.PlanStep) []any {
	if len(steps) == 0 {
		return nil
	}
	out := make([]any, 0, len(steps))
	for _, step := range steps {
		out = append(out, compactEventData(map[string]any{
			"id":        step.ID,
			"title":     step.Title,
			"content":   step.Content,
			"status":    step.Status,
			"tool_hint": step.ToolHint,
		}))
	}
	return out
}

func stepEventFromState(state airuntime.ExecutionState, step airuntime.StepState, status, result string) airuntime.StepEvent {
	event := airuntime.StepEvent{
		PlanID:  state.PlanID,
		TurnID:  state.TurnID,
		StepID:  step.StepID,
		Title:   step.Title,
		Status:  firstNonEmpty(status, string(step.Status)),
		Expert:  step.Expert,
		Summary: step.UserVisibleSummary,
		Result:  strings.TrimSpace(result),
	}
	if strings.TrimSpace(step.ToolName) != "" || len(step.ToolArgs) > 0 {
		event.Tool = &airuntime.ToolDescriptor{
			Name: step.ToolName,
			Args: step.ToolArgs,
			Mode: step.Mode,
			Risk: step.Risk,
		}
	}
	return event
}

func claimStepForTool(state *airuntime.ExecutionState, detector *airuntime.PhaseDetector, toolName string) (string, airuntime.StepState) {
	if state == nil {
		stepID := detector.NextStepID()
		return stepID, airuntime.StepState{
			StepID:   stepID,
			Title:    firstNonEmpty(toolName, stepID, "执行步骤"),
			Status:   airuntime.StepPending,
			ToolName: toolName,
		}
	}

	for _, status := range []airuntime.StepStatus{airuntime.StepRunning, airuntime.StepPending, airuntime.StepWaitingApproval} {
		for _, stepID := range sortedStepIDs(state.Steps) {
			step := state.Steps[stepID]
			if step.Status != status {
				continue
			}
			if strings.TrimSpace(toolName) == "" || strings.EqualFold(strings.TrimSpace(step.ToolName), toolName) {
				return stepID, step
			}
		}
	}

	stepID := detector.NextStepID()
	return stepID, airuntime.StepState{
		StepID:   stepID,
		Title:    firstNonEmpty(toolName, stepID, "执行步骤"),
		Status:   airuntime.StepPending,
		ToolName: toolName,
	}
}

func sortedStepIDs(steps map[string]airuntime.StepState) []string {
	if len(steps) == 0 {
		return nil
	}
	ids := make([]string, 0, len(steps))
	for stepID := range steps {
		ids = append(ids, stepID)
	}
	sort.Strings(ids)
	return ids
}

func isToolOutputEvent(event *adk.AgentEvent) bool {
	return event != nil &&
		event.Output != nil &&
		event.Output.MessageOutput != nil &&
		event.Output.MessageOutput.Role == schema.Tool
}

func compactEventData(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch v := value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(v) == "" {
				continue
			}
		case map[string]any:
			if len(v) == 0 {
				continue
			}
		}
		out[key] = value
	}
	return out
}

func mergeTextProgress(previous, current string) string {
	if strings.TrimSpace(current) == "" {
		return previous
	}
	if previous == "" {
		return current
	}
	if current == previous {
		return previous
	}
	if strings.HasPrefix(current, previous) {
		return current
	}
	if strings.HasPrefix(previous, current) {
		return previous
	}
	return previous + current
}

// eventTextContents 从 AgentEvent 提取消息文本分片。
// 对流式事件按 MessageStream 的原始 chunk 读取，避免被拼接成单个大文本。
func eventTextContents(event *adk.AgentEvent) ([]string, error) {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil
	}

	output := event.Output.MessageOutput
	if !output.IsStreaming {
		if output.Message == nil {
			return nil, nil
		}
		return []string{output.Message.Content}, nil
	}

	if output.MessageStream == nil {
		return nil, nil
	}

	chunks := make([]string, 0, 4)
	for {
		frame, err := output.MessageStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("recv message stream: %w", err)
		}
		if frame == nil {
			continue
		}
		chunks = append(chunks, frame.Content)
	}
	return chunks, nil
}

func eventMessageTextPreview(event *adk.AgentEvent) string {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil || event.Output.MessageOutput.Message == nil {
		return ""
	}
	return event.Output.MessageOutput.Message.Content
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

// pendingApprovalFromInterrupt 从 ADK 中断事件构造 PendingApproval 记录。
// 若工具未提供 Summary 则由 SummaryRenderer 自动生成人类可读摘要。
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

// interruptApprovalInfo 从 ADK 中断事件提取审批元信息。
// 兼容强类型（ApprovalInterruptInfo）和松散 map[string]any 两种形式。
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
