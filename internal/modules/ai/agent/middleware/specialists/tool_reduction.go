package specialists

import "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/middleware/shared"

const monitorMetricPointApproxBytes = 32

// ShouldReduceMonitorMetric reports whether monitor metric output should be reduced/offloaded.
func ShouldReduceMonitorMetric(pointCount int) bool {
	return monitorMetricNeedsArtifact(pointCount)
}

func monitorMetricNeedsArtifact(pointCount int) bool {
	return shared.ShouldOffloadResult("summary_plus_artifact", approximateMetricBytes(pointCount))
}

func approximateMetricBytes(pointCount int) int {
	if pointCount <= 0 {
		return 0
	}

	maxInt := int(^uint(0) >> 1)
	if pointCount > maxInt/monitorMetricPointApproxBytes {
		return maxInt
	}

	return pointCount * monitorMetricPointApproxBytes
}
