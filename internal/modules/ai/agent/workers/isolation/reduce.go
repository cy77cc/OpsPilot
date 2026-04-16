package isolation

import (
	"fmt"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func ReduceMetricPoints(metric string, points []float64) contracts.DelegationSummary {
	if len(points) == 0 {
		return contracts.DelegationSummary{
			TaskID:     "empty",
			AgentName:  "isolation_worker",
			Status:     contracts.StatusReturned,
			Summary:    "No metric samples were returned.",
			Confidence: "low",
		}
	}

	peak := points[0]
	for _, point := range points[1:] {
		if point > peak {
			peak = point
		}
	}

	return contracts.DelegationSummary{
		TaskID:    "reduced",
		AgentName: "isolation_worker",
		Status:    contracts.StatusReturned,
		Summary:   fmt.Sprintf("%s peaked at %.2f during the requested range.", metric, peak),
		KeyFindings: []string{
			fmt.Sprintf("peak value %.2f", peak),
			fmt.Sprintf("%d samples reduced to a compact summary", len(points)),
		},
		RiskLevel:  contracts.RiskMedium,
		Confidence: "medium",
		Metrics: map[string]any{
			"peak":         peak,
			"sample_count": len(points),
		},
	}
}
