package isolation

import "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"

func SummarizeMetricPoints(metric string, points []float64) contracts.DelegationSummary {
	return ReduceMetricPoints(metric, points)
}
