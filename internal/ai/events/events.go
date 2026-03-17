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
// 事件层次与 Plan-Execute-Replan 执行流的对应关系：
//
//	[会话层]   Meta
//	[路由层]   AgentHandoff
//	[规划层]   Plan  →  Replan（多轮迭代）
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
	// AgentHandoff 在 RouterAgent（OpsPilotAgent）通过 transfer_to_agent 工具
	// 将请求转交给子 Agent 时产生，携带 from / to 两个 Agent 名称。
	//
	// 对应 JSONL 中：action.transfer_to 非空（role=tool, tool_name=transfer_to_agent）。
	// 前端可据此展示「正在转交 DiagnosisAgent…」等路由状态提示。
	AgentHandoff EventName = "agent_handoff"

	// ── 规划层 ────────────────────────────────────────────────────────────────
	//
	// Plan 由 Planner 子 Agent 输出，内容为初始任务步骤列表（JSON steps 数组）。
	//
	// 对应 JSONL 中：agent_name=planner, role=assistant, content 为 {"steps":[...]}。
	// 前端可据此渲染任务清单 / 进度条 UI。
	Plan EventName = "plan"

	// Replan 由 Replanner 子 Agent 输出，内容为更新后的剩余步骤（JSON steps 数组）
	// 或最终答复（JSON response 字段，此时执行即将结束）。
	//
	// 对应 JSONL 中：agent_name=replanner, role=assistant, content 为
	// {"steps":[...]} 或 {"response":"..."}。
	// 前端可据此实时更新步骤清单（勾选已完成项、追加新步骤）。
	Replan EventName = "replan"

	// ── 执行层 ────────────────────────────────────────────────────────────────
	//
	// Delta 是 Executor 或其他 Agent 产生的流式文本增量。
	//
	// 对应 JSONL 中：role=assistant, is_streaming=true, content 非空且无 tool_calls，
	// 或既有 content 又有 tool_calls 时的文本部分。
	// 前端将连续的 Delta 事件拼接后渲染为消息气泡内容。
	Delta EventName = "delta"

	// ThinkingDelta 是扩展推理模型（如 o1/deepseek-r1）输出的思考过程增量。
	//
	// 与 Delta 的区别：ThinkingDelta 对应 <think>…</think> 区块，
	// 前端通常折叠展示或仅在调试模式下显示。
	// 保留用于兼容启用了 Thinking 模式的模型配置。
	ThinkingDelta EventName = "thinking_delta"

	// ── 工具层 ────────────────────────────────────────────────────────────────
	//
	// ToolCall 在 Executor 发出工具调用请求时产生，携带 tool_name / tool_call_id /
	// arguments 等信息。
	//
	// 对应 JSONL 中：role=assistant, is_streaming=true, tool_calls 非空。
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
	// 对应 JSONL 中：role=tool, is_streaming=false。
	// 前端通常以折叠卡片形式展示工具输出，避免遮挡主要文本。
	ToolResult EventName = "tool_result"

	// ── 终止层 ────────────────────────────────────────────────────────────────
	//
	// Done 在执行循环正常结束时产生，携带本次执行的迭代次数（iterations）统计。
	//
	// 对应 JSONL 中：action.break_loop.done=true。
	// 前端收到后停止流式动画，渲染最终结果。
	Done EventName = "done"

	// Error 在执行过程中发生不可恢复错误时产生，携带错误描述信息。
	//
	// 前端展示错误提示并终止当前会话的流式渲染。
	Error EventName = "error"
)
