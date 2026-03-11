// Package planner 实现 AI 编排的规划阶段。
//
// 架构概览:
//
//	┌─────────────────────────────────────────────────────────────┐
//	│                         Planner                             │
//	│                                                             │
//	│   Input (message + rewrite) ──▶ LLM ──▶ Decision           │
//	│                                                             │
//	│   Decision types:                                           │
//	│     clarify      → 需要用户澄清                              │
//	│     direct_reply → 直接回复，无需执行计划                     │
//	│     plan         → 生成 ExecutionPlan 并交给 Executor        │
//	└─────────────────────────────────────────────────────────────┘
//
// 文件分布:
//   - planner.go  : Planner struct + Plan/PlanStream 入口
//   - types.go    : 所有导出类型定义
//   - parse.go    : LLM 宽松 JSON 解析
//   - normalize.go: 计划规范化、步骤输入填充、执行前提校验
//   - collect.go  : 从 rewrite.Output 提取资源引用
//   - support.go  : 平台 DB 资源解析辅助
package planner

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"

	"github.com/cy77cc/OpsPilot/internal/ai/availability"
)

// Planner 是规划器核心，负责生成执行计划。
type Planner struct {
	runner *adk.Runner                                              // ADK 运行器
	runFn  func(context.Context, Input, func(string)) (Decision, error) // 执行函数
}

// New 创建新的规划器实例。
func New(runner *adk.Runner) *Planner {
	return &Planner{runner: runner}
}

// NewWithFunc 使用自定义执行函数创建规划器。
func NewWithFunc(runFn func(context.Context, Input, func(string)) (Decision, error)) *Planner {
	return &Planner{runFn: runFn}
}

// Plan 执行规划，返回决策结果。
func (p *Planner) Plan(ctx context.Context, in Input) (Decision, error) {
	return p.plan(ctx, in, nil)
}

// PlanStream 执行规划并支持流式输出。
func (p *Planner) PlanStream(ctx context.Context, in Input, onDelta func(string)) (Decision, error) {
	return p.plan(ctx, in, onDelta)
}

// plan 执行规划的核心逻辑。
func (p *Planner) plan(ctx context.Context, in Input, onDelta func(string)) (Decision, error) {
	if p != nil && p.runFn != nil {
		return p.runFn(ctx, in, onDelta)
	}
	if p == nil || p.runner == nil {
		return Decision{}, &PlanningError{
			Code:              "planner_runner_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerPlanner),
		}
	}
	raw, err := runADKPlanner(ctx, p.runner, buildPromptInput(in), onDelta)
	if err != nil {
		return Decision{}, &PlanningError{
			Code:              "planner_model_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerPlanner),
			Cause:             err,
		}
	}

	parsed, err := ParseDecision(strings.TrimSpace(raw))
	if err != nil {
		return Decision{}, &PlanningError{
			Code:              "planner_invalid_json",
			UserVisibleReason: availability.InvalidOutputMessage(availability.LayerPlanner),
			Cause:             err,
		}
	}
	return normalizeDecision(buildBasePlanContext(in), parsed)
}
