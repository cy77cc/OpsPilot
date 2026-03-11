// Package events 定义 AI 编排的事件类型和载荷结构。
//
// 本文件提供各阶段的事件载荷定义，用于 SSE 流式传输和状态同步。
// 事件用于在改写、规划、执行、总结等阶段之间传递状态和数据。
package events

import (
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

// RewriteResultPayload 改写结果事件载荷。
type RewriteResultPayload struct {
	Rewrite            rewrite.Output `json:"rewrite"`
	UserVisibleSummary string         `json:"user_visible_summary,omitempty"`
}

// PlannerStatePayload 规划器状态事件载荷。
type PlannerStatePayload struct {
	Status             string `json:"status"`
	UserVisibleSummary string `json:"user_visible_summary,omitempty"`
}

// PlanCreatedPayload 计划创建事件载荷。
type PlanCreatedPayload struct {
	Plan               *planner.ExecutionPlan `json:"plan,omitempty"`
	UserVisibleSummary string                 `json:"user_visible_summary,omitempty"`
}

// StageDeltaPayload 阶段增量事件载荷，用于流式输出。
type StageDeltaPayload struct {
	Stage        string `json:"stage"`
	ContentChunk string `json:"content_chunk,omitempty"`
	Status       string `json:"status,omitempty"`
	StepID       string `json:"step_id,omitempty"`
	Expert       string `json:"expert,omitempty"`
	Replace      bool   `json:"replace,omitempty"`
}

// StepUpdatePayload 步骤更新事件载荷。
type StepUpdatePayload struct {
	PlanID             string             `json:"plan_id,omitempty"`
	StepID             string             `json:"step_id,omitempty"`
	Status             runtime.StepStatus `json:"status"`
	Title              string             `json:"title,omitempty"`
	Expert             string             `json:"expert,omitempty"`
	UserVisibleSummary string             `json:"user_visible_summary,omitempty"`
}

// ApprovalResumeHint 审批回调的恢复信息。
type ApprovalResumeHint struct {
	SessionID string `json:"session_id"`
	PlanID    string `json:"plan_id"`
	StepID    string `json:"step_id"`
}

// ApprovalRequiredPayload 审批请求事件载荷。
type ApprovalRequiredPayload struct {
	SessionID          string             `json:"session_id,omitempty"`
	PlanID             string             `json:"plan_id,omitempty"`
	StepID             string             `json:"step_id,omitempty"`
	Title              string             `json:"title,omitempty"`
	Risk               string             `json:"risk,omitempty"`
	Mode               string             `json:"mode,omitempty"`
	Status             string             `json:"status,omitempty"`
	UserVisibleSummary string             `json:"user_visible_summary,omitempty"`
	Resume             ApprovalResumeHint `json:"resume"`
}

// ClarifyRequiredPayload 澄清请求事件载荷。
type ClarifyRequiredPayload struct {
	Kind       string           `json:"kind,omitempty"`
	Title      string           `json:"title,omitempty"`
	Message    string           `json:"message,omitempty"`
	Candidates []map[string]any `json:"candidates,omitempty"`
}

// ReplanStartedPayload 重规划开始事件载荷。
type ReplanStartedPayload struct {
	Reason         string `json:"reason,omitempty"`
	PreviousPlanID string `json:"previous_plan_id,omitempty"`
}

// DeltaPayload 增量内容事件载荷。
type DeltaPayload struct {
	ContentChunk string `json:"content_chunk,omitempty"`
}

// SummaryPayload 总结结果事件载荷。
type SummaryPayload struct {
	Summary string `json:"summary,omitempty"`
}

// ErrorPayload 错误事件载荷。
type ErrorPayload struct {
	Message     string `json:"message"`
	ErrorCode   string `json:"error_code,omitempty"`
	Stage       string `json:"stage,omitempty"`
	Recoverable bool   `json:"recoverable,omitempty"`
}
