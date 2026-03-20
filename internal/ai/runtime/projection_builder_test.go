package runtime

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
)

func TestBuildProjection_SplitsExecutorContentOnToolCall(t *testing.T) {
	events := buildProjectionTestEvents(t, []eventFixture{
		{id: "evt-1", eventType: EventTypeDelta, payload: &DeltaPayload{Agent: "executor", Content: "first part "}},
		{id: "evt-2", eventType: EventTypeToolCall, payload: &ToolCallPayload{Agent: "executor", CallID: "call-1", ToolName: "host_list_inventory", Arguments: map[string]any{"keyword": "volcano"}}},
		{id: "evt-3", eventType: EventTypeDelta, payload: &DeltaPayload{Agent: "executor", Content: "second part"}},
	})

	projection, contents, err := BuildProjection(events)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	if len(projection.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(projection.Blocks))
	}
	items := projection.Blocks[0].Items
	if len(items) != 3 {
		t.Fatalf("expected 3 executor items, got %d", len(items))
	}
	if items[0].Type != "content" || items[1].Type != "tool_call" || items[2].Type != "content" {
		t.Fatalf("unexpected item order: %#v", items)
	}
	if len(contents) != 3 {
		t.Fatalf("expected 3 contents, got %d", len(contents))
	}
}

func TestBuildProjection_NestsToolResultUnderToolCall(t *testing.T) {
	events := buildProjectionTestEvents(t, []eventFixture{
		{id: "evt-1", eventType: EventTypeDelta, payload: &DeltaPayload{Agent: "executor", Content: "start"}},
		{id: "evt-2", eventType: EventTypeToolCall, payload: &ToolCallPayload{Agent: "executor", CallID: "call-1", ToolName: "host_list_inventory", Arguments: map[string]any{"keyword": "volcano"}}},
		{id: "evt-3", eventType: EventTypeToolResult, payload: &ToolResultPayload{Agent: "executor", CallID: "call-1", ToolName: "host_list_inventory", Content: "{\"total\":0}", Status: "done"}},
	})

	projection, _, err := BuildProjection(events)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	result := projection.Blocks[0].Items[1].Result
	if result == nil {
		t.Fatal("expected nested tool result")
	}
	if result.Status != "done" || result.ResultContentID == "" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestBuildProjection_InlinesSummary(t *testing.T) {
	events := buildProjectionTestEvents(t, []eventFixture{
		{id: "evt-1", eventType: EventTypeDone, payload: &DonePayload{RunID: "run-1", Status: "completed", Summary: "final answer"}},
	})

	projection, _, err := BuildProjection(events)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	if projection.Summary == nil || projection.Summary.Content != "final answer" {
		t.Fatalf("unexpected summary: %#v", projection.Summary)
	}
}

func TestBuildProjection_MarksNonSteadyStatus(t *testing.T) {
	events := buildProjectionTestEvents(t, []eventFixture{
		{id: "evt-1", eventType: EventTypeError, payload: &ErrorPayload{Message: "runtime failure", Code: "runtime_failed"}},
	})

	projection, _, err := BuildProjection(events)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	if projection.Status != "failed_runtime" {
		t.Fatalf("expected failed_runtime status, got %q", projection.Status)
	}
}

func TestBuildProjection_ProjectsReplanBlock(t *testing.T) {
	events := buildProjectionTestEvents(t, []eventFixture{
		{id: "evt-1", eventType: EventTypePlan, payload: &PlanPayload{Iteration: 0, Steps: []string{"inspect pods"}}},
		{id: "evt-2", eventType: EventTypeReplan, payload: &ReplanPayload{Iteration: 1, Completed: 1, Steps: []string{"inspect nodes", "collect logs"}}},
	})

	projection, _, err := BuildProjection(events)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	if len(projection.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(projection.Blocks))
	}
	if projection.Blocks[0].Type != "replan" {
		t.Fatalf("expected final planning block to be replan, got %#v", projection.Blocks[0])
	}
	if len(projection.Blocks[0].Steps) != 2 {
		t.Fatalf("expected replanned steps to be preserved, got %#v", projection.Blocks[0].Steps)
	}
}

func TestBuildProjection_KeepsOnlyLatestReplanSteps(t *testing.T) {
	events := buildProjectionTestEvents(t, []eventFixture{
		{id: "evt-1", eventType: EventTypePlan, payload: &PlanPayload{Iteration: 0, Steps: []string{"inspect pods", "collect events"}}},
		{id: "evt-2", eventType: EventTypeReplan, payload: &ReplanPayload{Iteration: 1, Completed: 0, Steps: []string{"inspect nodes", "collect logs"}}},
		{id: "evt-3", eventType: EventTypeReplan, payload: &ReplanPayload{Iteration: 2, Completed: 1, Steps: []string{"describe pending pods"}}},
	})

	projection, _, err := BuildProjection(events)
	if err != nil {
		t.Fatalf("build projection: %v", err)
	}
	if len(projection.Blocks) != 1 {
		t.Fatalf("expected only the latest planning block to remain, got %d blocks", len(projection.Blocks))
	}
	if projection.Blocks[0].Type != "replan" {
		t.Fatalf("expected latest planning block to be replan, got %#v", projection.Blocks[0])
	}
	if len(projection.Blocks[0].Steps) != 1 || projection.Blocks[0].Steps[0] != "describe pending pods" {
		t.Fatalf("expected only latest replanned steps, got %#v", projection.Blocks[0].Steps)
	}
}

type eventFixture struct {
	id        string
	eventType EventType
	payload   any
}

func buildProjectionTestEvents(t *testing.T, fixtures []eventFixture) []model.AIRunEvent {
	t.Helper()

	events := make([]model.AIRunEvent, 0, len(fixtures))
	for index, fixture := range fixtures {
		raw, err := MarshalEventPayload(fixture.eventType, fixture.payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		events = append(events, model.AIRunEvent{
			ID:          fixture.id,
			RunID:       "run-1",
			SessionID:   "sess-1",
			Seq:         index + 1,
			EventType:   string(fixture.eventType),
			PayloadJSON: raw,
		})
	}
	return events
}
