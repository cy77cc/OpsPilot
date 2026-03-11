// Package executor 实现 AI 编排的执行阶段。
//
// Executor 负责执行 Planner 生成的执行计划，调用专家 Agent 完成各步骤。
// 支持步骤依赖管理、风险控制、审批流程和自动重试。
package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

// Request 是执行请求的结构。
type Request struct {
	TraceID        string              // 追踪 ID
	SessionID      string              // 会话 ID
	Message        string              // 用户原始消息
	Plan           planner.ExecutionPlan // 执行计划
	RuntimeContext runtime.ContextSnapshot // 运行时上下文
	EventMeta      events.EventMeta    // 事件元数据
	EmitEvent      EventEmitter        // 事件发射器
}

// ResumeRequest 是恢复执行的请求结构。
type ResumeRequest struct {
	SessionID string `json:"session_id"` // 会话 ID
	PlanID    string `json:"plan_id"`    // 计划 ID
	StepID    string `json:"step_id"`    // 步骤 ID
	Approved  bool   `json:"approved"`   // 是否批准
	Reason    string `json:"reason,omitempty"` // 拒绝原因
}

// ApprovalDecision 表示审批决策。
type ApprovalDecision struct {
	PlanID      string `json:"plan_id"`          // 计划 ID
	StepID      string `json:"step_id"`          // 步骤 ID
	Approved    bool   `json:"approved"`         // 是否批准
	Reason      string `json:"reason,omitempty"` // 拒绝原因
	Idempotency string `json:"idempotency"`      // 幂等键
	Status      string `json:"status,omitempty"` // 状态
}

// Evidence 表示步骤执行的证据。
type Evidence struct {
	Kind   string         `json:"kind,omitempty"`   // 证据类型
	Source string         `json:"source,omitempty"` // 证据来源
	Data   map[string]any `json:"data,omitempty"`   // 证据数据
}

// StepError 表示步骤执行错误。
type StepError struct {
	Code    string `json:"code,omitempty"`    // 错误码
	Message string `json:"message,omitempty"` // 错误消息
}

// StepResult 表示步骤执行结果。
type StepResult struct {
	StepID    string             `json:"step_id"`              // 步骤 ID
	Status    runtime.StepStatus `json:"status"`               // 步骤状态
	Summary   string             `json:"summary,omitempty"`    // 结果摘要
	Evidence  []Evidence         `json:"evidence,omitempty"`   // 执行证据
	Error     *StepError         `json:"error,omitempty"`      // 错误信息
	UpdatedAt time.Time          `json:"updated_at"`           // 更新时间
}

// Result 表示执行结果。
type Result struct {
	State runtime.ExecutionState `json:"state"`         // 执行状态
	Steps []StepResult           `json:"steps,omitempty"` // 步骤结果列表
}

// StepRunner 定义步骤运行器接口。
// 由 ExpertRunner 实现，负责调用专家 Agent 执行单个步骤。
type StepRunner interface {
	RunStep(ctx context.Context, req Request, step planner.PlanStep) (StepResult, error)
}

// EventEmitter 定义事件发射器函数类型。
type EventEmitter func(name string, meta events.EventMeta, data map[string]any) bool

// ExecutionError 表示执行错误。
type ExecutionError struct {
	Code        string // 错误码
	Message     string // 错误消息
	UserSummary string // 用户可见摘要
}

// Error 实现 error 接口。
func (e *ExecutionError) Error() string {
	if e == nil {
		return ""
	}
	return strings.TrimSpace(e.Message)
}

// Executor 是执行器核心，负责执行计划和管理步骤状态。
type Executor struct {
	store      *runtime.ExecutionStore // 执行状态存储
	stepRunner StepRunner              // 步骤运行器
}

// RiskPolicy 定义步骤的风险策略。
type RiskPolicy struct {
	RequiresApproval bool `json:"requires_approval"` // 是否需要审批
	MaxAttempts      int  `json:"max_attempts"`      // 最大尝试次数
	AutoRetry        bool `json:"auto_retry"`        // 是否自动重试
}

// Option 定义执行器选项函数。
type Option func(*Executor)

// WithStepRunner 设置步骤运行器。
func WithStepRunner(stepRunner StepRunner) Option {
	return func(e *Executor) {
		e.stepRunner = stepRunner
	}
}

// New 创建新的执行器实例。
func New(store *runtime.ExecutionStore, opts ...Option) *Executor {
	exec := &Executor{store: store}
	for _, opt := range opts {
		if opt != nil {
			opt(exec)
		}
	}
	return exec
}

// riskPolicy 根据操作模式和风险等级确定风险策略。
// mutating 或 high 风险的操作需要审批且不可自动重试。
func riskPolicy(mode, risk string) RiskPolicy {
	mode = strings.ToLower(strings.TrimSpace(mode))
	risk = strings.ToLower(strings.TrimSpace(risk))
	switch {
	case mode == "mutating" || risk == "high":
		return RiskPolicy{RequiresApproval: true, MaxAttempts: 1, AutoRetry: false}
	case risk == "medium":
		return RiskPolicy{RequiresApproval: true, MaxAttempts: 1, AutoRetry: false}
	default:
		return RiskPolicy{RequiresApproval: false, MaxAttempts: 2, AutoRetry: true}
	}
}

// PrepareState 准备执行状态，初始化所有步骤。
// 根据依赖关系确定初始状态，无依赖的步骤设为 Ready。
func (e *Executor) PrepareState(_ context.Context, req Request) (runtime.ExecutionState, []StepResult, error) {
	planID := strings.TrimSpace(req.Plan.PlanID)
	if planID == "" {
		return runtime.ExecutionState{}, nil, fmt.Errorf("plan_id is required")
	}
	if strings.TrimSpace(req.SessionID) == "" {
		return runtime.ExecutionState{}, nil, fmt.Errorf("session_id is required")
	}

	now := time.Now().UTC()
	steps := make(map[string]runtime.StepState, len(req.Plan.Steps))
	results := make([]StepResult, 0, len(req.Plan.Steps))
	for _, step := range req.Plan.Steps {
		if strings.TrimSpace(step.StepID) == "" {
			return runtime.ExecutionState{}, nil, fmt.Errorf("plan step missing step_id")
		}
		policy := riskPolicy(step.Mode, step.Risk)
		status := runtime.StepPending
		if len(step.DependsOn) == 0 {
			status = runtime.StepReady
		}
		steps[step.StepID] = runtime.StepState{
			StepID:             step.StepID,
			Title:              step.Title,
			Expert:             step.Expert,
			Intent:             step.Intent,
			Task:               step.Task,
			Input:              step.Input,
			DependsOn:          append([]string(nil), step.DependsOn...),
			Status:             status,
			Mode:               strings.TrimSpace(step.Mode),
			Risk:               strings.TrimSpace(step.Risk),
			MaxAttempts:        policy.MaxAttempts,
			IdempotencyKey:     runtime.ApprovalDecisionHash(planID, step.StepID, false),
			UserVisibleSummary: describePreparedStep(step, status),
			UpdatedAt:          now,
		}
		results = append(results, StepResult{
			StepID:    step.StepID,
			Status:    status,
			Summary:   describePreparedStep(step, status),
			UpdatedAt: now,
		})
	}

	state := runtime.ExecutionState{
		TraceID:        req.TraceID,
		SessionID:      req.SessionID,
		PlanID:         planID,
		Message:        strings.TrimSpace(req.Message),
		Status:         runtime.ExecutionStatusRunning,
		Phase:          "executor_prepared",
		RuntimeContext: req.RuntimeContext,
		Steps:          steps,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return state, results, nil
}

// SavePreparedState 准备并保存执行状态到存储。
func (e *Executor) SavePreparedState(ctx context.Context, req Request) (*Result, error) {
	state, steps, err := e.PrepareState(ctx, req)
	if err != nil {
		return nil, err
	}
	if e != nil && e.store != nil {
		if err := e.store.Save(ctx, state); err != nil {
			return nil, err
		}
	}
	return &Result{State: state, Steps: steps}, nil
}

// Resume 恢复等待审批的执行。
// 根据审批结果继续或取消后续步骤。
func (e *Executor) Resume(ctx context.Context, req ResumeRequest) (*Result, error) {
	if e == nil || e.store == nil {
		return nil, fmt.Errorf("execution store is not configured")
	}
	state, err := e.store.Load(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("execution state not found")
	}
	return advanceAfterApproval(ctx, e.store, state, req, e.stepRunner)
}

// Run 执行计划，保存状态并启动调度器。
func (e *Executor) Run(ctx context.Context, req Request) (*Result, error) {
	prepared, err := e.SavePreparedState(ctx, req)
	if err != nil {
		return nil, err
	}
	if e == nil || e.store == nil {
		return prepared, nil
	}
	return advanceScheduler(ctx, e.store, &prepared.State, req, e.stepRunner)
}

// RecordFailure 记录步骤执行失败。
// 根据风险策略决定是自动重试还是标记为失败。
func (e *Executor) RecordFailure(ctx context.Context, sessionID, stepID, code, message string) (*Result, error) {
	if e == nil || e.store == nil {
		return nil, fmt.Errorf("execution store is not configured")
	}
	state, err := e.store.Load(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, fmt.Errorf("execution state not found")
	}
	step, ok := state.Steps[stepID]
	if !ok {
		return nil, fmt.Errorf("step %s not found", stepID)
	}
	step.ErrorCode = strings.TrimSpace(code)
	step.ErrorMessage = strings.TrimSpace(message)
	if shouldAutoRetry(step) {
		step.Status = runtime.StepReady
		step.UserVisibleSummary = "step failed once and will be retried automatically"
	} else {
		step.Status = runtime.StepFailed
		step.UserVisibleSummary = "step failed and requires manual follow-up"
		state.Status = runtime.ExecutionStatusFailed
		state.Phase = "executor_failed"
		markDependentsBlocked(state, stepID)
	}
	step.UpdatedAt = time.Now().UTC()
	state.Steps[stepID] = step
	if err := e.store.Save(ctx, *state); err != nil {
		return nil, err
	}
	return &Result{State: *state, Steps: []StepResult{snapshotResult(step)}}, nil
}

// Approval 返回当前待审批信息。
func (r *Result) Approval() *runtime.PendingApproval {
	if r == nil {
		return nil
	}
	return r.State.PendingApproval
}

// StepState 返回指定步骤的状态。
func (r *Result) StepState(stepID string) (runtime.StepState, bool) {
	if r == nil || r.State.Steps == nil {
		return runtime.StepState{}, false
	}
	step, ok := r.State.Steps[stepID]
	return step, ok
}
