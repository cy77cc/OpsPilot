package monitor

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func BuildMonitorSummary(workerSummary contracts.DelegationSummary, service string, timeRange string) contracts.DelegationSummary {
	summaryText := strings.TrimSpace(workerSummary.Summary)
	if summaryText == "" {
		switch workerSummary.Status {
		case contracts.StatusFailed:
			summaryText = "MonitorAgent could not complete metric reduction for the requested scope."
		default:
			summaryText = "MonitorAgent completed metric reduction for the requested scope."
		}
	}

	nextAction := workerSummary.RecommendedNextAction
	if strings.TrimSpace(nextAction) == "" {
		if workerSummary.Status == contracts.StatusFailed {
			nextAction = "Ask deep_main to inspect failure details or retry with a narrower diagnostic scope."
		} else {
			nextAction = "Ask deep_main whether to continue with read-only diagnosis or prepare a governed action."
		}
	}

	return contracts.DelegationSummary{
		TaskID:    workerSummary.TaskID,
		AgentName: "monitor",
		Status:    workerSummary.Status,
		Summary:   summaryText,
		KeyFindings: append([]string{
			"service: " + service,
			"time_range: " + timeRange,
		}, workerSummary.KeyFindings...),
		RiskLevel:             workerSummary.RiskLevel,
		Confidence:            workerSummary.Confidence,
		RecommendedNextAction: nextAction,
		ArtifactRefs:          workerSummary.ArtifactRefs,
		Metrics:               workerSummary.Metrics,
	}
}
