// Package events 定义 AI 运行时流式事件的名称常量。
//
// 这些常量在 SSE 流、runtime 包和 orchestrator 中统一使用，
// 避免各处硬编码字符串。
package events

// Name 是 SSE 事件的规范名称类型。
type Name string

const (
	// --- 主运行时事件 ---
	//
	// 简化后的事件流：tool_call -> tool_approval(可选) -> tool_result
	Meta          Name = "meta"          // 会话元信息（session_id/plan_id/turn_id）
	Delta         Name = "delta"         // 模型文本增量输出
	ThinkingDelta Name = "thinking_delta" // 模型思考增量输出（兼容保留）
	ToolCall      Name = "tool_call"     // 工具调用请求
	ToolApproval  Name = "tool_approval" // 工具调用审批等待
	ToolResult    Name = "tool_result"   // 工具调用结果
	Done          Name = "done"          // 本次执行结束
	Error         Name = "error"         // 执行出错
)
