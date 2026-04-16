package specialists

import "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/middleware/shared"

const monitorMetricPointApproxBytes = 32

// ShouldReduceMonitorMetric reports whether monitor metric output should be reduced/offloaded.
func ShouldReduceMonitorMetric(pointCount int) bool {
	return monitorMetricNeedsArtifact(pointCount)
}

func monitorMetricNeedsArtifact(pointCount int) bool {
	if pointCount < 0 {
		pointCount = 0
	}
	return shared.ShouldOffloadResult("summary_plus_artifact", pointCount*monitorMetricPointApproxBytes)
}
