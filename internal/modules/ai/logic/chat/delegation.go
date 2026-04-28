package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	contracts "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
	sharedmiddleware "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/middleware/shared"
	workermiddleware "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/middleware/workers"
	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
	cicdspecialist "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/specialists/cicd"
	hostspecialist "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/specialists/host"
	kubernetesspecialist "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/specialists/kubernetes"
	monitorspecialist "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/specialists/monitor"
	isolationworker "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/workers/isolation"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/stream"
	"github.com/google/uuid"
)

type delegationWindow struct {
	DelegationID      string
	AgentName         string
	Intent            string
	Summary           string
	StructuredSummary *contracts.DelegationSummary
}

type delegationStreamState struct {
	active    *delegationWindow
	completed []delegationWindow
}

func (s *delegationStreamState) observe(events []airuntime.PublicStreamEvent) {
	for _, projected := range events {
		switch projected.Event {
		case "agent_handoff":
			data, _ := projected.Data.(map[string]any)
			s.observeHandoff(data)
		case "delta":
			data, _ := projected.Data.(map[string]any)
			s.observeDelta(data)
		case "tool_result":
			data, _ := projected.Data.(map[string]any)
			s.observeToolResult(data)
		}
	}
}

func (s *delegationStreamState) observeHandoff(data map[string]any) {
	if data == nil {
		return
	}
	from := strings.TrimSpace(stream.StringValue(data, "from"))
	to := strings.TrimSpace(stream.StringValue(data, "to"))
	intent := strings.TrimSpace(stream.StringValue(data, "intent"))

	if s.active != nil && isDelegationReturnTarget(to) {
		s.closeActiveWindow()
	}

	if !airuntime.IsDelegationHandoff(from, to, intent) {
		return
	}
	s.closeActiveWindow()
	s.active = &delegationWindow{
		DelegationID: uuid.NewString(),
		AgentName:    to,
		Intent:       intent,
	}
}

func (s *delegationStreamState) observeDelta(data map[string]any) {
	if data == nil || s.active == nil || s.active.StructuredSummary != nil {
		return
	}
	content := stream.StringValue(data, "content")
	if strings.TrimSpace(content) == "" {
		return
	}
	s.active.Summary += content
}

func (s *delegationStreamState) observeToolResult(data map[string]any) {
	if data == nil || s.active == nil || s.active.StructuredSummary != nil {
		return
	}
	if strings.TrimSpace(stream.StringValue(data, "tool_name")) != "monitor_metric" {
		return
	}

	agent := normalizeDelegationAgent(stream.StringValue(data, "agent"))
	if agent != "monitor" && agent != "isolation_worker" {
		return
	}

	summary, ok := buildStructuredMonitorMetricSummary(*s.active, stream.StringValue(data, "content"))
	if !ok {
		return
	}

	s.active.StructuredSummary = &summary
	s.active.AgentName = summary.AgentName
	s.active.Summary = summary.Summary
}

func (s *delegationStreamState) windowsForEmit() []delegationWindow {
	if s == nil {
		return nil
	}
	if s.active != nil {
		s.closeActiveWindow()
	}
	windows := s.completed
	s.completed = nil
	return windows
}

func (s *delegationStreamState) closeActiveWindow() {
	if s == nil || s.active == nil {
		return
	}
	window := *s.active
	s.completed = append(s.completed, window)
	s.active = nil
}

func sameAgentIdentity(left, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func isDelegationReturnTarget(target string) bool {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "executor", "deep_main", "orchestrator":
		return true
	default:
		return false
	}
}

func compactDelegationSummary(summary string) string {
	return strings.TrimSpace(summary)
}

func normalizeDelegationAgent(agent string) string {
	trimmed := strings.TrimSpace(agent)
	if trimmed == "" {
		return "specialist"
	}
	return trimmed
}

func buildDelegationNodeTitle(agent string) string {
	trimmed := strings.TrimSpace(agent)
	if trimmed == "" {
		return "Delegation summary"
	}
	return fmt.Sprintf("%s summary", trimmed)
}

func shouldEmitDelegationWindow(window delegationWindow) bool {
	if strings.TrimSpace(window.DelegationID) == "" {
		return false
	}
	if strings.TrimSpace(window.AgentName) == "" {
		return false
	}
	if strings.TrimSpace(window.Summary) == "" {
		return false
	}
	return true
}

func buildDelegationPayload(window delegationWindow, runRiskLevel string) map[string]any {
	summary := buildDelegationSummary(window, runRiskLevel)
	payload := map[string]any{
		"delegation_id": strings.TrimSpace(window.DelegationID),
		"agent_name":    normalizeDelegationAgent(summary.AgentName),
		"status":        string(summary.Status),
		"title":         buildDelegationNodeTitle(summary.AgentName),
		"summary":       compactDelegationSummary(summary.Summary),
	}
	if intent := strings.TrimSpace(window.Intent); intent != "" {
		payload["intent"] = intent
	}
	if risk := strings.TrimSpace(string(summary.RiskLevel)); risk != "" {
		payload["risk_level"] = risk
	}
	return payload
}

func buildDelegationSummary(window delegationWindow, runRiskLevel string) contracts.DelegationSummary {
	if window.StructuredSummary != nil {
		summary := *window.StructuredSummary
		summary.RiskLevel = firstNonEmptyRiskLevel(summary.RiskLevel, delegationRiskLevel(runRiskLevel))
		return summary
	}

	base := contracts.DelegationSummary{
		TaskID:    strings.TrimSpace(window.DelegationID),
		AgentName: normalizeDelegationAgent(window.AgentName),
		Status:    contracts.StatusReturned,
		Summary:   compactDelegationSummary(window.Summary),
		RiskLevel: delegationRiskLevel(runRiskLevel),
	}

	if strings.EqualFold(base.AgentName, "isolation_worker") {
		base = sharedmiddleware.ApplySummaryDefaults(
			base,
			"Isolation worker completed metric reduction for the requested scope.",
			"Ask the monitor specialist to return a compact read-only summary to deep_main.",
		)
		if err := workermiddleware.ValidateStrictSummary(base); err == nil {
			wrapped := monitorspecialist.BuildMonitorSummary(base, "", "")
			wrapped.RiskLevel = firstNonEmptyRiskLevel(wrapped.RiskLevel, delegationRiskLevel(runRiskLevel))
			return sharedmiddleware.ApplySummaryDefaults(
				wrapped,
				"MonitorAgent completed delegated analysis for the requested scope.",
				"Ask deep_main whether to continue with read-only diagnosis or prepare a governed action.",
			)
		}
	}

	switch normalizeDelegationAgent(base.AgentName) {
	case "monitor":
		base = monitorspecialist.BuildMonitorSummary(base, "", "")
	case "kubernetes":
		base = kubernetesspecialist.BuildKubernetesSummary(base, "", "")
	case "host":
		base = hostspecialist.BuildHostSummary(base, "")
	case "cicd":
		base = cicdspecialist.BuildCICDSummary(base, "")
	}

	return sharedmiddleware.ApplySummaryDefaults(
		base,
		fmt.Sprintf("%s completed delegated analysis for the requested scope.", buildDelegationNodeTitle(base.AgentName)),
		"Ask deep_main whether to continue with read-only diagnosis or prepare a governed action.",
	)
}

func buildStructuredMonitorMetricSummary(window delegationWindow, raw string) (contracts.DelegationSummary, bool) {
	type metricPoint struct {
		Value float64 `json:"value"`
	}
	type metricResult struct {
		Query     string        `json:"query"`
		TimeRange string        `json:"time_range"`
		Points    []metricPoint `json:"points"`
	}

	var payload metricResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &payload); err != nil {
		return contracts.DelegationSummary{}, false
	}
	if strings.TrimSpace(payload.Query) == "" {
		return contracts.DelegationSummary{}, false
	}

	values := make([]float64, 0, len(payload.Points))
	for _, point := range payload.Points {
		values = append(values, point.Value)
	}

	workerSummary := isolationworker.ReduceMetricPoints(strings.TrimSpace(window.DelegationID), payload.Query, values)
	if err := workermiddleware.ValidateStrictSummary(workerSummary); err != nil {
		return contracts.DelegationSummary{}, false
	}

	monitorSummary := monitorspecialist.BuildMonitorSummary(workerSummary, "", payload.TimeRange)
	monitorSummary = sharedmiddleware.ApplySummaryDefaults(
		monitorSummary,
		"MonitorAgent completed delegated metric analysis for the requested scope.",
		"Ask deep_main whether to continue with read-only diagnosis or prepare a governed action.",
	)
	return monitorSummary, true
}

func delegationRiskLevel(value string) contracts.RiskLevel {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(contracts.RiskHigh):
		return contracts.RiskHigh
	case string(contracts.RiskMedium):
		return contracts.RiskMedium
	case string(contracts.RiskLow):
		return contracts.RiskLow
	default:
		return ""
	}
}

func firstNonEmptyRiskLevel(levels ...contracts.RiskLevel) contracts.RiskLevel {
	for _, level := range levels {
		if strings.TrimSpace(string(level)) != "" {
			return level
		}
	}
	return ""
}

func emitDelegationWindows(ctx context.Context, l *Logic, shell ChatShell, state *delegationStreamState, seq *int, emit EventEmitter) error {
	for _, window := range state.windowsForEmit() {
		if !shouldEmitDelegationWindow(window) {
			continue
		}
		payload := buildDelegationPayload(window, shell.Run.RiskLevel)
		eid, err := AppendRunEventWithID(ctx, l, shell.Run.ID, shell.SessionID, seq, "delegation_node", payload)
		if err != nil {
			return err
		}
		emit("delegation_node", withEventID(payload, eid))
	}
	return nil
}
