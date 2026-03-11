// Package events 定义 AI 编排的事件名称和元数据结构。
//
// 本文件提供事件名称常量、受众类型和事件元数据定义，
// 用于构建 SSE 流式事件和事件路由。
package events

import "time"

// Audience 事件受众类型。
type Audience string

const (
	AudienceUser  Audience = "user"  // 用户可见事件
	AudienceDebug Audience = "debug" // 调试事件
)

// Name 事件名称类型。
type Name string

// 事件名称常量定义。
const (
	Meta             Name = "meta"              // 元数据事件
	RewriteResult    Name = "rewrite_result"    // 改写结果事件
	PlannerState     Name = "planner_state"     // 规划器状态事件
	PlanCreated      Name = "plan_created"      // 计划创建事件
	StageDelta       Name = "stage_delta"       // 阶段增量事件
	StepUpdate       Name = "step_update"       // 步骤更新事件
	ApprovalRequired Name = "approval_required" // 审批请求事件
	ClarifyRequired  Name = "clarify_required"  // 澄清请求事件
	ReplanStarted    Name = "replan_started"    // 重规划开始事件
	Delta            Name = "delta"             // 增量内容事件
	Summary          Name = "summary"           // 总结结果事件
	Done             Name = "done"              // 完成事件
	Error            Name = "error"             // 错误事件
	ToolCall         Name = "tool_call"         // 工具调用事件
	ToolResult       Name = "tool_result"       // 工具结果事件
	Heartbeat        Name = "heartbeat"         // 心跳事件
)

// EventMeta 事件元数据，包含会话和追踪信息。
type EventMeta struct {
	SessionID string    `json:"session_id,omitempty"` // 会话 ID
	TraceID   string    `json:"trace_id,omitempty"`   // 追踪 ID
	PlanID    string    `json:"plan_id,omitempty"`    // 计划 ID
	StepID    string    `json:"step_id,omitempty"`    // 步骤 ID
	Iteration int       `json:"iteration,omitempty"`  // 迭代次数
	Timestamp time.Time `json:"timestamp"`            // 时间戳
}

// WithDefaults 填充默认值。
func (m EventMeta) WithDefaults() EventMeta {
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now().UTC()
	}
	if m.Iteration == 0 {
		m.Iteration = 1
	}
	return m
}
