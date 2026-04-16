package monitor

import "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"

func BuildMonitorSummary(workerSummary contracts.DelegationSummary, service string, timeRange string) contracts.DelegationSummary {
	return contracts.DelegationSummary{
		TaskID:    workerSummary.TaskID,
		AgentName: "monitor",
		Status:    contracts.StatusReturned,
		Summary:   "MonitorAgent found a latency anomaly in the requested scope.",
		KeyFindings: append([]string{
			"service: " + service,
			"time_range: " + timeRange,
		}, workerSummary.KeyFindings...),
		RiskLevel:             workerSummary.RiskLevel,
		Confidence:            workerSummary.Confidence,
		RecommendedNextAction: "Ask the supervisor whether to continue with read-only diagnosis or prepare a governed action.",
		ArtifactRefs:          workerSummary.ArtifactRefs,
		Metrics:               workerSummary.Metrics,
	}
}
