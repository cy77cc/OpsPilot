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
	// 新的后端主路径以 turn 生命周期和正文/审批流为中心，
	// 这些事件应视为当前唯一的语义主链。
	Meta             Name = "meta"              // 会话元信息（session_id/plan_id/turn_id）
	Delta            Name = "delta"             // 模型文本增量输出
	ThinkingDelta    Name = "thinking_delta"    // 模型思考增量输出（兼容保留）
	ToolCall         Name = "tool_call"         // 工具调用请求
	ToolResult       Name = "tool_result"       // 工具调用结果
	TurnState        Name = "turn_state"        // 轮次状态变更（running/completed 等）
	ChainStarted     Name = "chain_started"     // 原生思维链开始
	ChainNodeOpen    Name = "chain_node_open"   // 原生思维链节点开始
	ChainNodePatch   Name = "chain_node_patch"  // 原生思维链节点增量更新
	ChainNodeClose   Name = "chain_node_close"  // 原生思维链节点结束
	ChainCollapsed   Name = "chain_collapsed"   // 原生思维链可折叠完成
	FinalAnswerStart Name = "final_answer_started"
	FinalAnswerDelta Name = "final_answer_delta"
	FinalAnswerDone  Name = "final_answer_done"
	Done             Name = "done"  // 本次执行结束
	Error            Name = "error" // 执行出错

)
