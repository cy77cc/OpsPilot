// Package events 定义 AI 运行时流式事件的名称常量。
//
// 这些常量在 SSE 流、runtime 包和 orchestrator 中统一使用，
// 避免各处硬编码字符串。
package events

// Name 是 SSE 事件的规范名称类型。
type Name string

const (
	// --- 通用事件 ---

	Meta             Name = "meta"             // 会话元信息（session_id/plan_id/turn_id）
	Delta            Name = "delta"            // 模型文本增量输出
	ThinkingDelta    Name = "thinking_delta"   // 模型思考链增量输出
	ToolCall         Name = "tool_call"        // 工具调用请求
	ToolResult       Name = "tool_result"      // 工具调用结果
	ApprovalRequired Name = "approval_required" // 需要人工审批
	Done             Name = "done"             // 本次执行结束
	Error            Name = "error"            // 执行出错

	// --- 规划阶段事件（plan-execute 架构专用）---

	RewriteResult   Name = "rewrite_result"   // 意图改写结果
	PlannerState    Name = "planner_state"    // 规划器状态变更
	PlanCreated     Name = "plan_created"     // 执行计划已生成
	StageDelta      Name = "stage_delta"      // 阶段级别的流式增量
	StepUpdate      Name = "step_update"      // 单步状态更新
	ClarifyRequired Name = "clarify_required" // 需要用户澄清意图
	ReplanStarted   Name = "replan_started"   // 重新规划开始

	// --- 轮次生命周期事件 ---

	TurnStarted Name = "turn_started" // 新一轮对话开始
	TurnState   Name = "turn_state"   // 轮次状态变更（running/completed 等）
)
