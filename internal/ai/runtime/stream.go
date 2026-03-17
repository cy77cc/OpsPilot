// Package runtime 提供 AI 运行时的 SSE 流式编码工具。
//
// 本包负责将内部事件序列化为前端可消费的 SSE 消息，
// 并通过白名单机制防止内部事件泄露给外部调用方。
//
// 事件分层说明：
//
//	公开事件（publicEventNames 白名单）：可通过 EncodePublicEvent 编码推送给前端。
//	内部事件（如 thinking_delta）：仅在运行时内部流转，不对外暴露。
package runtime

import (
	"encoding/json"
	"fmt"
)

// StreamEvent 是推送给前端的 SSE 消息体。
//
// Event 字段与前端约定的事件名称对应，Data 为该事件的结构化载荷。
type StreamEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// publicEventNames 是允许通过 EncodePublicEvent 编码的事件名称白名单。
//
// 白名单与 internal/ai/events/events.go 中的公开事件常量保持一致：
//
//	会话层：meta
//	路由层：agent_handoff
//	规划层：plan, replan
//	执行层：delta, tool_call, tool_result, tool_approval
//	终止层：done, error
//
// 注意：thinking_delta 为内部专用事件，不在白名单内。
var publicEventNames = map[string]struct{}{
	"meta":          {},
	"agent_handoff": {},
	"plan":          {},
	"replan":        {},
	"delta":         {},
	"tool_call":     {},
	"tool_result":   {},
	"tool_approval": {},
	"done":          {},
	"error":         {},
}

// EncodePublicEvent 将事件名称和载荷序列化为 JSON 格式的 StreamEvent。
//
// 参数：
//   - event: 事件名称，必须在 publicEventNames 白名单内
//   - data:  事件载荷，任意可序列化类型
//
// 返回：序列化后的 JSON 字节切片；若事件名称不在白名单内则返回错误。
//
// 副作用：无。调用方需自行将返回字节写入 SSE 响应流。
func EncodePublicEvent(event string, data any) ([]byte, error) {
	if _, ok := publicEventNames[event]; !ok {
		// 拒绝未登记的事件名，防止内部事件意外泄露到 SSE 流
		return nil, fmt.Errorf("unsupported public event %q", event)
	}
	return json.Marshal(StreamEvent{
		Event: event,
		Data:  data,
	})
}
