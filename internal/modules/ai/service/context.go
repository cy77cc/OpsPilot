package service

import "strings"

// Input represents layered AI context input.
type Input struct {
	Instruction      string
	SessionMemory    string
	TaskMemory       string
	RunScratchpad    string
	ArtifactExcerpts []string
}

// Assembler builds layered context slices in deterministic order.
type Assembler struct{}

func NewAssembler() *Assembler {
	return &Assembler{}
}

// Assemble returns non-empty context layers in canonical order:
// instruction -> session -> task -> scratchpad -> artifacts.
func (a *Assembler) Assemble(input Input) []string {
	layers := make([]string, 0, 4+len(input.ArtifactExcerpts))
	appendIfNotBlank := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			layers = append(layers, trimmed)
		}
	}

	appendIfNotBlank(input.Instruction)
	appendIfNotBlank(input.SessionMemory)
	appendIfNotBlank(input.TaskMemory)
	appendIfNotBlank(input.RunScratchpad)
	for _, excerpt := range input.ArtifactExcerpts {
		appendIfNotBlank(excerpt)
	}

	return layers
}

func (a *Assembler) Join(input Input, separator string) string {
	layers := a.Assemble(input)
	return strings.Join(layers, separator)
}
