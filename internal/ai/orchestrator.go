// Package ai 实现 AI 编排核心逻辑。
//
// 架构概览:
//
//	┌─────────────────────────────────────────────────────────────┐
//	│                      Orchestrator                           │
//	│                                                             │
//	│   ┌─────────┐   ┌─────────┐   ┌──────────┐   ┌───────────┐ │
//	│   │ Rewrite │──▶│ Planner │──▶│ Executor │──▶│ Summarizer│ │
//	│   └─────────┘   └─────────┘   └──────────┘   └───────────┘ │
//	│        │             │             │              │        │
//	│        ▼             ▼             ▼              ▼        │
//	│   normalize      plan+tools    expert agents    answer     │
//	└─────────────────────────────────────────────────────────────┘
//
// 执行流程:
//  1. Rewrite: 将口语化输入改写为结构化目标
//  2. Plan: 解析资源并生成执行计划
//  3. Execute: 调用专家 Agent 执行各步骤
//  4. Summarize: 汇总结果生成最终答案
//
// 主要入口:
//   - NewOrchestrator: 创建编排器实例
//   - Run: 执行完整流水线
//   - Resume: 恢复等待审批的执行
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/availability"
	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/experts"
	"github.com/cy77cc/OpsPilot/internal/ai/observability"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/ai/state"
	"github.com/cy77cc/OpsPilot/internal/ai/summarizer"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/google/uuid"
)

// StreamEmitter 定义流式事件回调函数类型。
// 返回 true 继续执行，false 表示客户端断开。
type StreamEmitter func(StreamEvent) bool

// Orchestrator 是 AI 编排核心，管理执行流水线的状态和依赖。
//
// 字段说明:
//   - sessions: 会话状态存储，用于持久化聊天历史
//   - executions: 执行状态存储，用于追踪计划和步骤状态
//   - rewriter: 输入改写阶段，将口语化输入标准化
//   - planner: 任务规划阶段，生成执行计划
//   - executor: 任务执行阶段，调用专家 Agent
//   - summarizer: 结果总结阶段，生成最终答案
//   - metrics: 指标收集器，用于监控和统计
//   - observability: 可观测性处理器，用于追踪 LLM/工具/Agent 调用
//   - maxIters: 最大迭代次数，防止无限重规划
//   - heartbeatInterval: 心跳间隔，保持长连接活跃
type Orchestrator struct {
	sessions          *state.SessionState     // 会话状态存储
	executions        *runtime.ExecutionStore // 执行状态存储
	rewriter          *rewrite.Rewriter       // 输入改写阶段
	planner           *planner.Planner        // 任务规划阶段
	executor          *executor.Executor      // 任务执行阶段
	summarizer        *summarizer.Summarizer  // 结果总结阶段
	metrics           *AIMetrics              // 指标收集器
	observability     *observability.Handler  // 可观测性处理器
	maxIters          int                     // 最大迭代次数
	heartbeatInterval time.Duration           // 心跳间隔
}

// NewOrchestrator 创建编排器实例，初始化所有阶段。
//
// 参数:
//   - sessions: 会话状态存储，用于持久化聊天历史
//   - executions: 执行状态存储，用于追踪执行状态
//   - deps: 平台依赖，包含各种服务客户端
//
// 注意: 各阶段使用不同的模型配置:
//   - Rewrite: 使用 RewriteChatModel (轻量级，快速响应)
//   - Planner: 使用 ToolCallingChatModel (支持工具调用)
//   - Summarizer: 使用 SummarizerChatModel (总结能力强)
func NewOrchestrator(sessions *state.SessionState, executions *runtime.ExecutionStore, deps common.PlatformDeps) *Orchestrator {
	out := &Orchestrator{
		sessions:          sessions,
		executions:        executions,
		executor:          executor.New(executions),
		metrics:           NewAIMetrics(),
		observability:     observability.GetHandler(),
		maxIters:          2,
		heartbeatInterval: 10 * time.Second,
	}
	bootstrapCtx := context.Background()
	if rewriteModel, err := NewRewriteChatModel(bootstrapCtx); err == nil {
		if stage, stageErr := rewrite.NewWithADK(bootstrapCtx, rewriteModel); stageErr == nil {
			out.rewriter = stage
		}
	}
	if plannerModel, err := NewPlannerChatModel(bootstrapCtx); err == nil {
		if stage, stageErr := planner.NewWithADK(bootstrapCtx, plannerModel, deps); stageErr == nil {
			out.planner = stage
		}
	}
	if expertModel, err := NewToolCallingChatModel(bootstrapCtx); err == nil {
		if stepRunner, stepErr := executor.NewAgentStepRunner(bootstrapCtx, expertModel, experts.DefaultRegistry(deps)); stepErr == nil {
			out.executor = executor.New(executions, executor.WithStepRunner(stepRunner))
		}
	}
	if summarizerModel, err := NewSummarizerChatModel(bootstrapCtx); err == nil {
		if stage, stageErr := summarizer.NewWithADK(bootstrapCtx, summarizerModel); stageErr == nil {
			out.summarizer = stage
		}
	}
	return out
}

// Run 启动编排流水线，处理用户消息并返回结果。
//
// 参数:
//   - ctx: 上下文，用于取消和超时控制
//   - req: 请求参数，包含用户消息和会话信息
//   - emit: 流式事件回调，用于向前端推送实时状态
//
// 返回: 成功返回 nil，失败返回错误
//
// 执行流程:
//  1. 参数校验和初始化 (sessionID, traceID)
//  2. 指标收集包装
//  3. 会话消息持久化
//  4. 执行状态初始化
//  5. 调用 Rewrite 阶段
//  6. 调用 Plan 阶段
//  7. 调用 Execute 阶段 (如果有计划)
//  8. 调用 Summarize 阶段
//  9. 清理和完成
func (o *Orchestrator) Run(ctx context.Context, req RunRequest, emit StreamEmitter) error {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return fmt.Errorf("message is required")
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	traceID := uuid.NewString()
	turnID := uuid.NewString()
	meta := events.EventMeta{
		SessionID: sessionID,
		TraceID:   traceID,
		TurnID:    turnID,
		Iteration: 1,
		Timestamp: time.Now().UTC(),
	}
	rollout := CurrentRolloutConfig()
	streamEmit := emit
	if o.metrics != nil {
		thoughtRun := o.metrics.StartThoughtChainRun()
		if thoughtRun != nil {
			defer thoughtRun.Finalize()
			streamEmit = func(evt StreamEvent) bool {
				thoughtRun.Observe(evt)
				if emit == nil {
					return true
				}
				return emit(evt)
			}
		}
	}
	projector := newTurnProjector(streamEmit, meta, rollout)

	if o.sessions != nil {
		if err := o.sessions.AppendMessage(ctx, sessionID, schema.UserMessage(message)); err != nil {
			return err
		}
		if err := o.sessions.EnsureTitle(ctx, sessionID, deriveSessionTitle(message)); err != nil {
			return err
		}
	}

	if o.executions != nil {
		resourceIDs := make([]string, 0, len(req.RuntimeContext.SelectedResources))
		for _, item := range req.RuntimeContext.SelectedResources {
			if item.ID != "" {
				resourceIDs = append(resourceIDs, item.ID)
			}
		}
		if err := o.executions.Save(ctx, runtime.ExecutionState{
			TraceID:   traceID,
			SessionID: sessionID,
			TurnID:    turnID,
			Message:   message,
			Status:    runtime.ExecutionStatusRunning,
			Phase:     "gateway_entry",
			RuntimeContext: runtime.ContextSnapshot{
				Scene:       req.RuntimeContext.Scene,
				Route:       req.RuntimeContext.Route,
				ProjectID:   req.RuntimeContext.ProjectID,
				CurrentPage: req.RuntimeContext.CurrentPage,
				ResourceIDs: resourceIDs,
			},
		}); err != nil {
			return err
		}
	}

	emitEvent(streamEmit, events.Meta, meta, map[string]any{
		"session_id": sessionID,
		"sessionId":  sessionID,
		"trace_id":   traceID,
		"traceId":    traceID,
		"turn_id":    turnID,
		"createdAt":  meta.Timestamp.Format(time.RFC3339),
	})
	projector.Start("rewrite")
	projector.SetState("streaming", "rewrite")
	stopHeartbeat := o.startHeartbeat(ctx, streamEmit, meta)
	defer stopHeartbeat()

	rewriteOut, rewriteErr := o.rewriter.RewriteStream(ctx, rewrite.Input{
		Message:           message,
		Scene:             req.RuntimeContext.Scene,
		CurrentPage:       req.RuntimeContext.CurrentPage,
		SelectedResources: toRewriteResources(req.RuntimeContext.SelectedResources),
	}, func(chunk string) {
		emitStageDelta(streamEmit, projector, meta, "rewrite", "loading", chunk, "", "")
	})
	if rewriteErr != nil {
		reply := rewriteFailureMessage(rewriteErr)
		projector.SetState("error", "rewrite")
		emitStageDelta(streamEmit, projector, meta, "rewrite", "error", reply, "", "")
		emitEvent(streamEmit, events.Error, meta, map[string]any{
			"message":    reply,
			"error_code": rewriteFailureCode(rewriteErr),
			"stage":      "rewrite",
		})
		projector.ExecutionEvent(string(events.Error), meta, map[string]any{
			"message":    reply,
			"error_code": rewriteFailureCode(rewriteErr),
			"stage":      "rewrite",
		})
		emitDeltaChunks(streamEmit, projector, meta, reply)
		if o.sessions != nil {
			if err := o.sessions.AppendMessage(ctx, sessionID, schema.AssistantMessage(reply, nil)); err != nil {
				return err
			}
		}
		sessionPayload := map[string]any{
			"id":        sessionID,
			"title":     deriveSessionTitle(message),
			"messages":  o.sessionMessages(ctx, sessionID),
			"createdAt": meta.Timestamp.Format(time.RFC3339),
			"updatedAt": time.Now().UTC().Format(time.RFC3339),
		}
		emitEvent(streamEmit, events.Done, meta, map[string]any{
			"session": sessionPayload,
		})
		projector.Done("error", "rewrite")
		if o.executions != nil {
			if st, err := o.executions.Load(ctx, sessionID); err == nil && st != nil {
				st.Status = runtime.ExecutionStatusFailed
				st.Phase = "rewrite_unavailable"
				st.ActiveBlockIDs = projector.ActiveBlockIDs()
				_ = o.executions.Save(ctx, *st)
			}
		}
		return nil
	}
	if o.metrics != nil {
		o.metrics.RecordRewrite(rewriteOut)
	}
	emitEvent(streamEmit, events.RewriteResult, meta, map[string]any{
		"rewrite":              rewriteOut,
		"user_visible_summary": rewriteOut.Narrative,
	})
	projector.SetState("streaming", "plan")

	reply, genErr := o.planAndReply(ctx, message, rewriteOut, req.RuntimeContext, meta, streamEmit, projector, sessionID)
	if genErr != nil {
		reply = fmt.Sprintf("AI 编排入口已经切换到新的宿主边界，但当前模型暂不可用：%s", genErr.Error())
		emitDeltaChunks(streamEmit, projector, meta, reply)
	}

	if o.sessions != nil {
		if err := o.sessions.AppendMessage(ctx, sessionID, schema.AssistantMessage(reply, nil)); err != nil {
			return err
		}
	}

	sessionPayload := map[string]any{
		"id":        sessionID,
		"title":     deriveSessionTitle(message),
		"messages":  o.sessionMessages(ctx, sessionID),
		"createdAt": meta.Timestamp.Format(time.RFC3339),
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	}
	emitEvent(streamEmit, events.Done, meta, map[string]any{
		"session": sessionPayload,
	})
	projector.Done("completed", "done")

	if o.executions != nil {
		st, err := o.executions.Load(ctx, sessionID)
		if err == nil && st != nil {
			if st.Status == runtime.ExecutionStatusRunning {
				st.Status = runtime.ExecutionStatusCompleted
				st.Phase = "completed"
			}
			st.ActiveBlockIDs = projector.ActiveBlockIDs()
			_ = o.executions.Save(ctx, *st)
		}
	}
	return nil
}

// Resume 恢复等待审批的执行流程。
//
// 当执行计划中某个步骤需要用户审批时，执行会暂停等待。
// 用户确认后调用此方法继续执行。
//
// 参数:
//   - ctx: 上下文
//   - req: 恢复请求，包含 sessionID、stepID 和审批结果
//
// 返回: 恢复结果，包含执行状态信息
//
// 特殊状态:
//   - idempotent: 重复的审批请求被忽略
//   - noop: 当前会话没有待审批步骤
//   - missing: 找不到对应的执行状态
func (o *Orchestrator) Resume(ctx context.Context, req ResumeRequest) (*ResumeResult, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if o.executor == nil {
		return &ResumeResult{
			Resumed:   false,
			SessionID: sessionID,
			Status:    "unavailable",
			Message:   "executor is not configured",
		}, nil
	}
	if o.executions != nil {
		if st, err := o.executions.Load(ctx, sessionID); err == nil && st != nil && st.PendingApproval != nil {
			stepID := firstNonEmpty(req.StepID, req.Target, st.PendingApproval.StepID)
			planID := firstNonEmpty(req.PlanID, st.PlanID, st.PendingApproval.PlanID)
			decisionHash := runtime.ApprovalDecisionHash(planID, stepID, req.Approved)
			if st.PendingApproval.DecisionHash == decisionHash {
				result := &ResumeResult{
					Resumed:   true,
					SessionID: sessionID,
					PlanID:    planID,
					StepID:    stepID,
					TurnID:    st.TurnID,
					Status:    "idempotent",
					Message:   "duplicate approval request ignored",
				}
				if o.metrics != nil {
					o.metrics.RecordResume(result.Status, nil)
				}
				return result, nil
			}
		}
	}
	result, err := o.executor.Resume(ctx, executor.ResumeRequest{
		SessionID: sessionID,
		PlanID:    req.PlanID,
		StepID:    firstNonEmpty(req.StepID, req.Target),
		Approved:  req.Approved,
		Reason:    req.Reason,
	})
	if err != nil {
		if o.metrics != nil {
			o.metrics.RecordResume("", err)
		}
		return nil, err
	}
	if result == nil {
		res := &ResumeResult{
			Resumed:   false,
			SessionID: sessionID,
			Status:    "missing",
			Message:   "execution state not found",
		}
		if o.metrics != nil {
			o.metrics.RecordResume(res.Status, nil)
		}
		return res, nil
	}
	if result.State.SessionID == "" {
		res := &ResumeResult{
			Resumed:   false,
			SessionID: sessionID,
			PlanID:    req.PlanID,
			StepID:    firstNonEmpty(req.StepID, req.Target),
			TurnID:    result.State.TurnID,
			Status:    "noop",
			Message:   "no pending approval for this session",
		}
		if o.metrics != nil {
			o.metrics.RecordResume(res.Status, nil)
		}
		return res, nil
	}
	pendingStepID := ""
	pendingPlanID := ""
	if result.State.PendingApproval != nil {
		pendingStepID = result.State.PendingApproval.StepID
		pendingPlanID = result.State.PendingApproval.PlanID
	}
	stepID := firstNonEmpty(req.StepID, req.Target, pendingStepID)
	planID := firstNonEmpty(req.PlanID, result.State.PlanID, pendingPlanID)
	status := firstNonEmpty(result.State.Phase)
	if approval := result.Approval(); approval != nil && approval.Status != "" {
		status = approval.Status
	}

	res := &ResumeResult{
		Resumed:     req.Approved,
		Interrupted: !req.Approved,
		SessionID:   sessionID,
		PlanID:      planID,
		StepID:      stepID,
		TurnID:      result.State.TurnID,
		Status:      status,
		Message:     resumeStatusMessage(status, req.Approved),
	}
	if o.metrics != nil {
		o.metrics.RecordResume(res.Status, nil)
	}
	return res, nil
}

// ResumeStream 以 SSE 方式恢复等待审批的执行流程，并继续在原 turn 上输出事件。
func (o *Orchestrator) ResumeStream(ctx context.Context, req ResumeRequest, emit StreamEmitter) (*ResumeResult, error) {
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	if o.executor == nil {
		res := &ResumeResult{
			Resumed:   false,
			SessionID: sessionID,
			Status:    "unavailable",
			Message:   "executor is not configured",
		}
		o.emitResumeFallbackStream(emit, sessionID, res)
		return res, nil
	}
	if o.executions == nil {
		res := &ResumeResult{
			Resumed:   false,
			SessionID: sessionID,
			Status:    "unavailable",
			Message:   "execution store is not configured",
		}
		o.emitResumeFallbackStream(emit, sessionID, res)
		return res, nil
	}
	state, err := o.executions.Load(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		res := &ResumeResult{
			Resumed:   false,
			SessionID: sessionID,
			Status:    "missing",
			Message:   "execution state not found",
		}
		o.emitResumeFallbackStream(emit, sessionID, res)
		return res, nil
	}
	if strings.TrimSpace(state.TurnID) == "" {
		state.TurnID = uuid.NewString()
		if saveErr := o.executions.Save(ctx, *state); saveErr != nil {
			return nil, saveErr
		}
	}
	meta := events.EventMeta{
		SessionID: sessionID,
		TraceID:   state.TraceID,
		PlanID:    firstNonEmpty(req.PlanID, state.PlanID),
		StepID:    firstNonEmpty(req.StepID, req.Target),
		TurnID:    state.TurnID,
		Iteration: 1,
		Timestamp: time.Now().UTC(),
	}
	rollout := CurrentRolloutConfig()
	projector := newTurnProjector(emit, meta, rollout)
	emitEvent(emit, events.Meta, meta, map[string]any{
		"session_id": sessionID,
		"sessionId":  sessionID,
		"trace_id":   state.TraceID,
		"traceId":    state.TraceID,
		"plan_id":    state.PlanID,
		"turn_id":    state.TurnID,
		"resumed":    true,
		"createdAt":  meta.Timestamp.Format(time.RFC3339),
	})
	projector.Start("execute")
	projector.SetState("streaming", "execute")
	stopHeartbeat := o.startHeartbeat(ctx, emit, meta)
	defer stopHeartbeat()

	result, err := o.executor.Resume(ctx, executor.ResumeRequest{
		SessionID: sessionID,
		PlanID:    req.PlanID,
		StepID:    firstNonEmpty(req.StepID, req.Target),
		Approved:  req.Approved,
		Reason:    req.Reason,
		EventMeta: meta,
		EmitEvent: func(name string, eventMeta events.EventMeta, data map[string]any) bool {
			projector.ExecutionEvent(name, eventMeta, data)
			emitEvent(emit, events.Name(name), eventMeta, data)
			emitExecuteStageDelta(emit, projector, eventMeta, name, data)
			return true
		},
	})
	if err != nil {
		projector.SetState("error", "execute")
		projector.ExecutionEvent(string(events.Error), meta, map[string]any{
			"message": err.Error(),
			"stage":   "execute",
		})
		emitEvent(emit, events.Error, meta, map[string]any{
			"message": err.Error(),
			"stage":   "execute",
		})
		projector.Done("error", "execute")
		return nil, err
	}

	res := buildResumeResult(req, result)
	if !req.Approved || res.Status == "noop" || res.Status == "missing" || res.Status == "idempotent" || res.Status == "unavailable" {
		emitDeltaChunks(emit, projector, meta, res.Message)
	} else if result != nil {
		approval := result.Approval()
		if approval == nil || strings.TrimSpace(approval.Status) == "" || strings.TrimSpace(approval.Status) == "approved" {
			planSnapshot := planFromExecutionState(result.State)
			if planSnapshot != nil {
				if _, summaryErr := o.emitExecutionSummary(ctx, result.State.Message, planSnapshot, result, emit, projector, meta); summaryErr != nil {
					projector.SetState("error", "summary")
				}
			}
		}
	}
	projector.SetState(turnStateStatus(result.State.Status), firstNonEmpty(resultPhase(result), "execute"))
	projector.Done(turnStateStatus(result.State.Status), firstNonEmpty(resultPhase(result), "execute"))
	if latest, loadErr := o.executions.Load(ctx, sessionID); loadErr == nil && latest != nil {
		latest.ActiveBlockIDs = projector.ActiveBlockIDs()
		if latest.TurnID == "" {
			latest.TurnID = meta.TurnID
		}
		_ = o.executions.Save(ctx, *latest)
	}
	donePayload := map[string]any{
		"status":  res.Status,
		"message": res.Message,
	}
	if o.sessions != nil {
		donePayload["session"] = map[string]any{
			"id":       sessionID,
			"messages": o.sessionMessages(ctx, sessionID),
		}
	}
	emitEvent(emit, events.Done, meta, donePayload)
	return res, nil
}

func (o *Orchestrator) emitResumeFallbackStream(emit StreamEmitter, sessionID string, res *ResumeResult) {
	if emit == nil || res == nil {
		return
	}
	turnID := strings.TrimSpace(res.TurnID)
	if turnID == "" {
		turnID = uuid.NewString()
	}
	meta := events.EventMeta{
		SessionID: strings.TrimSpace(sessionID),
		TraceID:   uuid.NewString(),
		TurnID:    turnID,
		Iteration: 1,
		Timestamp: time.Now().UTC(),
	}
	projector := newTurnProjector(emit, meta, CurrentRolloutConfig())
	emitEvent(emit, events.Meta, meta, map[string]any{
		"session_id": sessionID,
		"sessionId":  sessionID,
		"trace_id":   meta.TraceID,
		"traceId":    meta.TraceID,
		"turn_id":    turnID,
		"resumed":    true,
		"createdAt":  meta.Timestamp.Format(time.RFC3339),
	})
	projector.Start("execute")
	projector.SetState(firstNonEmpty(res.Status, "error"), "execute")
	if strings.TrimSpace(res.Message) != "" {
		emitDeltaChunks(emit, projector, meta, res.Message)
	}
	if res.Status == "missing" || res.Status == "unavailable" {
		emitEvent(emit, events.Error, meta, map[string]any{
			"message": res.Message,
			"stage":   "execute",
		})
	}
	projector.Done(firstNonEmpty(res.Status, "error"), "execute")
	emitEvent(emit, events.Done, meta, map[string]any{
		"status":  res.Status,
		"message": res.Message,
	})
}

// MetricsSnapshot 返回当前指标快照。
// 用于监控和统计 AI 编排器的运行状态。
func (o *Orchestrator) MetricsSnapshot() AIMetricsSnapshot {
	if o == nil || o.metrics == nil {
		return AIMetricsSnapshot{}
	}
	return o.metrics.Snapshot()
}

// planAndReply 执行规划和回复生成。
//
// 参数:
//   - ctx: 上下文
//   - message: 用户原始消息
//   - rewritten: Rewrite 阶段的输出
//   - runtimeCtx: 运行时上下文 (场景、路由、资源等)
//   - meta: 事件元数据
//   - emit: 流式事件回调
//   - sessionID: 会话 ID
//
// 返回: 回复文本和可能的错误
//
// 决策类型:
//   - clarify: 需要用户澄清，返回澄清消息
//   - direct_reply: 直接回复，无需执行计划
//   - plan: 生成执行计划，调用 executor 执行
func (o *Orchestrator) planAndReply(ctx context.Context, message string, rewritten rewrite.Output, runtimeCtx RuntimeContext, meta events.EventMeta, emit StreamEmitter, projector *turnProjector, sessionID string) (string, error) {
	if o.planner == nil {
		reply := plannerFailureMessage(&planner.PlanningError{
			Code:              "planner_runner_unavailable",
			UserVisibleReason: "AI 规划模块当前不可用，请稍后重试或手动在页面中执行操作。",
		})
		emitEvent(emit, events.PlannerState, meta, map[string]any{
			"status":               "error",
			"user_visible_summary": reply,
		})
		projector.SetState("error", "plan")
		emitStageDelta(emit, projector, meta, "plan", "error", reply, "", "")
		emitEvent(emit, events.Error, meta, map[string]any{
			"message":    reply,
			"error_code": "planner_runner_unavailable",
			"stage":      "plan",
		})
		projector.ExecutionEvent(string(events.Error), meta, map[string]any{
			"message":    reply,
			"error_code": "planner_runner_unavailable",
			"stage":      "plan",
		})
		emitDeltaChunks(emit, projector, meta, reply)
		return reply, nil
	}

	projector.SetState("streaming", "plan")
	emitEvent(emit, events.PlannerState, meta, map[string]any{
		"status":               "planning",
		"user_visible_summary": "正在根据 Rewrite 结果整理执行计划。",
	})
	replanAttempts := 0
	decision, err := o.planner.PlanStream(ctx, planner.Input{
		Message: message,
		Rewrite: rewritten,
		OnReplan: func(info planner.ReplanAttempt) {
			replanAttempts++
			if o.metrics != nil {
				o.metrics.RecordPlannerReplanAttempt()
			}
			emitEvent(emit, events.ReplanStarted, meta, map[string]any{
				"reason":              info.Reason,
				"attempt":             info.Attempt,
				"max_attempts":        info.MaxAttempts,
				"previous_error_code": info.PreviousErrorCode,
			})
			emitEvent(emit, events.PlannerState, meta, map[string]any{
				"status":               "replanning",
				"user_visible_summary": "规划结果不符合要求，正在自动修复。",
			})
			emitEvent(emit, events.StageDelta, meta, map[string]any{
				"stage":         "plan",
				"status":        "loading",
				"content_chunk": "规划结果不符合要求，正在自动修复。",
				"replace":       true,
			})
			if projector != nil {
				projector.StageDelta("plan", "loading", "规划结果不符合要求，正在自动修复。", "", "")
			}
		},
	}, func(chunk string) {
		emitStageDelta(emit, projector, meta, "plan", "loading", chunk, "", "")
	})
	if err != nil {
		if o.metrics != nil && replanAttempts > 0 {
			o.metrics.RecordPlannerReplanOutcome(false, true)
		}
		reply := plannerFailureMessage(err)
		projector.SetState("error", "plan")
		emitStageDelta(emit, projector, meta, "plan", "error", reply, "", "")
		emitEvent(emit, events.Error, meta, map[string]any{
			"message":    reply,
			"error_code": plannerFailureCode(err),
			"stage":      "plan",
		})
		projector.ExecutionEvent(string(events.Error), meta, map[string]any{
			"message":    reply,
			"error_code": plannerFailureCode(err),
			"stage":      "plan",
		})
		emitDeltaChunks(emit, projector, meta, reply)
		return reply, nil
	}
	if o.metrics != nil && replanAttempts > 0 {
		o.metrics.RecordPlannerReplanOutcome(true, false)
	}
	if o.metrics != nil {
		o.metrics.RecordPlanner(decision)
	}
	switch decision.Type {
	case planner.DecisionClarify:
		projector.SetState("waiting_user", "plan")
		emitEvent(emit, events.ClarifyRequired, meta, map[string]any{
			"title":      "需要你补充信息",
			"message":    decision.Message,
			"candidates": decision.Candidates,
			"kind":       "clarify",
		})
		emitStageDelta(emit, projector, meta, "plan", "error", decision.Message, "", "")
		emitDeltaChunks(emit, projector, meta, decision.Message)
		return decision.Message, nil
	case planner.DecisionDirectReply:
		projector.SetState("streaming", "summary")
		emitStageDelta(emit, projector, meta, "plan", "success", firstNonEmpty(decision.Message, decision.Narrative), "", "")
		emitDeltaChunks(emit, projector, meta, decision.Message)
		projector.CloseText("text:final", "completed")
		return decision.Message, nil
	case planner.DecisionPlan:
		if decision.Plan != nil {
			meta.PlanID = decision.Plan.PlanID
			emitEvent(emit, events.PlanCreated, meta, map[string]any{
				"plan":                 decision.Plan,
				"user_visible_summary": decision.Narrative,
			})
			projector.Plan(firstNonEmpty(decision.Narrative, decision.Plan.Narrative), map[string]any{
				"plan":                 decision.Plan,
				"user_visible_summary": decision.Narrative,
			})
			projector.SetState("streaming", "execute")
			emitStageDelta(emit, projector, meta, "plan", "success", firstNonEmpty(decision.Narrative, decision.Plan.Narrative), "", "")
			if o.executor != nil {
				if gatedStepID := firstGateablePlanStep(*decision.Plan); gatedStepID != "" {
					gated, gateErr := o.executor.PrepareApprovalGate(ctx, executor.Request{
						TraceID:   meta.TraceID,
						SessionID: sessionID,
						Message:   message,
						Plan:      *decision.Plan,
						RuntimeContext: runtime.ContextSnapshot{
							Scene:       runtimeCtx.Scene,
							Route:       runtimeCtx.Route,
							ProjectID:   runtimeCtx.ProjectID,
							CurrentPage: runtimeCtx.CurrentPage,
							ResourceIDs: selectedResourceIDs(runtimeCtx.SelectedResources),
						},
						EventMeta: meta,
						EmitEvent: func(name string, eventMeta events.EventMeta, data map[string]any) bool {
							projector.ExecutionEvent(name, eventMeta, data)
							emitEvent(emit, events.Name(name), eventMeta, data)
							emitExecuteStageDelta(emit, projector, eventMeta, name, data)
							return true
						},
					}, gatedStepID)
					if gateErr != nil {
						return "", gateErr
					}
					if gated != nil && gated.Approval() != nil && strings.TrimSpace(gated.Approval().Status) == "pending" {
						projector.SetState("waiting_user", "execute")
						waitingSummary := "此步骤需要你确认后继续执行。"
						emitDeltaChunks(emit, projector, meta, waitingSummary)
						return waitingSummary, nil
					}
				}
				executed, execErr := o.executor.Run(ctx, executor.Request{
					TraceID:   meta.TraceID,
					SessionID: sessionID,
					Message:   message,
					Plan:      *decision.Plan,
					RuntimeContext: runtime.ContextSnapshot{
						Scene:       runtimeCtx.Scene,
						Route:       runtimeCtx.Route,
						ProjectID:   runtimeCtx.ProjectID,
						CurrentPage: runtimeCtx.CurrentPage,
						ResourceIDs: selectedResourceIDs(runtimeCtx.SelectedResources),
					},
					EventMeta: meta,
					EmitEvent: func(name string, eventMeta events.EventMeta, data map[string]any) bool {
						projector.ExecutionEvent(name, eventMeta, data)
						emitEvent(emit, events.Name(name), eventMeta, data)
						emitExecuteStageDelta(emit, projector, eventMeta, name, data)
						return true
					},
				})
				if execErr == nil && executed != nil {
					if approval := executed.Approval(); approval != nil && strings.TrimSpace(approval.Status) == "pending" {
						projector.SetState("waiting_user", "execute")
						waitingSummary := "此步骤需要你确认后继续执行。"
						emitDeltaChunks(emit, projector, meta, waitingSummary)
						return waitingSummary, nil
					}
					summaryText, summaryErr := o.emitExecutionSummary(ctx, message, decision.Plan, executed, emit, projector, meta)
					if body := o.renderAndEmitFinalAnswer(decision.Plan, executed, summaryText, emit, meta); body != "" {
						return body, nil
					}
					_ = summaryErr
				}
			}
		}
	}
	return "", nil
}

// renderAndEmitFinalAnswer 渲染并发送最终答案。
// 最终正文直接透传 summary 阶段产出的完整结果。
func (o *Orchestrator) renderAndEmitFinalAnswer(plan *planner.ExecutionPlan, result *executor.Result, summaryText string, emit StreamEmitter, meta events.EventMeta) string {
	_ = plan
	_ = result
	_ = emit
	_ = meta
	return strings.TrimSpace(summaryText)
}

func (o *Orchestrator) emitExecutionSummary(
	ctx context.Context,
	message string,
	plan *planner.ExecutionPlan,
	executed *executor.Result,
	emit StreamEmitter,
	projector *turnProjector,
	meta events.EventMeta,
) (string, error) {
	projector.SetState("streaming", "summary")
	emitStageDelta(emit, projector, meta, "summary", "loading", "正在思考并整理最终回答。", "", "")
	summaryText, summaryErr := o.summarizeExecution(ctx, message, plan, executed, func(chunk string) {
		projector.TextDelta("thinking:summary", "thinking", "summary", chunk)
		emitEvent(emit, events.ThinkingDelta, meta, map[string]any{
			"content_chunk": chunk,
			"contentChunk":  chunk,
		})
	}, func(chunk string) {
		projector.TextDelta("text:final", "text", "summary", chunk)
		emitEvent(emit, events.Delta, meta, map[string]any{
			"content_chunk": chunk,
			"contentChunk":  chunk,
		})
	})
	projector.CloseText("thinking:summary", summaryStageStatus(summaryErr))
	projector.CloseText("text:final", summaryStageStatus(summaryErr))
	emitEvent(emit, events.Summary, meta, map[string]any{
		"status":  summaryStageStatus(summaryErr),
		"summary": summaryText,
	})
	return summaryText, summaryErr
}

func firstGateablePlanStep(plan planner.ExecutionPlan) string {
	for _, step := range plan.Steps {
		if len(step.DependsOn) != 0 {
			continue
		}
		if executor.StepRequiresApproval(step.Mode, step.Risk) {
			return strings.TrimSpace(step.StepID)
		}
	}
	return ""
}

func planFromExecutionState(state runtime.ExecutionState) *planner.ExecutionPlan {
	if strings.TrimSpace(state.Plan.PlanID) != "" {
		planCopy := state.Plan
		return &planCopy
	}
	if strings.TrimSpace(state.PlanID) == "" {
		return nil
	}
	steps := make([]planner.PlanStep, 0, len(state.Steps))
	for _, step := range state.Steps {
		steps = append(steps, planner.PlanStep{
			StepID:    step.StepID,
			Title:     step.Title,
			Expert:    step.Expert,
			Intent:    step.Intent,
			Task:      step.Task,
			Input:     step.Input,
			DependsOn: append([]string(nil), step.DependsOn...),
			Mode:      step.Mode,
			Risk:      step.Risk,
		})
	}
	return &planner.ExecutionPlan{
		PlanID: state.PlanID,
		Goal:   state.Plan.Goal,
		Steps:  steps,
	}
}

// sessionMessages 获取会话的所有消息列表。
// 用于在完成时返回完整的会话状态给前端。
func (o *Orchestrator) sessionMessages(ctx context.Context, sessionID string) []map[string]any {
	if o.sessions == nil {
		return []map[string]any{}
	}
	snapshot, err := o.sessions.Load(ctx, sessionID)
	if err != nil || snapshot == nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(snapshot.Messages))
	for i, msg := range snapshot.Messages {
		out = append(out, map[string]any{
			"id":        fmt.Sprintf("%s-%d", sessionID, i+1),
			"role":      msg.Role,
			"content":   msg.Content,
			"timestamp": msg.Timestamp.Format(time.RFC3339),
		})
	}
	return out
}

// emitEvent 发送流式事件到客户端。
// 如果 emit 为 nil 则什么都不做。
func emitEvent(emit StreamEmitter, name events.Name, meta events.EventMeta, data map[string]any) {
	if emit == nil {
		return
	}
	payload := cloneMap(data)
	if payload == nil {
		payload = map[string]any{}
	}
	if meta.TurnID != "" {
		payload["turn_id"] = meta.TurnID
	}
	if meta.BlockID != "" {
		payload["block_id"] = meta.BlockID
	}
	emit(StreamEvent{
		Type:     name,
		Audience: events.AudienceUser,
		Meta:     meta.WithDefaults(),
		Data:     payload,
	})
}

// startHeartbeat 启动心跳 goroutine，定期发送心跳事件。
// 返回停止心跳的函数，应在 Run 结束时调用。
// 心跳用于保持长连接活跃，防止连接超时断开。
func (o *Orchestrator) startHeartbeat(ctx context.Context, emit StreamEmitter, meta events.EventMeta) func() {
	if emit == nil || o == nil || o.heartbeatInterval <= 0 {
		return func() {}
	}
	stop := make(chan struct{})
	ticker := time.NewTicker(o.heartbeatInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case now := <-ticker.C:
				emitEvent(emit, events.Heartbeat, meta, map[string]any{
					"status":    "streaming",
					"timestamp": now.UTC().Format(time.RFC3339),
				})
			}
		}
	}()
	return func() {
		close(stop)
	}
}

// emitStageDelta 发送阶段增量事件。
// 用于向前端推送各阶段 (rewrite/plan/execute/summary) 的实时进度。
//
// 参数:
//   - stage: 阶段名称 (rewrite/plan/execute/summary)
//   - status: 状态 (loading/success/error)
//   - chunk: 内容片段
//   - stepID: 步骤 ID (执行阶段使用)
//   - expert: 专家名称 (执行阶段使用)
func emitStageDelta(emit StreamEmitter, projector *turnProjector, meta events.EventMeta, stage, status, chunk, stepID, expert string) {
	chunk = strings.TrimSpace(chunk)
	if emit == nil || stage == "" || chunk == "" {
		return
	}
	payload := map[string]any{
		"stage":         stage,
		"status":        status,
		"content_chunk": chunk,
	}
	if stepID != "" {
		payload["step_id"] = stepID
	}
	if expert != "" {
		payload["expert"] = expert
	}
	emitEvent(emit, events.StageDelta, meta, payload)
	if projector != nil {
		projector.StageDelta(stage, status, chunk, stepID, expert)
	}
}

// emitExecuteStageDelta 将执行阶段的事件转换为阶段增量事件。
// 处理 StepUpdate、ToolCall、ToolResult 三种事件类型。
func emitExecuteStageDelta(emit StreamEmitter, projector *turnProjector, meta events.EventMeta, name string, data map[string]any) {
	switch name {
	case string(events.StepUpdate):
		emitStageDelta(
			emit,
			projector,
			meta,
			"execute",
			stageStatusFromValue(data["status"]),
			firstNonEmpty(stringValue(data["title"]), stringValue(data["user_visible_summary"])),
			stringValue(data["step_id"]),
			stringValue(data["expert"]),
		)
	case string(events.ToolCall):
		emitStageDelta(
			emit,
			projector,
			meta,
			"execute",
			"loading",
			firstNonEmpty(stringValue(data["summary"]), stringValue(data["tool_name"])),
			stringValue(data["step_id"]),
			firstNonEmpty(stringValue(data["expert"]), stringValue(data["tool_name"])),
		)
	case string(events.ToolResult):
		status := "success"
		if strings.TrimSpace(stringValue(data["error"])) != "" || strings.EqualFold(stringValue(data["status"]), "error") {
			status = "error"
		}
		emitStageDelta(
			emit,
			projector,
			meta,
			"execute",
			status,
			firstNonEmpty(stringValue(data["summary"]), stringValue(data["error"]), stringValue(data["tool_name"])),
			stringValue(data["step_id"]),
			firstNonEmpty(stringValue(data["expert"]), stringValue(data["tool_name"])),
		)
	}
}

// stageStatusFromValue 将步骤状态值转换为阶段状态。
// completed/success -> success
// failed/error/blocked/cancelled/rejected -> error
// 其他 -> loading
func stageStatusFromValue(value any) string {
	switch strings.TrimSpace(stringValue(value)) {
	case "completed", "success":
		return "success"
	case "failed", "error", "blocked", "cancelled", "rejected":
		return "error"
	default:
		return "loading"
	}
}

// emitDeltaChunks 将内容分块发送。
// 用于流式输出大段文本，每块 24 个字符。
func emitDeltaChunks(emit StreamEmitter, projector *turnProjector, meta events.EventMeta, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}
	runes := []rune(content)
	const chunkSize = 24
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := string(runes[start:end])
		if projector != nil {
			projector.TextDelta("text:final", "text", "summary", chunk)
		}
		emitEvent(emit, events.Delta, meta, map[string]any{
			"content_chunk": chunk,
			"contentChunk":  chunk,
		})
	}
	if projector != nil {
		projector.CloseText("text:final", "completed")
	}
}

// rewriteFailureMessage 根据错误类型生成用户可见的失败消息。
func rewriteFailureMessage(err error) string {
	var unavailable *rewrite.ModelUnavailableError
	if errors.As(err, &unavailable) {
		return unavailable.UserVisibleMessage()
	}
	return availability.UnavailableMessage(availability.LayerRewrite)
}

// rewriteFailureCode 从错误中提取错误码，用于前端错误处理。
func rewriteFailureCode(err error) string {
	var unavailable *rewrite.ModelUnavailableError
	if errors.As(err, &unavailable) && strings.TrimSpace(unavailable.Code) != "" {
		return strings.TrimSpace(unavailable.Code)
	}
	return "rewrite_unavailable"
}

// plannerFailureMessage 根据规划错误类型生成用户可见的失败消息。
func plannerFailureMessage(err error) string {
	var unavailable *planner.PlanningError
	if errors.As(err, &unavailable) {
		return unavailable.UserVisibleMessage()
	}
	return availability.UnavailableMessage(availability.LayerPlanner)
}

// plannerFailureCode 从规划错误中提取错误码。
func plannerFailureCode(err error) string {
	var unavailable *planner.PlanningError
	if errors.As(err, &unavailable) && strings.TrimSpace(unavailable.Code) != "" {
		return strings.TrimSpace(unavailable.Code)
	}
	return "planner_unavailable"
}

// eventsJSON 将对象序列化为 JSON 字符串。
func eventsJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// deriveSessionTitle 从用户消息中派生会话标题。
// 取消息的前 24 个字符作为标题。
func deriveSessionTitle(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "新对话"
	}
	runes := []rune(message)
	if len(runes) > 24 {
		return string(runes[:24])
	}
	return message
}

// firstNonEmpty 返回第一个非空字符串。
// 用于从多个候选值中选择有效值。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// stringValue 安全地将任意值转换为字符串。
func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func buildResumeResult(req ResumeRequest, result *executor.Result) *ResumeResult {
	if result == nil {
		return &ResumeResult{
			Resumed:   false,
			SessionID: strings.TrimSpace(req.SessionID),
			PlanID:    strings.TrimSpace(req.PlanID),
			StepID:    firstNonEmpty(req.StepID, req.Target),
			Status:    "missing",
			Message:   "execution state not found",
		}
	}
	state := result.State
	if state.SessionID == "" {
		return &ResumeResult{
			Resumed:   false,
			SessionID: strings.TrimSpace(req.SessionID),
			PlanID:    strings.TrimSpace(req.PlanID),
			StepID:    firstNonEmpty(req.StepID, req.Target),
			Status:    "noop",
			Message:   "no pending approval for this session",
		}
	}
	pendingStepID := ""
	pendingPlanID := ""
	if state.PendingApproval != nil {
		pendingStepID = state.PendingApproval.StepID
		pendingPlanID = state.PendingApproval.PlanID
	}
	stepID := firstNonEmpty(req.StepID, req.Target, pendingStepID)
	planID := firstNonEmpty(req.PlanID, state.PlanID, pendingPlanID)
	status := firstNonEmpty(state.Phase)
	if approval := result.Approval(); approval != nil && approval.Status != "" {
		status = approval.Status
	}
	return &ResumeResult{
		Resumed:     req.Approved,
		Interrupted: !req.Approved,
		SessionID:   state.SessionID,
		PlanID:      planID,
		StepID:      stepID,
		TurnID:      state.TurnID,
		Status:      status,
		Message:     resumeStatusMessage(status, req.Approved),
	}
}

func resultPhase(result *executor.Result) string {
	if result == nil {
		return ""
	}
	return strings.TrimSpace(result.State.Phase)
}

func turnStateStatus(status runtime.ExecutionStatus) string {
	switch status {
	case runtime.ExecutionStatusWaitingApproval:
		return "waiting_user"
	case runtime.ExecutionStatusCompleted:
		return "completed"
	case runtime.ExecutionStatusFailed:
		return "error"
	default:
		return "streaming"
	}
}

// resumeStatusMessage 根据恢复状态生成用户可见的消息。
func resumeStatusMessage(status string, approved bool) string {
	switch strings.TrimSpace(status) {
	case "approved", "approval_granted":
		return "审批已通过，待审批步骤会继续执行。"
	case "rejected":
		return "审批已拒绝，待审批步骤不会执行，相关下游步骤已被取消或阻断。"
	case "cancelled":
		return "当前执行已取消，后续步骤不会继续执行。"
	case "idempotent":
		if approved {
			return "重复的审批通过请求已忽略，系统不会重复执行该步骤。"
		}
		return "重复的审批拒绝请求已忽略，系统不会再次取消该步骤。"
	case "noop":
		return "当前没有可恢复的待审批步骤。"
	case "missing":
		return "未找到对应的执行状态，无法恢复该步骤。"
	case "unavailable":
		return "当前恢复能力不可用。"
	default:
		if approved {
			return "审批已记录，执行链路可继续恢复。"
		}
		return "审批已拒绝，当前待审批步骤不会继续执行。"
	}
}

// summarizeExecution 调用 Summarizer 阶段生成执行总结。
//
// 参数:
//   - ctx: 上下文
//   - message: 用户原始消息
//   - plan: 执行计划
//   - result: 执行结果
//   - onDelta: 流式输出回调
//
// 返回: 总结输出和可能的错误
func (o *Orchestrator) summarizeExecution(
	ctx context.Context,
	message string,
	plan *planner.ExecutionPlan,
	result *executor.Result,
	onThinkingDelta func(string),
	onAnswerDelta func(string),
) (string, error) {
	if o.summarizer == nil {
		fallback := summarizerUnavailableSummary(nil)
		if onAnswerDelta != nil {
			onAnswerDelta(fallback)
		}
		return fallback, &summarizer.UnavailableError{
			Code:              "summarizer_runner_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerSummarizer),
		}
	}
	out, err := o.summarizer.SummarizeStream(ctx, summarizer.Input{
		Message: message,
		Plan:    plan,
		State:   result.State,
		Steps:   result.Steps,
	}, onThinkingDelta, onAnswerDelta)
	if err != nil {
		fallback := summarizerUnavailableSummary(err)
		if onAnswerDelta != nil {
			onAnswerDelta(fallback)
		}
		return fallback, err
	}
	return out, nil
}

// summarizerUnavailableSummary 当 Summarizer 不可用时生成降级总结。
// 提示用户查看原始执行证据。
func summarizerUnavailableSummary(err error) string {
	message := availability.UnavailableMessage(availability.LayerSummarizer)
	var unavailable *summarizer.UnavailableError
	if errors.As(err, &unavailable) {
		message = unavailable.UserVisibleMessage()
	}
	return strings.TrimSpace(message)
}

// summaryStageStatus 根据错误判断总结阶段状态。
func summaryStageStatus(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}

// toRewriteResources 将 SelectedResource 列表转换为 rewrite 模块的资源格式。
func toRewriteResources(items []SelectedResource) []rewrite.SelectedResource {
	out := make([]rewrite.SelectedResource, 0, len(items))
	for _, item := range items {
		out = append(out, rewrite.SelectedResource{
			Type: item.Type,
			ID:   item.ID,
			Name: item.Name,
		})
	}
	return out
}

// selectedResourceIDs 从 SelectedResource 列表中提取非空的 ID 列表。
func selectedResourceIDs(items []SelectedResource) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		out = append(out, item.ID)
	}
	return out
}
