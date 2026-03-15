// 本文件实现基于 eino ADK 的 AI 编排器。
//
// Orchestrator 是 AI 模块对外的唯一入口，负责：
//   - 初始化 ADK Runner，并驱动单条后端主路径执行
//   - 接收用户请求，将执行进度收口为 tool_call/tool_approval/tool_result 事件
//   - 处理人工审批中断：在敏感变更工具调用前暂停执行，等待外部 Resume 信号
//   - 持久化执行状态（ExecutionStore）和断点（CheckpointStore）以支持会话恢复

package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
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
	"github.com/cy77cc/OpsPilot/internal/model"
)

// executionStats 累加单次执行的统计数据。
type executionStats struct {
	startedAt     time.Time
	firstTokenAt  time.Time
	firstTokenOnce sync.Once

	promptTokens     int64
	completionTokens int64
	totalTokens      int64

	toolCallCount  int
	toolErrorCount int

	approvalCount  int
	approvalWaitMs int64
}

func newExecutionStats() *executionStats {
	return &executionStats{
		startedAt: time.Now().UTC(),
	}
}

func (s *executionStats) recordTokens(prompt, completion, total int64) {
	s.promptTokens += prompt
	s.completionTokens += completion
	s.totalTokens += total
}

func (s *executionStats) recordFirstToken() {
	s.firstTokenOnce.Do(func() {
		s.firstTokenAt = time.Now().UTC()
	})
}

func (s *executionStats) firstTokenMs() int {
	if s.firstTokenAt.IsZero() {
		return 0
	}
	return int(s.firstTokenAt.Sub(s.startedAt).Milliseconds())
}

func (s *executionStats) tokensPerSecond() float64 {
	if s.firstTokenAt.IsZero() {
		return 0
	}
	elapsed := time.Since(s.firstTokenAt).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(s.completionTokens) / elapsed
}

// extractTokenUsage 从 AgentEvent 提取 token 使用量。
func extractTokenUsage(event *adk.AgentEvent) (prompt, completion, total int64) {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return 0, 0, 0
	}
	msg := event.Output.MessageOutput.Message
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Usage == nil {
		return 0, 0, 0
	}
	usage := msg.ResponseMeta.Usage
	return int64(usage.PromptTokens), int64(usage.CompletionTokens), int64(usage.TotalTokens)
}

// isToolError 检查工具输出是否包含错误。
func isToolError(event *adk.AgentEvent) bool {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return false
	}
	msg := event.Output.MessageOutput.Message
	if msg == nil {
		return false
	}
	content := strings.TrimSpace(msg.Content)
	if strings.Contains(content, `"error"`) || strings.Contains(content, `"failed"`) {
		return true
	}
	return false
}

// isToolCallEvent 检查是否为工具调用请求事件。
func isToolCallEvent(event *adk.AgentEvent) bool {
	return event != nil &&
		event.Output != nil &&
		event.Output.MessageOutput != nil &&
		event.Output.MessageOutput.Role == schema.Assistant &&
		event.Output.MessageOutput.Message != nil &&
		len(event.Output.MessageOutput.Message.ToolCalls) > 0
}

// sanitizeErrorMessage 脱敏错误信息。
func sanitizeErrorMessage(msg string) string {
	if len(msg) > 500 {
		msg = msg[:500]
	}
	msg = regexp.MustCompile(`(?i)(api[_-]?key|token|secret)[\s:=]*[\w-]{20,}`).ReplaceAllString(msg, "[REDACTED]")
	msg = regexp.MustCompile(`/[\w/.-]+`).ReplaceAllString(msg, "[PATH]")
	return msg
}

// Orchestrator 封装 ADK Runner，提供流式执行与审批恢复能力。
type Orchestrator struct {
	runner      *adk.Runner                      // ADK 运行器；nil 表示模型不可用
	checkpoints *airuntime.CheckpointStore       // 保存 Agent 断点，支持审批后续跑
	executions  *airuntime.ExecutionStore        // 保存每次执行的状态快照
	converter   *airuntime.SSEConverter          // 将 Agent 事件转换为标准 SSE StreamEvent
	approvals   *airuntime.ApprovalDecisionMaker // 判断工具调用是否需要人工审批
	summaries   *approvaltools.SummaryRenderer   // 生成审批请求的人类可读摘要
	usageLogDAO common.UsageLogDAOInterface      // 使用统计 DAO（从 deps 获取）
	initErr     error                            // 初始化阶段的根因错误，runner 不可用时向上返回
	runQuery    func(context.Context, string, ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent]
}

// NewOrchestrator 创建并初始化 Orchestrator。
// 若 Agent 构建失败（如模型不可用），返回不含 runner 的降级实例，
// Run 调用时会立即返回错误，Resume 相关能力仍可正常工作。
func NewOrchestrator(_ any, executionStore *airuntime.ExecutionStore, deps common.PlatformDeps) *Orchestrator {
	ctx := context.Background()
	sceneResolver := airuntime.NewSceneConfigResolver(nil)
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
		PlatformDeps:  deps,
		DecisionMaker: approvals,
	})
	if err != nil {
		return &Orchestrator{
			executions:    executionStore,
			checkpoints:   checkpointStore,
			converter:     airuntime.NewSSEConverter(),
			approvals:     approvals,
			summaries:     summaries,
			usageLogDAO:   deps.UsageLogDAO,
			initErr:       err,
		}
	}

	return &Orchestrator{
		runner: adk.NewRunner(ctx, adk.RunnerConfig{
			Agent:           agent,
			CheckPointStore: checkpointStore,
			EnableStreaming: true,
		}),
		checkpoints:   checkpointStore,
		executions:    executionStore,
		converter:     airuntime.NewSSEConverter(),
		approvals:     approvals,
		summaries:     summaries,
		usageLogDAO:   deps.UsageLogDAO,
		runQuery:      nil,
	}
}

// Run 启动一次新的 AI 对话执行。
// 会初始化执行状态并持久化，然后驱动 ADK Runner 流式处理，
// 将每个 Agent 事件通过 emit 回调推送给调用方。
// 若执行中途遇到审批中断，会保存断点后返回，调用方通过 ResumeStream 继续。
func (o *Orchestrator) Run(ctx context.Context, req airuntime.RunRequest, emit airuntime.StreamEmitter) error {
	startedAt := time.Now().UTC()
	if o == nil || o.runner == nil {
		if o != nil && o.runQuery != nil {
			goto runnerReady
		}
		if o != nil && o.initErr != nil {
			return fmt.Errorf("orchestrator unavailable: %w", o.initErr)
		}
		return fmt.Errorf("orchestrator runner is nil")
	}
runnerReady:
	if strings.TrimSpace(req.Message) == "" {
		return fmt.Errorf("message is empty")
	}
	if emit == nil {
		emit = func(airuntime.StreamEvent) bool { return true }
	}

	ctx = airuntime.ContextWithRuntimeContext(ctx, req.RuntimeContext)
	sessionID := firstNonEmpty(req.SessionID, uuid.NewString())
	planID := uuid.NewString()
	turnID := uuid.NewString()
	checkpointID := uuid.NewString()
	envelope := buildRuntimeContextEnvelope(req.RuntimeContext)
	composedInput := composeUserInput(envelope, req.Message)
	adkValues := map[string]any{
		airuntime.SessionKeyRuntimeContext: req.RuntimeContext,
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
	}
	_ = o.executions.Save(ctx, state)

	// 发送 meta 事件
	emit(airuntime.StreamEvent{Type: airuntime.EventMeta, Data: map[string]any{
		"session_id": sessionID,
		"plan_id":    planID,
		"turn_id":    turnID,
		"trace_id":   state.TraceID,
	}})

	query := o.runQuery
	if query == nil {
		query = o.runner.Query
	}
	iter := query(ctx, composedInput,
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

func buildRuntimeContextEnvelope(runtimeCtx airuntime.RuntimeContext) string {
	lines := make([]string, 0, 5)
	appendLine := func(key, value string) {
		value = normalizeRuntimeLine(value)
		if value == "" {
			return
		}
		lines = append(lines, fmt.Sprintf("%s: %s", key, value))
	}

	appendLine("scene", runtimeCtx.Scene)
	appendLine("project", firstNonEmpty(runtimeCtx.ProjectName, runtimeCtx.ProjectID))
	appendLine("page", firstNonEmpty(runtimeCtx.CurrentPage, runtimeCtx.Route))
	appendLine("selected_resources", summarizeSelectedResources(runtimeCtx.SelectedResources))

	if len(lines) == 0 {
		return ""
	}
	return "[Runtime Context]\n" + strings.Join(lines, "\n")
}

func composeUserInput(envelope, raw string) string {
	if strings.TrimSpace(envelope) == "" {
		return "[User Request]\n" + raw
	}
	return envelope + "\n\n[User Request]\n" + raw
}

func normalizeRuntimeLine(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func summarizeSelectedResources(resources []airuntime.SelectedResource) string {
	if len(resources) == 0 {
		return ""
	}

	items := make([]string, 0, len(resources))
	for _, resource := range resources {
		name := normalizeRuntimeLine(firstNonEmpty(resource.Name, resource.ID))
		typ := normalizeRuntimeLine(resource.Type)
		if name == "" && typ == "" {
			continue
		}
		switch {
		case name != "" && typ != "":
			items = append(items, fmt.Sprintf("%s(%s)", name, typ))
		case name != "":
			items = append(items, name)
		default:
			items = append(items, fmt.Sprintf("unknown(%s)", typ))
		}
	}
	return strings.Join(items, ", ")
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
	observeApprovalResolution := func(status string) {
		if state.PendingApproval == nil {
			return
		}
		duration := time.Duration(0)
		if !state.PendingApproval.CreatedAt.IsZero() {
			duration = time.Since(state.PendingApproval.CreatedAt)
		}
		aiobs.ObserveThoughtChainApproval(aiobs.ThoughtChainApprovalRecord{
			Scene:    state.Scene,
			Status:   status,
			Duration: duration,
		})
		aiobs.ObserveToolExecutionLifecycle(aiobs.ToolExecutionLifecycleRecord{
			Event:     "approval_resolved",
			TraceID:   state.TraceID,
			SessionID: state.SessionID,
			ToolName:  state.PendingApproval.ToolName,
			Status:    status,
		})
	}
	if emit != nil {
		emit(airuntime.StreamEvent{Type: airuntime.EventMeta, Data: map[string]any{
			"session_id": state.SessionID,
			"plan_id":    state.PlanID,
			"turn_id":    state.TurnID,
			"trace_id":   state.TraceID,
		}})
	}

	if !req.Approved {
		observeApprovalResolution("rejected")
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
		emit(o.converter.OnDone(string(state.Status)))
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
		observeApprovalResolution("approved")
		state.Status = airuntime.ExecutionStatusCompleted
		if step := state.Steps[stepID]; step.StepID != "" {
			step.Status = airuntime.StepSucceeded
			state.Steps[stepID] = step
		}
		if state.PendingApproval != nil {
			state.PendingApproval.Status = "approved"
		}
		_ = o.executions.Save(ctx, state)
		emit(o.converter.OnDone(string(state.Status)))
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
		resumeData := map[string]any{
			"approved": true,
			"reason":   strings.TrimSpace(req.Reason),
		}
		// 如果有编辑后的参数，传递给 ADK
		if strings.TrimSpace(req.EditedArguments) != "" {
			resumeData["edited_arguments"] = strings.TrimSpace(req.EditedArguments)
		}
		params.Targets = map[string]any{
			target: resumeData,
		}
	}
	ctx = airuntime.ContextWithRuntimeContext(ctx, state.RuntimeContext)
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
	observeApprovalResolution("approved")
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

	// 初始化统计累加器
	stats := newExecutionStats()

	var lastText string

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}

		// 提取 token 使用量
		prompt, completion, total := extractTokenUsage(event)
		if total > 0 {
			stats.recordTokens(prompt, completion, total)
		}

		if event.Err != nil {
			// 写入失败记录
			o.writeUsageLog(ctx, state, stats, "failed", "model_error", event.Err)
			state.Status = airuntime.ExecutionStatusFailed
			state.Phase = "failed"
			_ = o.executions.Save(ctx, *state)
			emit(o.converter.OnError(state.Phase, event.Err))
			aiobs.ObserveToolExecutionLifecycle(aiobs.ToolExecutionLifecycleRecord{
				Event:     "execution_failed",
				TraceID:   state.TraceID,
				SessionID: state.SessionID,
				Status:    "failed",
			})
			return nil, event.Err
		}

		// 检测工具调用请求事件
		if isToolCallEvent(event) {
			stats.toolCallCount++
			stats.recordFirstToken()
			for _, tc := range event.Output.MessageOutput.Message.ToolCalls {
				emit(o.converter.OnToolCall(
					tc.ID,
					tc.Function.Name,
					"", // tool_display_name 将由工具注册表解析
					tc.Function.Arguments,
				))
				aiobs.ObserveToolExecutionLifecycle(aiobs.ToolExecutionLifecycleRecord{
					Event:     "tool_call",
					TraceID:   state.TraceID,
					SessionID: state.SessionID,
					ToolName:  tc.Function.Name,
					Status:    "pending",
				})
			}
			continue
		}

		contents, err := eventTextContents(event)
		if err != nil {
			return nil, fmt.Errorf("failed to parse agent message chunks: %w", err)
		}

		// 处理工具输出事件
		if isToolOutputEvent(event) {
			stats.toolCallCount++
			if isToolError(event) {
				stats.toolErrorCount++
			}
			toolName := strings.TrimSpace(event.Output.MessageOutput.ToolName)
			toolResult := strings.TrimSpace(strings.Join(contents, ""))
			// 提取 call_id（如果有）
			callID := extractCallIDFromToolResult(event)
			emit(airuntime.StreamEvent{
				Type: airuntime.EventToolResult,
				Data: compactEventData(map[string]any{
					"call_id":   callID,
					"tool_name": toolName,
					"result":    toolResult,
				}),
			})
			aiobs.ObserveToolExecutionLifecycle(aiobs.ToolExecutionLifecycleRecord{
				Event:     "tool_result",
				TraceID:   state.TraceID,
				SessionID: state.SessionID,
				ToolName:  toolName,
				Status:    "completed",
			})
			continue
		}

		// 处理文本增量事件
		for _, content := range contents {
			if strings.TrimSpace(content) == "" {
				continue
			}
			stats.recordFirstToken()
			emit(airuntime.StreamEvent{
				Type: airuntime.EventDelta,
				Data: map[string]any{"content": content},
			})
			lastText = mergeTextProgress(lastText, content)
		}

		// 处理审批中断事件
		if event.Action != nil && event.Action.Interrupted != nil {
			stats.approvalCount++
			return o.handleInterrupt(ctx, event, state, emit, stats)
		}
	}

	// 循环结束，写入成功记录
	o.writeUsageLog(ctx, state, stats, "completed", "", nil)

	state.Status = airuntime.ExecutionStatusCompleted
	state.Phase = "completed"
	_ = o.executions.Save(ctx, *state)
	emit(o.converter.OnDone(string(state.Status)))

	aiobs.ObserveToolExecutionLifecycle(aiobs.ToolExecutionLifecycleRecord{
		Event:     "execution_completed",
		TraceID:   state.TraceID,
		SessionID: state.SessionID,
		Status:    "completed",
	})
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

// handleInterrupt 处理审批中断事件。
func (o *Orchestrator) handleInterrupt(ctx context.Context, event *adk.AgentEvent, state *airuntime.ExecutionState, emit airuntime.StreamEmitter, stats *executionStats) (*airuntime.ResumeResult, error) {
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

	// 写入等待审批记录
	o.writeUsageLog(ctx, state, stats, "waiting_approval", "", nil)

	// 发送 tool_approval 事件
	emit(airuntime.StreamEvent{
		Type: airuntime.EventToolApproval,
		Data: compactEventData(map[string]any{
			"call_id":           stepID,
			"tool_name":         pending.ToolName,
			"tool_display_name": pending.ToolDisplayName,
			"risk":              pending.Risk,
			"summary":           pending.Summary,
			"arguments_json":    pending.ArgumentsInJSON,
			"approval_id":       pending.ID,
			"checkpoint_id":     pending.CheckpointID,
			"plan_id":           pending.PlanID,
			"step_id":           pending.StepID,
		}),
	})
	emit(o.converter.OnDone(string(state.Status)))

	aiobs.ObserveThoughtChainApproval(aiobs.ThoughtChainApprovalRecord{
		Scene:    state.Scene,
		Status:   "pending",
		Duration: 0,
	})
	aiobs.ObserveToolExecutionLifecycle(aiobs.ToolExecutionLifecycleRecord{
		Event:     "tool_approval_pending",
		TraceID:   state.TraceID,
		SessionID: state.SessionID,
		ToolName:  pending.ToolName,
		Status:    "waiting_approval",
	})

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

// extractCallIDFromToolResult 从工具结果事件中提取 call_id。
func extractCallIDFromToolResult(event *adk.AgentEvent) string {
	if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
		return ""
	}
	msg := event.Output.MessageOutput.Message
	if msg == nil {
		return ""
	}
	// 尝试从 Message 的 ToolCallID 字段获取
	if msg.ToolCallID != "" {
		return msg.ToolCallID
	}
	// 尝试从 Content 中解析 call_id
	content := strings.TrimSpace(msg.Content)
	if content == "" {
		return ""
	}
	// 尝试解析 JSON 格式的结果
	var result map[string]any
	if err := json.Unmarshal([]byte(content), &result); err == nil {
		if callID, ok := result["call_id"].(string); ok {
			return callID
		}
	}
	return ""
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
		ID:              uuid.NewString(),
		PlanID:          state.PlanID,
		StepID:          stepID,
		CheckpointID:    firstNonEmpty(info.CheckpointID, state.CheckpointID),
		Target:          firstNonEmpty(info.Target, stepID),
		Status:          "pending",
		Title:           firstNonEmpty(info.ToolDisplayName, info.ToolName, "待确认步骤"),
		Mode:            firstNonEmpty(info.Mode, "mutating"),
		Risk:            firstNonEmpty(info.RiskLevel, "medium"),
		Summary:         summary,
		ApprovalKey:     airuntime.ResumeIdentity(state.SessionID, state.PlanID, stepID),
		ToolName:        info.ToolName,
		ToolDisplayName: info.ToolDisplayName,
		Params:          info.Params,
		ArgumentsInJSON: info.ArgumentsInJSON,
		CreatedAt:       timeNowUTC(),
		ExpiresAt:       timeNowUTC().Add(24 * time.Hour),
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
				CheckpointID:    mapString(info["checkpoint_id"]),
				Target:          mapString(info["target"]),
				ToolName:        mapString(info["tool_name"]),
				ToolDisplayName: mapString(info["tool_display_name"]),
				Mode:            mapString(info["mode"]),
				RiskLevel:       firstNonEmpty(mapString(info["risk_level"]), mapString(info["risk"])),
				Summary:         mapString(info["summary"]),
				Params:          mapParams(info["params"]),
				ArgumentsInJSON: mapString(info["arguments_json"]),
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

// writeUsageLog 写入使用统计记录。
func (o *Orchestrator) writeUsageLog(ctx context.Context, state *airuntime.ExecutionState, stats *executionStats, status, errorType string, err error) {
	if o.usageLogDAO == nil {
		return
	}

	log := &model.AIUsageLog{
		TraceID:          state.TraceID,
		SessionID:        state.SessionID,
		PlanID:           state.PlanID,
		TurnID:           state.TurnID,
		UserID:           0,
		Scene:            state.Scene,
		Operation:        "run",
		Status:           status,
		PromptTokens:     int(stats.promptTokens),
		CompletionTokens: int(stats.completionTokens),
		TotalTokens:      int(stats.totalTokens),
		DurationMs:       int(time.Since(stats.startedAt).Milliseconds()),
		FirstTokenMs:     stats.firstTokenMs(),
		TokensPerSecond:  stats.tokensPerSecond(),
		ApprovalCount:    stats.approvalCount,
		ApprovalStatus:   "none",
		ToolCallCount:    stats.toolCallCount,
		ToolErrorCount:   stats.toolErrorCount,
		ErrorType:        errorType,
	}

	if err != nil {
		log.ErrorMessage = sanitizeErrorMessage(err.Error())
	}

	if err := o.usageLogDAO.Create(ctx, log); err != nil {
		// 记录日志但不影响主流程
	}

	// 上报 Prometheus 指标
	if stats.firstTokenMs() > 0 {
		aiobs.ObserveFirstToken(state.Scene, time.Duration(stats.firstTokenMs())*time.Millisecond)
	}
}
