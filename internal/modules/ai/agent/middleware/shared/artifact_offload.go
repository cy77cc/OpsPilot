package shared

import "strings"

const (
	summaryPlusArtifactMode = "summary_plus_artifact"
	summaryModeMaxBytes     = 256
	inlineModeMaxBytes      = 1024
)

// ShouldOffloadResult decides whether tool output should be persisted as an artifact.
func ShouldOffloadResult(outputMode string, sizeBytes int) bool {
	if sizeBytes < 0 {
		sizeBytes = 0
	}

	if strings.EqualFold(strings.TrimSpace(outputMode), summaryPlusArtifactMode) && sizeBytes > summaryModeMaxBytes {
		return true
	}

	return sizeBytes > inlineModeMaxBytes
}
