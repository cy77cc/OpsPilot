// Package availability 提供 AI 模块可用性相关的消息定义。
//
// 本文件定义各 AI 层的不可用和无效输出消息，用于友好的错误提示。
package availability

// Layer AI 处理层类型。
type Layer string

// AI 处理层常量。
const (
	LayerRewrite    Layer = "rewrite"    // 改写层
	LayerPlanner    Layer = "planner"    // 规划层
	LayerExpert     Layer = "expert"     // 专家执行层
	LayerSummarizer Layer = "summarizer" // 总结层
)

// UnavailableMessage 返回指定层不可用的用户友好消息。
func UnavailableMessage(layer Layer) string {
	switch layer {
	case LayerRewrite:
		return "AI 理解模块当前不可用，请稍后重试或手动在页面中执行操作。"
	case LayerPlanner:
		return "AI 规划模块当前不可用，请稍后重试或手动在页面中执行操作。"
	case LayerExpert:
		return "AI 执行专家当前不可用，请稍后重试或手动在页面中执行操作。"
	case LayerSummarizer:
		return "AI 总结模块当前不可用，你可以直接查看原始执行结果。"
	default:
		return "AI 模块当前不可用，请稍后重试。"
	}
}

// InvalidOutputMessage 返回指定层输出无效的用户友好消息。
func InvalidOutputMessage(layer Layer) string {
	switch layer {
	case LayerRewrite:
		return "AI 理解模块返回了无效结果，请稍后重试或手动在页面中执行操作。"
	case LayerPlanner:
		return "AI 规划模块返回了无效结果，请稍后重试或手动在页面中执行操作。"
	case LayerExpert:
		return "AI 执行专家返回了无效结果，请稍后重试或手动在页面中执行操作。"
	case LayerSummarizer:
		return "AI 总结模块返回了无效结果，你可以直接查看原始执行结果。"
	default:
		return "AI 模块返回了无效结果，请稍后重试。"
	}
}
