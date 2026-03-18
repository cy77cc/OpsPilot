package runtime

import (
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func TestStreamProjector_ConsumeTracksPlanAndReplanIterations(t *testing.T) {
	t.Parallel()

	projector := NewStreamProjector()

	plannerEvent := adk.EventFromMessage(schema.AssistantMessage(`{"steps":["inspect pods","check events"]}`, nil), nil, schema.Assistant, "")
	plannerEvent.AgentName = "planner"

	replannerEvent := adk.EventFromMessage(schema.AssistantMessage(`{"response":"root cause found"}`, nil), nil, schema.Assistant, "")
	replannerEvent.AgentName = "replanner"

	first := projector.Consume(plannerEvent)
	second := projector.Consume(replannerEvent)

	if len(first) != 1 || first[0].Event != "plan" {
		t.Fatalf("expected plan output, got %#v", first)
	}
	// 第二个事件应该返回 replan + delta（因为 response 内容为 17 字符，虽然小于 50，
	// 但引号关闭时会强制刷新缓冲区）
	if len(second) != 2 || second[0].Event != "replan" || second[1].Event != "delta" {
		t.Fatalf("expected final replan output, got %#v", second)
	}
}

func TestStreamProjector_BuffersStreamingPlannerAndReplannerJSON(t *testing.T) {
	t.Parallel()

	projector := NewStreamProjector()

	firstPlannerChunk := adk.EventFromMessage(schema.AssistantMessage(`{"steps":["inspect pods",`, nil), nil, schema.Assistant, "")
	firstPlannerChunk.AgentName = "planner"

	secondPlannerChunk := adk.EventFromMessage(schema.AssistantMessage(`"check events"]}`, nil), nil, schema.Assistant, "")
	secondPlannerChunk.AgentName = "planner"

	first := projector.Consume(firstPlannerChunk)
	second := projector.Consume(secondPlannerChunk)

	if len(first) != 0 {
		t.Fatalf("expected no public events for partial planner chunk, got %#v", first)
	}
	if len(second) != 1 || second[0].Event != "plan" {
		t.Fatalf("expected buffered planner chunk to emit one plan event, got %#v", second)
	}

	// Replanner 现在支持流式提取 response 字段，并有缓冲机制
	// 第一个 chunk 内容只有 7 字符，会被缓冲
	firstReplannerChunk := adk.EventFromMessage(schema.AssistantMessage(`{"response":"## done`, nil), nil, schema.Assistant, "")
	firstReplannerChunk.AgentName = "replanner"

	// 第二个 chunk 添加更多内容并关闭引号
	// 注意：这里的 \n 是字面上的反斜杠和 n，不是换行符
	secondReplannerChunk := adk.EventFromMessage(schema.AssistantMessage(`\n\nall clear"}`, nil), nil, schema.Assistant, "")
	secondReplannerChunk.AgentName = "replanner"

	third := projector.Consume(firstReplannerChunk)
	fourth := projector.Consume(secondReplannerChunk)

	// 第一个 replanner chunk：内容被缓冲（只有 7 字符），只发送 replan 事件
	if len(third) != 1 || third[0].Event != "replan" {
		t.Fatalf("expected first replanner chunk to emit only replan (content buffered), got %#v", third)
	}

	// 第二个 replanner chunk：引号关闭，刷新缓冲区，发送 delta
	if len(fourth) != 1 || fourth[0].Event != "delta" {
		t.Fatalf("expected second replanner chunk to emit delta (buffer flushed on quote close), got %#v", fourth)
	}
	fourthDataMap, ok := fourth[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected fourth[0].Data to be map[string]any, got %T", fourth[0].Data)
	}
	fourthData, ok := fourthDataMap["content"].(string)
	// JSON 字符串中的 \n 应该被解析为真正的换行符
	expectedContent := "## done\n\nall clear"
	if !ok || fourthData != expectedContent {
		t.Fatalf("expected delta content %q, got %#v", expectedContent, fourthDataMap["content"])
	}
}

func TestStreamProjector_ReplannerBufferFlushOnLargeContent(t *testing.T) {
	t.Parallel()

	projector := NewStreamProjector()

	// 先发送 plan
	plannerEvent := adk.EventFromMessage(schema.AssistantMessage(`{"steps":["step1"]}`, nil), nil, schema.Assistant, "")
	plannerEvent.AgentName = "planner"
	projector.Consume(plannerEvent)

	// 发送一个超过 50 字符的 response 内容（但引号未关闭）
	largeContent := "This is a large content that exceeds fifty characters threshold for buffering!!!"
	firstReplannerChunk := adk.EventFromMessage(schema.AssistantMessage(`{"response":"`+largeContent, nil), nil, schema.Assistant, "")
	firstReplannerChunk.AgentName = "replanner"

	third := projector.Consume(firstReplannerChunk)

	// 因为内容超过 50 字符，应该立即发送 replan + delta
	if len(third) != 2 || third[0].Event != "replan" || third[1].Event != "delta" {
		t.Fatalf("expected replan+delta for large content, got %#v", third)
	}

	dataMap, ok := third[1].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected delta data to be map[string]any, got %T", third[1].Data)
	}
	content, ok := dataMap["content"].(string)
	if !ok || content != largeContent {
		t.Fatalf("expected content %q, got %#v", largeContent, dataMap["content"])
	}
}

func TestStreamProjector_FinishAndFail(t *testing.T) {
	t.Parallel()

	projector := NewStreamProjector()

	done := projector.Finish("run-1")
	if done.Event != "done" {
		t.Fatalf("expected done event, got %#v", done)
	}

	failed := projector.Fail("run-1", assertErr("boom"))
	if failed.Event != "error" {
		t.Fatalf("expected error event, got %#v", failed)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
