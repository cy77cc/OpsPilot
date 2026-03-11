// Package ai 提供 AI 编排层的指标收集功能。
//
// 本文件定义了 AI 编排各阶段的指标收集结构和逻辑。
// 用于监控 Rewrite、Planner、Resume 和 ThoughtChain 的运行状态。
package ai

import (
	"strings"
	"sync"

	"github.com/cy77cc/OpsPilot/internal/ai/events"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
)

// AIMetrics 是 AI 编排层的指标收集器。
// 线程安全，使用互斥锁保护内部状态。
type AIMetrics struct {
	mu           sync.Mutex                 // 保护并发访问
	rewrite      RewriteMetricsSnapshot     // 改写阶段指标
	planner      PlannerMetricsSnapshot     // 规划阶段指标
	resume       ResumeMetricsSnapshot      // 恢复操作指标
	thoughtChain ThoughtChainMetricsSnapshot // 思维链指标
}

// AIMetricsSnapshot 是指标快照，用于序列化和导出。
type AIMetricsSnapshot struct {
	Rewrite      RewriteMetricsSnapshot      `json:"rewrite"`       // 改写阶段指标
	Planner      PlannerMetricsSnapshot      `json:"planner"`       // 规划阶段指标
	Resume       ResumeMetricsSnapshot       `json:"resume"`        // 恢复操作指标
	ThoughtChain ThoughtChainMetricsSnapshot `json:"thought_chain"` // 思维链指标
}

// RewriteMetricsSnapshot 记录改写阶段的指标。
type RewriteMetricsSnapshot struct {
	Total             int     `json:"total"`              // 总请求数
	StructuredOutputs int     `json:"structured_outputs"` // 结构化输出数
	Fallbacks         int     `json:"fallbacks"`          // 降级次数
	AmbiguousOutputs  int     `json:"ambiguous_outputs"`  // 模糊输出数
	QualityRate       float64 `json:"quality_rate"`       // 质量率
}

// PlannerMetricsSnapshot 记录规划阶段的指标。
type PlannerMetricsSnapshot struct {
	Total              int     `json:"total"`               // 总请求数
	Clarify            int     `json:"clarify"`             // 需要澄清的次数
	Plans              int     `json:"plans"`               // 生成计划的次数
	ExecutablePlans    int     `json:"executable_plans"`    // 可执行计划数
	DirectReplies      int     `json:"direct_replies"`      // 直接回复次数
	Rejected           int     `json:"rejected"`            // 拒绝次数
	ClarifyRate        float64 `json:"clarify_rate"`        // 澄清率
	ExecutablePlanRate float64 `json:"executable_plan_rate"` // 可执行计划率
}

// ResumeMetricsSnapshot 记录恢复操作的指标。
type ResumeMetricsSnapshot struct {
	Total                  int     `json:"total"`                   // 总恢复请求数
	Successful             int     `json:"successful"`              // 成功次数
	Failures               int     `json:"failures"`                // 失败次数
	DuplicateIntercepted   int     `json:"duplicate_intercepted"`   // 重复请求拦截数
	SuccessRate            float64 `json:"success_rate"`            // 成功率
	DuplicateInterceptRate float64 `json:"duplicate_intercept_rate"` // 重复拦截率
}

// ThoughtChainMetricsSnapshot 记录思维链的指标。
type ThoughtChainMetricsSnapshot struct {
	Runs                   int     `json:"runs"`                    // 总运行次数
	ExpectedStageSignals   int     `json:"expected_stage_signals"`  // 预期阶段信号数
	DeliveredStageSignals  int     `json:"delivered_stage_signals"` // 实际阶段信号数
	RunsWithMissingSignals int     `json:"runs_with_missing_signals"` // 缺失信号的运行数
	EventCompletenessRate  float64 `json:"event_completeness_rate"`  // 事件完整率
}

// thoughtChainRunMetrics 追踪单次思维链运行的指标。
type thoughtChainRunMetrics struct {
	parent          *AIMetrics            // 父指标收集器
	requiredStages  map[string]struct{}   // 必需的阶段
	deliveredStages map[string]struct{}   // 已交付的阶段
}

// NewAIMetrics 创建新的指标收集器。
func NewAIMetrics() *AIMetrics {
	return &AIMetrics{}
}

// Snapshot 返回当前指标的快照。
// 线程安全，可用于导出和展示。
func (m *AIMetrics) Snapshot() AIMetricsSnapshot {
	if m == nil {
		return AIMetricsSnapshot{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return AIMetricsSnapshot{
		Rewrite:      m.rewrite,
		Planner:      m.planner,
		Resume:       m.resume,
		ThoughtChain: m.thoughtChain,
	}
}

// RecordRewrite 记录改写阶段的指标。
// 根据输出结构统计结构化输出、降级和模糊输出数量。
func (m *AIMetrics) RecordRewrite(out rewrite.Output) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rewrite.Total++
	if isStructuredRewrite(out) {
		m.rewrite.StructuredOutputs++
	}
	if isRewriteFallback(out) {
		m.rewrite.Fallbacks++
	}
	if len(out.AmbiguityFlags) > 0 || len(out.Ambiguities) > 0 {
		m.rewrite.AmbiguousOutputs++
	}
	m.rewrite.QualityRate = rate(m.rewrite.StructuredOutputs, m.rewrite.Total)
}

// RecordPlanner 记录规划阶段的指标。
// 根据决策类型统计 clarify、plan、direct_reply、reject 数量。
func (m *AIMetrics) RecordPlanner(decision planner.Decision) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.planner.Total++
	switch decision.Type {
	case planner.DecisionClarify:
		m.planner.Clarify++
	case planner.DecisionPlan:
		m.planner.Plans++
		if planIsExecutable(decision.Plan) {
			m.planner.ExecutablePlans++
		}
	case planner.DecisionDirectReply:
		m.planner.DirectReplies++
	case planner.DecisionReject:
		m.planner.Rejected++
	}
	m.planner.ClarifyRate = rate(m.planner.Clarify, m.planner.Total)
	m.planner.ExecutablePlanRate = rate(m.planner.ExecutablePlans, m.planner.Plans)
}

// RecordResume 记录恢复操作的指标。
// 根据状态统计成功、失败和重复拦截数量。
func (m *AIMetrics) RecordResume(status string, err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resume.Total++
	if err != nil {
		m.resume.Failures++
	} else {
		switch strings.TrimSpace(status) {
		case "idempotent":
			m.resume.Successful++
			m.resume.DuplicateIntercepted++
		case "approved", "approval_granted", "rejected", "cancelled", "noop":
			m.resume.Successful++
		default:
			m.resume.Failures++
		}
	}
	m.resume.SuccessRate = rate(m.resume.Successful, m.resume.Total)
	m.resume.DuplicateInterceptRate = rate(m.resume.DuplicateIntercepted, m.resume.Total)
}

// StartThoughtChainRun 开始追踪一次思维链运行。
// 返回的运行指标对象会在思维链结束时自动汇总到父指标。
func (m *AIMetrics) StartThoughtChainRun() *thoughtChainRunMetrics {
	if m == nil {
		return nil
	}
	return &thoughtChainRunMetrics{
		parent:          m,
		requiredStages:  make(map[string]struct{}),
		deliveredStages: make(map[string]struct{}),
	}
}

// Observe 观察流式事件，追踪阶段交付情况。
// 根据事件类型记录必需阶段和已交付阶段。
func (r *thoughtChainRunMetrics) Observe(evt StreamEvent) {
	if r == nil {
		return
	}
	switch evt.Type {
	case events.RewriteResult:
		r.requiredStages["rewrite"] = struct{}{}
	case events.PlannerState, events.PlanCreated:
		r.requiredStages["plan"] = struct{}{}
	case events.StepUpdate, events.ToolCall, events.ToolResult:
		r.requiredStages["execute"] = struct{}{}
	case events.ApprovalRequired, events.ClarifyRequired:
		r.requiredStages["user_action"] = struct{}{}
		r.deliveredStages["user_action"] = struct{}{}
	case events.StageDelta:
		stage := strings.TrimSpace(stringValue(evt.Data["stage"]))
		if stage != "" && stage != "summary" {
			r.deliveredStages[stage] = struct{}{}
		}
	}
}

// Finalize 完成思维链运行的指标收集。
// 将本次运行的阶段交付情况汇总到父指标。
func (r *thoughtChainRunMetrics) Finalize() {
	if r == nil || r.parent == nil {
		return
	}
	requiredCount := len(r.requiredStages)
	if requiredCount == 0 {
		return
	}
	deliveredCount := 0
	for stage := range r.requiredStages {
		if _, ok := r.deliveredStages[stage]; ok {
			deliveredCount++
		}
	}

	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()
	r.parent.thoughtChain.Runs++
	r.parent.thoughtChain.ExpectedStageSignals += requiredCount
	r.parent.thoughtChain.DeliveredStageSignals += deliveredCount
	if deliveredCount < requiredCount {
		r.parent.thoughtChain.RunsWithMissingSignals++
	}
	r.parent.thoughtChain.EventCompletenessRate = rate(
		r.parent.thoughtChain.DeliveredStageSignals,
		r.parent.thoughtChain.ExpectedStageSignals,
	)
}

// isStructuredRewrite 判断改写输出是否为结构化输出。
// 结构化输出需要包含 NormalizedGoal、OperationMode 和 Narrative。
func isStructuredRewrite(out rewrite.Output) bool {
	return strings.TrimSpace(out.NormalizedGoal) != "" &&
		strings.TrimSpace(out.OperationMode) != "" &&
		strings.TrimSpace(out.Narrative) != ""
}

// isRewriteFallback 判断改写是否使用了降级逻辑。
// 降级逻辑会在假设列表中添加 "rewrite_" 前缀的假设。
func isRewriteFallback(out rewrite.Output) bool {
	for _, assumption := range out.Assumptions {
		if strings.HasPrefix(strings.TrimSpace(assumption), "rewrite_") {
			return true
		}
	}
	return false
}

// planIsExecutable 判断执行计划是否可执行。
// 可执行计划需要有有效的 PlanID 和所有步骤都有有效的 StepID 和 Expert。
func planIsExecutable(plan *planner.ExecutionPlan) bool {
	if plan == nil || strings.TrimSpace(plan.PlanID) == "" || len(plan.Steps) == 0 {
		return false
	}
	for _, step := range plan.Steps {
		if strings.TrimSpace(step.StepID) == "" || strings.TrimSpace(step.Expert) == "" {
			return false
		}
	}
	return true
}

// rate 计算比率，避免除零错误。
func rate(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}
