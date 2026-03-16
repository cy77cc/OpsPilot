package ai

import (
	"encoding/json"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/testutil"
)

func TestStreamEvent_UsesPhase1PublicEventNames(t *testing.T) {
	expected := []StreamEventName{
		StreamEventInit,
		StreamEventIntent,
		StreamEventStatus,
		StreamEventDelta,
		StreamEventProgress,
		StreamEventReportReady,
		StreamEventError,
		StreamEventDone,
	}

	testutil.AssertEqual(t, "init", string(expected[0]))
	testutil.AssertEqual(t, "intent", string(expected[1]))
	testutil.AssertEqual(t, "status", string(expected[2]))
	testutil.AssertEqual(t, "delta", string(expected[3]))
	testutil.AssertEqual(t, "progress", string(expected[4]))
	testutil.AssertEqual(t, "report_ready", string(expected[5]))
	testutil.AssertEqual(t, "error", string(expected[6]))
	testutil.AssertEqual(t, "done", string(expected[7]))
}

func TestStreamEvent_JSONEnvelope(t *testing.T) {
	event := StreamEvent{
		Event: StreamEventStatus,
		Data: map[string]any{
			"run_id": "run-1",
			"status": "running",
		},
	}

	raw, err := json.Marshal(event)
	testutil.RequireNoError(t, err)

	var decoded map[string]any
	testutil.RequireNoError(t, json.Unmarshal(raw, &decoded))
	testutil.AssertEqual(t, "status", decoded["event"].(string))
}
