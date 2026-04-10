package artifact

import "testing"

func TestBuildReferenceReturnsInlineWhenBelowLimit(t *testing.T) {
	svc := NewService(10)
	result := svc.BuildReference("short", "")

	if result.Mode != ModeInline {
		t.Fatalf("expected inline mode, got %q", result.Mode)
	}
	if result.ArtifactID != "" {
		t.Fatalf("expected no artifact id for inline mode, got %q", result.ArtifactID)
	}
	if result.Content != "short" {
		t.Fatalf("expected inline content, got %q", result.Content)
	}
}

func TestBuildReferenceReturnsArtifactWhenAboveLimit(t *testing.T) {
	svc := NewService(10)
	result := svc.BuildReference("0123456789abcdef", "")

	if result.Mode != ModeArtifact {
		t.Fatalf("expected artifact mode, got %q", result.Mode)
	}
	if result.ArtifactID != "" {
		t.Fatalf("expected no artifact id for non-persistent scaffolding, got %q", result.ArtifactID)
	}
	if len(result.Summary) > 10 {
		t.Fatalf("expected summary length <= 10, got %d (%q)", len(result.Summary), result.Summary)
	}
}
