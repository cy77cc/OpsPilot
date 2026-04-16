package host

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func BuildHostSummary(workerSummary contracts.DelegationSummary, hostID string) contracts.DelegationSummary {
	summaryText := strings.TrimSpace(workerSummary.Summary)
	if summaryText == "" {
		switch workerSummary.Status {
		case contracts.StatusFailed:
			summaryText = "Host specialist could not complete runtime diagnostics for the requested host scope."
		default:
			summaryText = "Host specialist completed runtime diagnostics for the requested host scope."
		}
	}

	nextAction := strings.TrimSpace(workerSummary.RecommendedNextAction)
	if nextAction == "" {
		switch workerSummary.Status {
		case contracts.StatusFailed:
			nextAction = "Ask deep_main to retry with narrower host checks or inspect failure details."
		default:
			nextAction = "Ask deep_main whether to continue read-only diagnostics or prepare a governed host action."
		}
	}

	findings := append([]string{}, workerSummary.KeyFindings...)
	if id := strings.TrimSpace(hostID); id != "" {
		findings = append([]string{"host_id: " + id}, findings...)
	}

	return contracts.DelegationSummary{
		TaskID:                strings.TrimSpace(workerSummary.TaskID),
		AgentName:             name,
		Status:                workerSummary.Status,
		Summary:               summaryText,
		KeyFindings:           findings,
		RiskLevel:             workerSummary.RiskLevel,
		Confidence:            workerSummary.Confidence,
		RecommendedNextAction: nextAction,
		ArtifactRefs:          workerSummary.ArtifactRefs,
		Metrics:               workerSummary.Metrics,
	}
}
