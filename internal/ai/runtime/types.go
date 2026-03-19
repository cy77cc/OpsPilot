// Package runtime 定义 AI 运行时类型。
//
// 包含流式投影、持久化状态跟踪等核心类型。
package runtime

// PersistedRuntime 存储到数据库的运行时状态。
//
// 与前端 AssistantReplyRuntime 类型保持一致，用于持久化和恢复对话的运行时状态。
// 字段说明:
//   - Phase: 执行阶段枚举（preparing/planning/executing/completed）
//   - PhaseLabel: 阶段显示文本（如"正在规划"）
//   - Plan: 步骤计划信息
//   - Activities: 工具调用活动记录
//   - Summary: 执行摘要
//   - Status: 运行时状态（streaming/completed/error）
type PersistedRuntime struct {
	Phase      string               `json:"phase,omitempty"`
	PhaseLabel string               `json:"phaseLabel,omitempty"`
	Plan       *PersistedPlan       `json:"plan,omitempty"`
	Activities []PersistedActivity  `json:"activities,omitempty"`
	Summary    *PersistedSummary    `json:"summary,omitempty"`
	Status     *PersistedStatus     `json:"status,omitempty"`
}

// PersistedPlan 步骤计划。
//
// 包含步骤列表和当前活动步骤索引。
// ActiveStepIndex 为 -1 或不存在时表示无活动步骤（已完成或未开始）。
type PersistedPlan struct {
	Steps           []PersistedStep `json:"steps,omitempty"`
	ActiveStepIndex int             `json:"activeStepIndex,omitempty"`
}

// PersistedStep 单个步骤。
type PersistedStep struct {
	ID       string             `json:"id"`
	Title    string             `json:"title"`
	Status   string             `json:"status"` // pending, active, done
	Content  string             `json:"content,omitempty"`
	Segments []PersistedSegment `json:"segments,omitempty"` // 内容片段序列（记录文本和工具引用的顺序）
}

// PersistedSegment 内容片段。
//
// 用于记录 step 内容中文本和工具引用的顺序关系。
// Type 可选值: text, tool_ref
type PersistedSegment struct {
	Type   string `json:"type"`             // "text" 或 "tool_ref"
	Text   string `json:"text,omitempty"`   // 文本内容（type=text 时）
	CallID string `json:"callId,omitempty"` // 工具调用 ID（type=tool_ref 时）
}

// PersistedActivity 工具调用活动。
//
// 记录工具调用、审批、结果等活动信息。
// Kind 可选值: agent_handoff, plan, replan, tool_call, tool_approval, tool_result, hint, error
type PersistedActivity struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"`
	Label      string         `json:"label"`
	Detail     string         `json:"detail,omitempty"`
	Status     string         `json:"status,omitempty"`
	StepIndex  int            `json:"stepIndex,omitempty"`
	Arguments  map[string]any `json:"arguments,omitempty"`  // 工具调用参数
	RawContent string         `json:"rawContent,omitempty"` // 完整结果内容
}

// PersistedSummary 执行摘要。
type PersistedSummary struct {
	Title string                 `json:"title,omitempty"`
	Items []PersistedSummaryItem `json:"items,omitempty"`
}

// PersistedSummaryItem 摘要项。
type PersistedSummaryItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Tone  string `json:"tone,omitempty"` // default, success, warning, danger
}

// PersistedStatus 运行时状态。
type PersistedStatus struct {
	Kind  string `json:"kind"`           // streaming, completed, error, interrupted
	Label string `json:"label,omitempty"` // 状态显示文本
}
