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

	// Replanner 现在支持流式提取 response 字段
	firstReplannerChunk := adk.EventFromMessage(schema.AssistantMessage(`{"response":"## done`, nil), nil, schema.Assistant, "")
	firstReplannerChunk.AgentName = "replanner"

	secondReplannerChunk := adk.EventFromMessage(schema.AssistantMessage(`\n\nall clear"}`, nil), nil, schema.Assistant, "")
	secondReplannerChunk.AgentName = "replanner"

	third := projector.Consume(firstReplannerChunk)
	fourth := projector.Consume(secondReplannerChunk)

	// 第一个 replanner chunk 应该返回 replan + delta（流式提取）
	if len(third) != 2 || third[0].Event != "replan" || third[1].Event != "delta" {
		t.Fatalf("expected first replanner chunk to emit replan+delta, got %#v", third)
	}
	thirdDataMap, ok := third[1].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected third[1].Data to be map[string]any, got %T", third[1].Data)
	}
	thirdData, ok := thirdDataMap["content"].(string)
	if !ok || thirdData != "## done" {
		t.Fatalf("expected first delta content, got %#v", thirdDataMap["content"])
	}

	// 第二个 replanner chunk 应该返回剩余的 delta
	if len(fourth) != 1 || fourth[0].Event != "delta" {
		t.Fatalf("expected second replanner chunk to emit delta, got %#v", fourth)
	}
	fourthDataMap, ok := fourth[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected fourth[0].Data to be map[string]any, got %T", fourth[0].Data)
	}
	fourthData, ok := fourthDataMap["content"].(string)
	if !ok || fourthData != "\n\nall clear" {
		t.Fatalf("expected second delta content, got %#v", fourthDataMap["content"])
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
