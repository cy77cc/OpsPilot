// Package runtime 提供 AI 运行时的公共事件类型。
//
// PublicStreamEvent 是向外部（前端 SSE）公开的事件结构。
package runtime

// PublicStreamEvent 公开流式事件（与 StreamEvent 等价）。
type PublicStreamEvent = StreamEvent
