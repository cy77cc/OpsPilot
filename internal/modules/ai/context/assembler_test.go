package context

import "testing"

func TestAssemblerOrdersLayersDeterministically(t *testing.T) {
	assembler := NewAssembler()
	layers := assembler.Assemble(Input{
		Instruction:      "sys",
		SessionMemory:    "session",
		TaskMemory:       "task",
		RunScratchpad:    "scratch",
		ArtifactExcerpts: []string{"artifact-a", "artifact-b"},
	})

	if len(layers) != 6 {
		t.Fatalf("expected 6 layers, got %d (%#v)", len(layers), layers)
	}
	if layers[0] != "sys" || layers[1] != "session" || layers[2] != "task" || layers[3] != "scratch" {
		t.Fatalf("unexpected primary layer order: %#v", layers)
	}
	if layers[4] != "artifact-a" || layers[5] != "artifact-b" {
		t.Fatalf("unexpected artifact ordering: %#v", layers)
	}
}

func TestAssemblerOmitsBlankLayers(t *testing.T) {
	assembler := NewAssembler()
	layers := assembler.Assemble(Input{
		Instruction:      "  ",
		SessionMemory:    "session",
		TaskMemory:       "",
		RunScratchpad:    "   scratch   ",
		ArtifactExcerpts: []string{"", " artifact "},
	})

	if len(layers) != 3 {
		t.Fatalf("expected 3 non-blank layers, got %d (%#v)", len(layers), layers)
	}
	if layers[0] != "session" || layers[1] != "scratch" || layers[2] != "artifact" {
		t.Fatalf("unexpected layer values: %#v", layers)
	}
}
