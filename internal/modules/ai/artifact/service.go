package artifact

import (
	"strings"
)

const (
	ModeInline   = "inline"
	ModeArtifact = "artifact"
)

type Result struct {
	Mode       string `json:"mode"`
	Summary    string `json:"summary"`
	ArtifactID string `json:"artifact_id,omitempty"`
	Content    string `json:"content,omitempty"`
}

type Service struct {
	inlineLimit int
}

func NewService(inlineLimit int) *Service {
	if inlineLimit <= 0 {
		inlineLimit = 512
	}
	return &Service{inlineLimit: inlineLimit}
}

// BuildReference decides whether content stays inline or should be represented
// as an artifact handle. This is a scaffolding service and does not persist.
func (s *Service) BuildReference(content, preferredArtifactID string) Result {
	normalized := strings.TrimSpace(content)
	if len([]rune(normalized)) <= s.inlineLimit {
		return Result{
			Mode:    ModeInline,
			Summary: normalized,
			Content: normalized,
		}
	}

	summary := normalized
	summaryRunes := []rune(summary)
	if len(summaryRunes) > s.inlineLimit {
		if s.inlineLimit <= len("...") {
			summary = string(summaryRunes[:s.inlineLimit])
		} else {
			summary = string(summaryRunes[:s.inlineLimit-3]) + "..."
		}
	}

	return Result{
		Mode:    ModeArtifact,
		Summary: summary,
		// Scaffolding only: no resolvable artifact handle is emitted until
		// a persistent artifact backend exists.
		ArtifactID: strings.TrimSpace(preferredArtifactID),
	}
}
