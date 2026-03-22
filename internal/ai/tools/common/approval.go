// Package common 提供 AI 工具的公共类型和审批机制。
//
// 本文件定义工具审批相关的核心类型，用于支持 Human-in-the-Loop (HITL) 工作流：
//   - ApprovalInfo: 审批请求信息，展示给用户决策
//   - ApprovalResult: 用户审批结果，通过 ResumeWithParams 传递
//   - ApprovalPreview: 审批预览，帮助用户理解操作影响
//
// 审批流程:
//  1. 工具执行前通过 StatefulInterrupt 暂停，发送 ApprovalInfo
//  2. 前端展示审批界面，用户确认或拒绝
//  3. 用户决策通过 ResumeWithParams 携带 ApprovalResult 恢复执行
//  4. 中间件根据审批结果决定继续执行或返回拒绝消息
package common

import (
	"context"
	"time"
)

// ApprovalInfo 审批请求信息，展示给用户决策。
//
// 当高风险工具需要人工确认时，中间件会创建此结构并通过
// tool.StatefulInterrupt 发送给前端。
type ApprovalInfo struct {
	// ToolName 需要审批的工具名称
	ToolName string `json:"tool_name"`

	// ArgumentsInJSON 工具调用参数的 JSON 字符串
	ArgumentsInJSON string `json:"arguments_in_json"`

	// Preview 审批预览信息，帮助用户理解操作影响
	Preview ApprovalPreview `json:"preview"`

	// TimeoutSeconds 审批超时时间（秒），超时后自动拒绝
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
}

// ApprovalPreview 审批预览信息，帮助用户理解操作影响。
//
// 前端应该将这些信息清晰地展示给用户，帮助用户做出正确的审批决策。
type ApprovalPreview struct {
	// Action 操作类型描述，如 "批量执行命令"、"扩缩容部署" 等
	Action string `json:"action"`

	// Target 操作目标，如主机名、Pod名、服务名、主机数量等
	Target string `json:"target"`

	// RiskLevel 风险等级: low, medium, high, critical
	RiskLevel string `json:"risk_level"`

	// Impact 操作影响描述，说明此操作可能带来的影响
	Impact string `json:"impact"`

	// Warnings 警告信息列表，提示用户需要注意的风险点
	Warnings []string `json:"warnings,omitempty"`

	// Extra 额外的预览信息，特定工具可以扩展
	Extra map[string]any `json:"extra,omitempty"`
}

// ApprovalResult 用户审批结果，通过 ResumeWithParams 传递。
//
// 用户在前端审批界面做出决策后，结果会封装为此结构，
// 通过 API 调用传递给后端的 ResumeWithParams 方法恢复执行。
type ApprovalResult struct {
	// Approved 是否批准执行
	Approved bool `json:"approved"`

	// DisapproveReason 拒绝原因，当 Approved 为 false 时可选填写
	DisapproveReason *string `json:"disapprove_reason,omitempty"`

	// ApprovedBy 审批人标识，用于审计记录
	ApprovedBy string `json:"approved_by,omitempty"`

	// ApprovedAt 审批时间，用于审计记录
	ApprovedAt *time.Time `json:"approved_at,omitempty"`

	// Comment 审批备注
	Comment string `json:"comment,omitempty"`
}

// ToolRiskConfig 工具风险配置，定义工具的审批策略。
//
// 用于 ApprovalMiddleware 配置，决定哪些工具需要审批以及如何生成预览。
type ToolRiskConfig struct {
	// ToolName 工具名称
	ToolName string

	// RiskLevel 默认风险等级: low, medium, high, critical
	RiskLevel string

	// NeedsApproval 是否需要审批
	NeedsApproval bool

	// PreviewGenerator 预览生成器函数，根据参数生成 ApprovalPreview
	// 参数 args 为工具调用的 JSON 参数字符串
	PreviewGenerator func(args string) ApprovalPreview
}

// ApprovalEvalMeta carries runtime metadata used to evaluate a tool-call approval policy.
type ApprovalEvalMeta struct {
	SessionID      string
	RunID          string
	CheckpointID   string
	CallID         string
	Scene          string
	AgentRole      string
	CommandClass   string
	UserID         uint64
	TimeoutSeconds int
}

// ApprovalDecision captures the evaluator output used by the approval middleware.
type ApprovalDecision struct {
	RequiresApproval  bool
	ApprovalID        string
	Preview           ApprovalPreview
	TimeoutSeconds    int
	MatchedRuleID     *uint64
	PolicyVersion     string
	DecisionSource    string
	ExpiresAt         time.Time
	BoundSessionID    string
	BoundAgentRole    string
	ApproverID        string
	ApprovalTimestamp *time.Time
	RejectReason      string
	PolicyViolations  []string
}

// ApprovalEvaluator can decide whether a tool call requires human approval.
type ApprovalEvaluator interface {
	Evaluate(ctx context.Context, toolName string, args string, meta ApprovalEvalMeta) (*ApprovalDecision, error)
}

// RiskLevel 风险等级常量。
const (
	RiskLevelLow      = "low"
	RiskLevelMedium   = "medium"
	RiskLevelHigh     = "high"
	RiskLevelCritical = "critical"
)

// DefaultApprovalTimeout 默认审批超时时间（秒）。
const DefaultApprovalTimeout = 300 // 5分钟
