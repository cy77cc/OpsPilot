// Package events 定义 AI 运行时事件名称常量。
//
// 事件分为两层：
//
//	内部事件（Internal）：在运行时流水线内部流转，由 orchestrator / ADK hook 产生，
//	不直接暴露给前端，但可映射为公开 SSE 事件。
//
//	公开 SSE 事件（Public）：通过 runtime.EncodePublicEvent 推送给前端，
//	子集由 runtime/stream.go 的 publicEventNames 白名单控制。
//
// DeepAgents 架构事件流：
//
//	[会话层]   Meta
//	[路由层]   AgentHandoff（通过 TaskTool 委派给 Sub-Agent）
//	[执行层]   Delta / ThinkingDelta / ToolCall / ToolApproval / ToolResult
//	[终止层]   Done / Error
package events

// EventName 是 SSE 事件的规范名称类型。
//
// 使用具名类型而非裸字符串，避免在调用点拼写错误。
type EventName string

const (
	// ── 会话层 ────────────────────────────────────────────────────────────────
	//
	// Meta 在对话建立时推送，携带 session_id / turn_id 等元信息。
	// 前端凭此初始化会话上下文。
	Meta EventName = "meta"

	// ── 路由层 ────────────────────────────────────────────────────────────────
	//
	// AgentHandoff 在 Main Agent 通过 TaskTool 将请求委派给 Sub-Agent 时产生，
	// 携带 from / to 两个 Agent 名称。
	//
	// 前端可据此展示「正在转交 K8sAgent…」等路由状态提示。
	AgentHandoff EventName = "agent_handoff"

	// ── 执行层 ────────────────────────────────────────────────────────────────
	//
	// Delta 是 Agent 产生的流式文本增量。
	//
	// 前端将连续的 Delta 事件拼接后渲染为消息气泡内容。
	Delta EventName = "delta"

	// ThinkingDelta 是扩展推理模型（如 o1/deepseek-r1）输出的思考过程增量。
	//
	// 与 Delta 的区别：ThinkingDelta 对应  Thoughts 区块，
	// 前端通常折叠展示或仅在调试模式下显示。
	// 保留用于兼容启用了 Thinking 模式的模型配置。
	ThinkingDelta EventName = "thinking_delta"

	// ── 工具层 ────────────────────────────────────────────────────────────────
	//
	// ToolCall 在 Agent 发出工具调用请求时产生，携带 tool_name / tool_call_id /
	// arguments 等信息。
	//
	// 前端可据此展示「🔧 正在执行 host_exec…」等工具调用指示器。
	ToolCall EventName = "tool_call"

	// ToolApproval 在高风险工具（command_class=dangerous / risk_level=high）
	// 需要人工审批（Human-in-the-Loop）时产生，携带预览信息供用户确认。
	//
	// 前端需弹出审批确认对话框，用户操作后通过 /api/v1/ai/resume/step/stream 恢复执行。
	ToolApproval EventName = "tool_approval"

	// ToolResult 在工具执行完成并返回结果时产生，携带 tool_name / tool_call_id /
	// content（工具原始输出）。
	//
	// 前端通常以折叠卡片形式展示工具输出，避免遮挡主要文本。
	ToolResult EventName = "tool_result"

	// OpsPlanUpdated 在 Ops 计划快照发生整体替换时产生。
	OpsPlanUpdated EventName = "ops_plan_updated"

	// ── 终止层 ────────────────────────────────────────────────────────────────
	//
	// Done 在执行循环正常结束时产生，携带本次执行的迭代次数（iterations）统计。
	//
	// 前端收到后停止流式动画，渲染最终结果。
	Done EventName = "done"

	// Error 在执行过程中发生不可恢复错误时产生，携带错误描述信息。
	//
	// 前端展示错误提示并终止当前会话的流式渲染。
	Error EventName = "error"
)
