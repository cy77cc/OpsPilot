package chat

import (
	"testing"

	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
)

func TestBuildDelegationPayload_IsolationWorkerWrapsIntoMonitorSummary(t *testing.T) {
	t.Parallel()

	payload := buildDelegationPayload(delegationWindow{
		DelegationID: "delegation-1",
		AgentName:    "isolation_worker",
		Intent:       "metric_anomaly_summary",
		Summary:      "worker summarized p95 increase",
	}, "medium")

	if got := payload["agent_name"]; got != "monitor" {
		t.Fatalf("expected monitor agent name, got %#v", got)
	}
	if got := payload["status"]; got != "returned" {
		t.Fatalf("expected returned status, got %#v", got)
	}
	if got := payload["summary"]; got != "worker summarized p95 increase" {
		t.Fatalf("expected wrapped summary to preserve worker text, got %#v", got)
	}
	if got := payload["risk_level"]; got != "medium" {
		t.Fatalf("expected risk level to be preserved, got %#v", got)
	}
}

func TestBuildDelegationPayload_HostDelegationUsesHostSpecialistSummary(t *testing.T) {
	t.Parallel()

	payload := buildDelegationPayload(delegationWindow{
		DelegationID: "delegation-host-1",
		AgentName:    "host",
		Summary:      "",
	}, "low")

	if got := payload["agent_name"]; got != "host" {
		t.Fatalf("expected host agent name, got %#v", got)
	}
	if got := payload["status"]; got != "returned" {
		t.Fatalf("expected returned status, got %#v", got)
	}
	if got := payload["summary"]; got == "" {
		t.Fatalf("expected non-empty host summary, got %#v", payload)
	}
}

func TestDelegationStreamState_PrefersStructuredMonitorMetricSummaryOverDelta(t *testing.T) {
	t.Parallel()

	state := &delegationStreamState{
		active: &delegationWindow{
			DelegationID: "delegation-1",
			AgentName:    "monitor",
			Intent:       "metric_anomaly_summary",
		},
	}

	state.observe([]airuntime.PublicStreamEvent{{
		Event: "tool_result",
		Data: map[string]any{
			"agent":     "isolation_worker",
			"tool_name": "monitor_metric",
			"content":   `{"query":"http_request_duration_seconds","time_range":"24h","count":3,"points":[{"value":0.18},{"value":0.21},{"value":1.60}]}`,
			"status":    "done",
		},
	}})
	state.observe([]airuntime.PublicStreamEvent{{
		Event: "delta",
		Data: map[string]any{
			"agent":   "isolation_worker",
			"content": "free-form assistant text that should not override structured reduction",
		},
	}})

	windows := state.windowsForEmit()
	if len(windows) != 1 {
		t.Fatalf("expected one delegation window, got %#v", windows)
	}

	payload := buildDelegationPayload(windows[0], "medium")
	summary, _ := payload["summary"].(string)
	if summary == "" {
		t.Fatalf("expected non-empty summary, got %#v", payload)
	}
	if summary == "free-form assistant text that should not override structured reduction" {
		t.Fatalf("expected structured tool summary to win over delta, got %#v", payload)
	}
	if got := payload["agent_name"]; got != "monitor" {
		t.Fatalf("expected monitor agent name, got %#v", got)
	}
}
