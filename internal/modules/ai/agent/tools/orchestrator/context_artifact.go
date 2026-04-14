package orchestrator

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

const (
	artifactModeInline   = "inline"
	artifactModeArtifact = "artifact"
)

type contextAssembleInput struct {
	Instruction      string
	SessionMemory    string
	TaskMemory       string
	RunScratchpad    string
	ArtifactExcerpts []string
}

type contextAssembler struct{}

func newContextAssembler() contextAssembler {
	return contextAssembler{}
}

func (contextAssembler) Assemble(input contextAssembleInput) []string {
	layers := make([]string, 0, 5+len(input.ArtifactExcerpts))
	appendLayer := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			layers = append(layers, value)
		}
	}
	appendLayer(input.Instruction)
	appendLayer(input.SessionMemory)
	appendLayer(input.TaskMemory)
	appendLayer(input.RunScratchpad)
	for _, excerpt := range input.ArtifactExcerpts {
		appendLayer(excerpt)
	}
	return layers
}

type artifactReference struct {
	Mode       string
	Summary    string
	ArtifactID string
	Content    string
}

type artifactService struct {
	maxInlineChars int
}

func newArtifactService(maxInlineChars int) artifactService {
	if maxInlineChars <= 0 {
		maxInlineChars = 512
	}
	return artifactService{maxInlineChars: maxInlineChars}
}

func (s artifactService) BuildReference(content, preferredID string) artifactReference {
	content = strings.TrimSpace(content)
	if content == "" {
		return artifactReference{Mode: artifactModeInline}
	}
	if len([]rune(content)) <= s.maxInlineChars {
		return artifactReference{
			Mode:    artifactModeInline,
			Summary: content,
			Content: content,
		}
	}
	artifactID := strings.TrimSpace(preferredID)
	if artifactID == "" {
		sum := sha1.Sum([]byte(content))
		artifactID = "artifact_" + hex.EncodeToString(sum[:8])
	}
	return artifactReference{
		Mode:       artifactModeArtifact,
		Summary:    truncateText(content, 240),
		ArtifactID: artifactID,
	}
}
