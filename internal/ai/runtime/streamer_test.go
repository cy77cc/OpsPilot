package runtime

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestProjectAssistantMessage_FinalReplanPreservesCurrentSteps(t *testing.T) {
	t.Parallel()

	state := &a2uiProjectionState{}

	planEvents := projectAssistantMessage("planner", schema.AssistantMessage(`{"steps":["step one","step two"]}`, nil), state)
	if len(planEvents) != 1 || planEvents[0].Event != "plan" {
		t.Fatalf("expected plan event, got %#v", planEvents)
	}

	finalEvents := projectAssistantMessage("replanner", schema.AssistantMessage(`{"response":"done"}`, nil), state)
	if len(finalEvents) != 2 || finalEvents[0].Event != "replan" {
		t.Fatalf("expected final replan + delta, got %#v", finalEvents)
	}

	data, ok := finalEvents[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", finalEvents[0].Data)
	}

	steps, ok := data["steps"].([]string)
	if !ok || len(steps) != 2 || steps[0] != "step one" || steps[1] != "step two" {
		t.Fatalf("expected final replan to preserve current steps, got %#v", data["steps"])
	}
}
