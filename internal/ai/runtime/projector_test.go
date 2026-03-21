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
	replanData, ok := second[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected replan data map, got %T", second[0].Data)
	}
	steps, ok := replanData["steps"].([]string)
	if !ok || len(steps) != 1 || steps[0] != "inspect pods" {
		t.Fatalf("expected final replan to keep only first phase step, got %#v", replanData["steps"])
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
	thirdDataMap, ok := third[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected replan data map, got %T", third[0].Data)
	}
	thirdSteps, ok := thirdDataMap["steps"].([]string)
	if !ok || len(thirdSteps) != 1 || thirdSteps[0] != "inspect pods" {
		t.Fatalf("expected buffered final replan to keep only first phase step, got %#v", thirdDataMap["steps"])
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
	replanData, ok := third[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected replan data map, got %T", third[0].Data)
	}
	replanSteps, ok := replanData["steps"].([]string)
	if !ok || len(replanSteps) != 1 || replanSteps[0] != "step1" {
		t.Fatalf("expected large final replan to preserve plan steps, got %#v", replanData["steps"])
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

func TestProjectorEmitsToolResultErrorWithoutFatalError(t *testing.T) {
	t.Parallel()

	projector := NewStreamProjector()
	callEvent := adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
		{ID: "call-err", Function: schema.FunctionCall{Name: "kubectl_get_pods", Arguments: `{"namespace":"default"}`}},
	}), nil, schema.Assistant, "")
	callEvent.AgentName = "executor"
	projector.Consume(callEvent)

	message := schema.ToolMessage(`{"status":"error","message":"tool failed"}`, "call-err", schema.WithToolName("kubectl_get_pods"))
	resultEvent := adk.EventFromMessage(message, nil, schema.Tool, message.ToolName)
	resultEvent.AgentName = "executor"

	got := projector.Consume(resultEvent)

	if len(got) != 1 {
		t.Fatalf("expected one projected event, got %#v", got)
	}
	if got[0].Event != "tool_result" {
		t.Fatalf("expected tool_result event, got %#v", got[0])
	}

	data, ok := got[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected tool_result data to be a map, got %T", got[0].Data)
	}
	if data["content"] == "" {
		t.Fatal("expected tool_result content to be preserved")
	}

	persisted := projector.GetPersistedState()
	if len(persisted.Activities) != 2 {
		t.Fatalf("expected tool_call + tool_result persisted activities, got %#v", persisted.Activities)
	}
	if persisted.Activities[0].Kind != "tool_call" || persisted.Activities[0].Status != "error" {
		t.Fatalf("expected tool call activity status error, got %#v", persisted.Activities[0])
	}
	if persisted.Activities[1].Kind != "tool_result" || persisted.Activities[1].Status != "error" {
		t.Fatalf("expected tool result activity status error, got %#v", persisted.Activities[1])
	}
}

func TestStreamProjector_MergesStreamingToolCallChunksIntoOneEvent(t *testing.T) {
	t.Parallel()

	projector := NewStreamProjector()

	first := adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
		{ID: "call-1", Function: schema.FunctionCall{Name: "os_get_net_stat", Arguments: ""}},
	}), nil, schema.Assistant, "")
	first.AgentName = "executor"

	second := adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
		{Function: schema.FunctionCall{Arguments: `{"target": `}},
	}), nil, schema.Assistant, "")
	second.AgentName = "executor"

	third := adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
		{Function: schema.FunctionCall{Arguments: `"2"`}},
	}), nil, schema.Assistant, "")
	third.AgentName = "executor"

	fourth := adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
		{Function: schema.FunctionCall{Arguments: `}`}},
	}), nil, schema.Assistant, "")
	fourth.AgentName = "executor"

	got1 := projector.Consume(first)
	got2 := projector.Consume(second)
	got3 := projector.Consume(third)
	got4 := projector.Consume(fourth)

	if len(got1) != 0 || len(got2) != 0 || len(got3) != 0 {
		t.Fatalf("expected partial tool call chunks to stay buffered, got %#v %#v %#v", got1, got2, got3)
	}
	if len(got4) != 1 || got4[0].Event != "tool_call" {
		t.Fatalf("expected one merged tool_call event, got %#v", got4)
	}

	data, ok := got4[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected tool_call data to be a map, got %T", got4[0].Data)
	}
	if data["call_id"] != "call-1" || data["tool_name"] != "os_get_net_stat" {
		t.Fatalf("unexpected merged tool call identity: %#v", data)
	}
	args, ok := data["arguments"].(map[string]any)
	if !ok {
		t.Fatalf("expected merged tool arguments to be a map, got %T", data["arguments"])
	}
	if args["target"] != "2" {
		t.Fatalf("expected merged target argument, got %#v", args)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
