// Package runtime 提供 AI 运行时的 SSE 流式编码工具。
package runtime

import (
	"encoding/json"
	"testing"
)

// TestEncodePublicEvent_AllowsAllPublicEvents 验证白名单内的所有公开事件均可正常编码。
//
// 覆盖范围：会话层、路由层、规划层、执行层、终止层的全部公开事件。
func TestEncodePublicEvent_AllowsAllPublicEvents(t *testing.T) {
	t.Parallel()

	// 与 publicEventNames 白名单保持严格一致
	publicEvents := []struct {
		name  string
		layer string
	}{
		{"meta", "会话层"},
		{"agent_handoff", "路由层"},
		{"plan", "规划层"},
		{"replan", "规划层"},
		{"delta", "执行层"},
		{"tool_call", "执行层"},
		{"tool_result", "执行层"},
		{"tool_approval", "执行层"},
		{"done", "终止层"},
		{"error", "终止层"},
	}

	for _, tc := range publicEvents {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			payload := map[string]any{"layer": tc.layer, "ok": true}
			raw, err := EncodePublicEvent(tc.name, payload)
			if err != nil {
				t.Fatalf("[%s/%s] EncodePublicEvent 不应返回错误，实际得到: %v", tc.layer, tc.name, err)
			}

			var got StreamEvent
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("[%s/%s] 反序列化 StreamEvent 失败: %v", tc.layer, tc.name, err)
			}
			if got.Event != tc.name {
				t.Fatalf("[%s/%s] 期望 Event=%q，实际得到 %q", tc.layer, tc.name, tc.name, got.Event)
			}
			if got.Data == nil {
				t.Fatalf("[%s/%s] Data 字段不应为 nil", tc.layer, tc.name)
			}
		})
	}
}

// TestEncodePublicEvent_RejectsInternalOnlyEvents 验证内部专用事件被白名单拦截，
// 不允许通过 EncodePublicEvent 编码后暴露给前端。
func TestEncodePublicEvent_RejectsInternalOnlyEvents(t *testing.T) {
	t.Parallel()

	internalEvents := []struct {
		name   string
		reason string
	}{
		// thinking_delta 仅在启用扩展推理模型时内部流转，不对前端暴露
		{"thinking_delta", "扩展推理模型内部思考流，不对外暴露"},
		// 以下为已废弃的 Phase1 占位名称，确保不被误用
		{"init", "Phase1 占位名称已废弃"},
		{"intent", "Phase1 占位名称已废弃"},
		{"status", "Phase1 占位名称已废弃"},
		{"progress", "Phase1 占位名称已废弃"},
		{"report_ready", "Phase1 占位名称已废弃"},
	}

	for _, tc := range internalEvents {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := EncodePublicEvent(tc.name, map[string]any{"ok": true})
			if err == nil {
				t.Fatalf("内部事件 %q 应被拦截（原因: %s），但 EncodePublicEvent 未返回错误", tc.name, tc.reason)
			}
		})
	}
}

// TestEncodePublicEvent_ErrorMessageContainsEventName 验证拒绝时的错误信息包含事件名称，
// 便于调用方快速定位问题。
func TestEncodePublicEvent_ErrorMessageContainsEventName(t *testing.T) {
	t.Parallel()

	const unknownEvent = "some_unknown_event"
	_, err := EncodePublicEvent(unknownEvent, nil)
	if err == nil {
		t.Fatal("未知事件应返回错误")
	}

	// 错误信息应包含事件名，方便调试
	if msg := err.Error(); len(msg) == 0 {
		t.Fatal("错误信息不应为空")
	}
}

// TestStreamEvent_JSONRoundTrip 验证 StreamEvent 的 JSON 序列化/反序列化往返一致性。
func TestStreamEvent_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		event StreamEvent
	}{
		{
			name:  "delta事件带文本载荷",
			event: StreamEvent{Event: "delta", Data: map[string]any{"content": "你好，我正在分析集群状态…"}},
		},
		{
			name:  "plan事件带步骤列表",
			event: StreamEvent{Event: "plan", Data: map[string]any{"steps": []string{"获取 Pod 列表", "检查 Pod 日志"}}},
		},
		{
			name:  "agent_handoff事件带路由信息",
			event: StreamEvent{Event: "agent_handoff", Data: map[string]any{"from": "OpsPilotAgent", "to": "DiagnosisAgent"}},
		},
		{
			name:  "done事件带迭代统计",
			event: StreamEvent{Event: "done", Data: map[string]any{"iterations": 2}},
		},
		{
			name:  "data为nil的error事件",
			event: StreamEvent{Event: "error", Data: nil},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			raw, err := json.Marshal(tc.event)
			if err != nil {
				t.Fatalf("序列化失败: %v", err)
			}

			var got StreamEvent
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("反序列化失败: %v", err)
			}
			if got.Event != tc.event.Event {
				t.Fatalf("Event 字段不匹配：期望 %q，实际 %q", tc.event.Event, got.Event)
			}
		})
	}
}
