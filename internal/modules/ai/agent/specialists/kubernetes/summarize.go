package kubernetes

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func BuildKubernetesSummary(workerSummary contracts.DelegationSummary, cluster string, namespace string) contracts.DelegationSummary {
	summaryText := strings.TrimSpace(workerSummary.Summary)
	if summaryText == "" {
		switch workerSummary.Status {
		case contracts.StatusFailed:
			summaryText = "Kubernetes specialist could not complete workload diagnostics for the requested scope."
		default:
			summaryText = "Kubernetes specialist completed workload diagnostics for the requested scope."
		}
	}

	nextAction := strings.TrimSpace(workerSummary.RecommendedNextAction)
	if nextAction == "" {
		switch workerSummary.Status {
		case contracts.StatusFailed:
			nextAction = "Ask deep_main to narrow cluster scope or inspect error details before retrying."
		default:
			nextAction = "Ask deep_main whether to continue read-only verification or prepare a governed remediation."
		}
	}

	findings := append([]string{}, workerSummary.KeyFindings...)
	if c := strings.TrimSpace(cluster); c != "" {
		findings = append([]string{"cluster: " + c}, findings...)
	}
	if ns := strings.TrimSpace(namespace); ns != "" {
		findings = append([]string{"namespace: " + ns}, findings...)
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
