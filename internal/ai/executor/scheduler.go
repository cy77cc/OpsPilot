// Package executor 实现 AI 编排的执行阶段。
//
// 本文件实现步骤调度器，负责管理步骤状态转换和依赖处理。
// 调度器根据步骤依赖关系和执行状态决定执行顺序。
package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

// advanceScheduler 推进执行调度，处理步骤状态转换。
// 循环处理 Pending -> Ready -> Running -> Completed/Failed 状态转换。
func advanceScheduler(ctx context.Context, store *runtime.ExecutionStore, state *runtime.ExecutionState, req Request, stepRunner StepRunner) (*Result, error) {
	if state == nil {
		return nil, fmt.Errorf("execution state is required")
	}
	results := make([]StepResult, 0, len(state.Steps))
	progress := true
	for progress {
		progress = false
		for stepID, step := range state.Steps {
			switch step.Status {
			case runtime.StepPending:
				if depsSatisfied(state, stepID) {
					if err := transitionStep(state, stepID, runtime.StepReady, "dependencies satisfied"); err != nil {
						return nil, err
					}
					results = append(results, snapshotResult(state.Steps[stepID]))
					progress = true
				} else if depsBlocked(state, stepID) {
					if err := transitionStep(state, stepID, runtime.StepBlocked, "upstream dependency failed"); err != nil {
						return nil, err
					}
					results = append(results, snapshotResult(state.Steps[stepID]))
					progress = true
				}
			case runtime.StepReady:
				if err := transitionStep(state, stepID, runtime.StepRunning, "step entered executor runtime"); err != nil {
					return nil, err
				}
				runningStep := state.Steps[stepID]
				results = append(results, snapshotResult(runningStep))
				emitStepUpdate(req, state, runningStep)
				progress = true

				step = state.Steps[stepID]
				if needsApproval(step) {
					policy := riskPolicy(step.Mode, step.Risk)
					pending := &runtime.PendingApproval{
						PlanID:      state.PlanID,
						StepID:      stepID,
						ApprovalKey: fmt.Sprintf("%s:%s", state.PlanID, stepID),
						Status:      "pending",
						Title:       step.Title,
						Mode:        step.Mode,
						Risk:        step.Risk,
						Summary:     step.UserVisibleSummary,
						RequestedAt: time.Now().UTC(),
					}
					state.PendingApproval = pending
					state.Status = runtime.ExecutionStatusWaitingApproval
					state.Phase = fmt.Sprintf("waiting_approval:%s", approvalStageName(policy))
					if err := transitionStep(state, stepID, runtime.StepWaitingApproval, "step requires approval before execution"); err != nil {
						return nil, err
					}
					waitingStep := state.Steps[stepID]
					results = append(results, snapshotResult(waitingStep))
					emitStepUpdate(req, state, waitingStep)
					emitApprovalRequired(req, state, pending, waitingStep)
					if store != nil {
						if err := store.Save(ctx, *state); err != nil {
							return nil, err
						}
					}
					return &Result{State: *state, Steps: results}, nil
				}

				executed, err := executeStep(ctx, state, stepID, req, stepRunner)
				if err != nil {
					return nil, err
				}
				results = append(results, executed...)
				progress = len(executed) > 0
			}
		}
	}

	if hasFailedOrBlocked(state) {
		state.Status = runtime.ExecutionStatusFailed
		state.Phase = "executor_blocked"
	} else {
		state.Status = runtime.ExecutionStatusCompleted
		state.Phase = "executor_completed"
	}
	if store != nil {
		if err := store.Save(ctx, *state); err != nil {
			return nil, err
		}
	}
	return &Result{State: *state, Steps: results}, nil
}

// advanceAfterApproval 处理审批后的执行恢复。
// 根据审批结果继续执行或取消步骤。
func advanceAfterApproval(ctx context.Context, store *runtime.ExecutionStore, state *runtime.ExecutionState, req ResumeRequest, stepRunner StepRunner) (*Result, error) {
	if state == nil {
		return nil, fmt.Errorf("execution state is required")
	}
	if state.PendingApproval == nil {
		return &Result{State: *state}, nil
	}
	stepID := strings.TrimSpace(req.StepID)
	if stepID == "" {
		stepID = state.PendingApproval.StepID
	}
	if stepID == "" {
		return nil, fmt.Errorf("step_id is required")
	}
	if strings.TrimSpace(req.PlanID) != "" && strings.TrimSpace(req.PlanID) != state.PlanID {
		return nil, fmt.Errorf("plan_id mismatch")
	}

	decision := ApprovalDecision{
		PlanID:      state.PlanID,
		StepID:      stepID,
		Approved:    req.Approved,
		Reason:      strings.TrimSpace(req.Reason),
		Idempotency: runtime.ApprovalDecisionHash(state.PlanID, stepID, req.Approved),
	}
	if state.PendingApproval.DecisionHash == decision.Idempotency {
		return &Result{State: *state}, nil
	}
	now := time.Now().UTC()
	decision.Status = "rejected"
	if req.Approved {
		decision.Status = "approved"
	}
	state.PendingApproval.DecisionHash = decision.Idempotency
	state.PendingApproval.Approved = &decision.Approved
	state.PendingApproval.Reason = decision.Reason
	state.PendingApproval.ResolvedAt = now
	state.PendingApproval.Status = decision.Status

	if req.Approved {
		step := state.Steps[stepID]
		step.ApprovalSatisfied = true
		state.Steps[stepID] = step
		if err := transitionStep(state, stepID, runtime.StepReady, "审批已通过，步骤已返回待执行队列。"); err != nil {
			return nil, err
		}
		state.Status = runtime.ExecutionStatusRunning
		state.Phase = "approval_granted"
	} else {
		if err := transitionStep(state, stepID, runtime.StepCancelled, "审批已拒绝，当前步骤不会执行。"); err != nil {
			return nil, err
		}
		state.Status = runtime.ExecutionStatusFailed
		state.Phase = "rejected"
		markDependentsBlocked(state, stepID)
	}
	return advanceScheduler(ctx, store, state, Request{
		TraceID:        state.TraceID,
		SessionID:      state.SessionID,
		Message:        state.Message,
		Plan:           planner.ExecutionPlan{PlanID: state.PlanID},
		RuntimeContext: state.RuntimeContext,
		EventMeta: EventMeta{
			SessionID: state.SessionID,
			TraceID:   state.TraceID,
			PlanID:    state.PlanID,
		},
	}, stepRunner)
}

// executeStep 执行单个步骤，调用专家 Agent。
func executeStep(ctx context.Context, state *runtime.ExecutionState, stepID string, req Request, stepRunner StepRunner) ([]StepResult, error) {
	step := state.Steps[stepID]
	if stepRunner == nil {
		return markStepFailure(req, state, stepID, "expert_tool_stream_failed", "expert runner is not configured", "专家执行链路未正确初始化，当前步骤无法执行。")
	}
	emitToolCall(req, state, step, step.Expert, firstNonEmpty(step.Task, step.Title, "专家开始执行步骤"))

	planStep := planner.PlanStep{
		StepID:    step.StepID,
		Title:     step.Title,
		Expert:    step.Expert,
		Intent:    step.Intent,
		Task:      step.Task,
		Input:     step.Input,
		DependsOn: append([]string(nil), step.DependsOn...),
		Mode:      step.Mode,
		Risk:      step.Risk,
	}
	for _, candidate := range req.Plan.Steps {
		if candidate.StepID == stepID {
			planStep = candidate
			break
		}
	}

	result, err := stepRunner.RunStep(ctx, req, planStep)
	if err != nil {
		code := "expert_tool_execution_failed"
		userSummary := "专家执行失败，需要人工跟进。"
		errMessage := err.Error()
		if execErr, ok := err.(*ExecutionError); ok {
			code = firstNonEmpty(execErr.Code, code)
			if strings.TrimSpace(execErr.UserSummary) != "" {
				userSummary = strings.TrimSpace(execErr.UserSummary)
			}
			errMessage = firstNonEmpty(execErr.Message, errMessage)
			err = errors.New(strings.TrimSpace(execErr.Message))
		} else if isProviderTimeoutError(errMessage) {
			code = "expert_tool_stream_failed"
			userSummary = fmt.Sprintf("专家 %s 调用模型超时，请稍后重试。", strings.TrimSpace(step.Expert))
		} else if summary, field, ok := summarizeMissingPrerequisite(errMessage); ok {
			code = "missing_execution_prerequisite"
			userSummary = fmt.Sprintf("%s。缺少前置上下文：%s", summary, field)
		}
		compactMessage := compactToolError(errMessage)
		emitToolResult(req, state, step, step.Expert, "error", userSummary, compactMessage, map[string]any{"ok": false})
		return markStepFailure(req, state, stepID, code, compactMessage, userSummary)
	}

	summary := firstNonEmpty(result.Summary, "step executed by expert agent")
	if err := transitionStep(state, stepID, runtime.StepCompleted, summary); err != nil {
		return nil, err
	}
	result.StepID = stepID
	result.Status = runtime.StepCompleted
	result.Summary = summary
	result.UpdatedAt = state.Steps[stepID].UpdatedAt
	emitToolResult(req, state, state.Steps[stepID], step.Expert, "success", summary, "", map[string]any{"ok": true})
	emitStepUpdate(req, state, state.Steps[stepID])
	return []StepResult{result}, nil
}

// markStepFailure 标记步骤失败，根据策略决定是否自动重试。
func markStepFailure(req Request, state *runtime.ExecutionState, stepID, code, message, userSummary string) ([]StepResult, error) {
	step, ok := state.Steps[stepID]
	if !ok {
		return nil, fmt.Errorf("step %s not found", stepID)
	}
	step.ErrorCode = strings.TrimSpace(code)
	step.ErrorMessage = strings.TrimSpace(message)
	summary := firstNonEmpty(userSummary, "专家执行失败，需要人工跟进。")
	if shouldAutoRetry(step) {
		step.Status = runtime.StepReady
		step.UserVisibleSummary = summary
		step.UpdatedAt = time.Now().UTC()
		state.Steps[stepID] = step
		state.UpdatedAt = step.UpdatedAt
		emitStepUpdate(req, state, step)
		return []StepResult{snapshotResult(step)}, nil
	}
	step.Status = runtime.StepFailed
	step.UserVisibleSummary = summary
	step.UpdatedAt = time.Now().UTC()
	state.Steps[stepID] = step
	state.Status = runtime.ExecutionStatusFailed
	state.Phase = "executor_failed"
	state.UpdatedAt = step.UpdatedAt
	markDependentsBlocked(state, stepID)
	emitStepUpdate(req, state, step)
	return []StepResult{snapshotResult(step)}, nil
}

// depsSatisfied 检查步骤的所有依赖是否已满足。
func depsSatisfied(state *runtime.ExecutionState, stepID string) bool {
	deps := stepDependencies(state, stepID)
	if len(deps) == 0 {
		return true
	}
	for _, dep := range deps {
		if state.Steps[dep].Status != runtime.StepCompleted {
			return false
		}
	}
	return true
}

// depsBlocked 检查步骤是否有被阻塞的依赖。
func depsBlocked(state *runtime.ExecutionState, stepID string) bool {
	for _, dep := range stepDependencies(state, stepID) {
		status := state.Steps[dep].Status
		if status == runtime.StepFailed || status == runtime.StepBlocked || status == runtime.StepCancelled {
			return true
		}
	}
	return false
}

// stepDependencies 获取步骤的依赖列表。
func stepDependencies(state *runtime.ExecutionState, stepID string) []string {
	if state == nil || state.Steps == nil {
		return nil
	}
	raw, ok := state.Steps[stepID]
	if !ok {
		return nil
	}
	if raw.DependsOn != nil {
		return raw.DependsOn
	}
	return nil
}

// needsApproval 检查步骤是否需要审批。
func needsApproval(step runtime.StepState) bool {
	if step.ApprovalSatisfied {
		return false
	}
	return riskPolicy(step.Mode, step.Risk).RequiresApproval
}

// shouldAutoRetry 检查步骤是否应该自动重试。
func shouldAutoRetry(step runtime.StepState) bool {
	policy := riskPolicy(step.Mode, step.Risk)
	if !policy.AutoRetry {
		return false
	}
	return step.Attempts < step.MaxAttempts
}

// approvalStageName 根据风险策略返回审批阶段名称。
func approvalStageName(policy RiskPolicy) string {
	if !policy.RequiresApproval {
		return "none"
	}
	if policy.MaxAttempts <= 1 {
		return "strict"
	}
	return "guarded"
}

// hasFailedOrBlocked 检查执行是否有失败或被阻塞的步骤。
func hasFailedOrBlocked(state *runtime.ExecutionState) bool {
	for _, step := range state.Steps {
		if step.Status == runtime.StepFailed || step.Status == runtime.StepBlocked || step.Status == runtime.StepCancelled {
			return true
		}
	}
	return false
}

// markDependentsBlocked 将依赖失败步骤的所有后续步骤标记为阻塞。
func markDependentsBlocked(state *runtime.ExecutionState, failedStepID string) {
	for stepID := range state.Steps {
		for _, dep := range stepDependencies(state, stepID) {
			if dep == failedStepID {
				_ = transitionStep(state, stepID, runtime.StepBlocked, "上游步骤已取消，当前步骤不会继续执行。")
				break
			}
		}
	}
}

// transitionStep 执行步骤状态转换。
func transitionStep(state *runtime.ExecutionState, stepID string, target runtime.StepStatus, summary string) error {
	if state == nil {
		return fmt.Errorf("execution state is required")
	}
	step, ok := state.Steps[stepID]
	if !ok {
		return fmt.Errorf("step %s not found", stepID)
	}
	if !validTransition(step.Status, target) {
		return fmt.Errorf("invalid step transition: %s -> %s", step.Status, target)
	}
	step.Status = target
	if target == runtime.StepRunning {
		step.Attempts++
		if step.MaxAttempts == 0 {
			step.MaxAttempts = riskPolicy(step.Mode, step.Risk).MaxAttempts
		}
	}
	step.UserVisibleSummary = strings.TrimSpace(summary)
	step.UpdatedAt = time.Now().UTC()
	state.Steps[stepID] = step
	state.UpdatedAt = step.UpdatedAt
	return nil
}

// validTransition 检查状态转换是否有效。
// 定义了步骤状态机的合法转换。
func validTransition(from, to runtime.StepStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case runtime.StepPending:
		return to == runtime.StepReady || to == runtime.StepBlocked
	case runtime.StepReady:
		return to == runtime.StepRunning || to == runtime.StepCancelled
	case runtime.StepRunning:
		return to == runtime.StepCompleted || to == runtime.StepFailed || to == runtime.StepWaitingApproval || to == runtime.StepCancelled
	case runtime.StepWaitingApproval:
		return to == runtime.StepReady || to == runtime.StepCancelled
	case runtime.StepCompleted, runtime.StepFailed, runtime.StepBlocked, runtime.StepCancelled:
		return false
	default:
		return false
	}
}

// snapshotResult 从步骤状态创建结果快照。
func snapshotResult(step runtime.StepState) StepResult {
	var errInfo *StepError
	if step.ErrorCode != "" || step.ErrorMessage != "" {
		errInfo = &StepError{Code: step.ErrorCode, Message: step.ErrorMessage}
	}
	return StepResult{
		StepID:    step.StepID,
		Status:    step.Status,
		Summary:   step.UserVisibleSummary,
		Error:     errInfo,
		UpdatedAt: step.UpdatedAt,
	}
}

// describePreparedStep 生成步骤准备状态的描述文本。
func describePreparedStep(step planner.PlanStep, status runtime.StepStatus) string {
	switch status {
	case runtime.StepReady:
		return fmt.Sprintf("步骤 %q 已满足依赖，等待 Executor Runtime 调度。", step.Title)
	default:
		return fmt.Sprintf("步骤 %q 已记录，等待上游步骤完成。", step.Title)
	}
}
