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
