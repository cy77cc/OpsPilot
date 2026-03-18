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

	firstReplannerChunk := adk.EventFromMessage(schema.AssistantMessage(`{"response":"## done`, nil), nil, schema.Assistant, "")
	firstReplannerChunk.AgentName = "replanner"

	secondReplannerChunk := adk.EventFromMessage(schema.AssistantMessage(`\n\nall clear"}`, nil), nil, schema.Assistant, "")
	secondReplannerChunk.AgentName = "replanner"

	third := projector.Consume(firstReplannerChunk)
	fourth := projector.Consume(secondReplannerChunk)

	if len(third) != 0 {
		t.Fatalf("expected no public events for partial replanner chunk, got %#v", third)
	}
	if len(fourth) != 2 || fourth[0].Event != "replan" || fourth[1].Event != "delta" {
		t.Fatalf("expected buffered replanner chunk to emit final replan+delta, got %#v", fourth)
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
