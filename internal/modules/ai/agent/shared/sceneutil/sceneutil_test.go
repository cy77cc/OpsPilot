package sceneutil

import "testing"

func TestNormalizeScene(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "ai"},
		{"  ", "ai"},
		{"Kubernetes", "kubernetes"},
		{"  MONITORING  ", "monitoring"},
		{"ai", "ai"},
	}
	for _, tt := range tests {
		got := NormalizeScene(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeScene(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAllowedToolSet(t *testing.T) {
	set := NewAllowedToolSet([]string{"tool_a", "tool_b", "tool_c"})

	if !set.IsAllowed("tool_a") {
		t.Error("expected tool_a to be allowed")
	}
	if set.IsAllowed("tool_x") {
		t.Error("expected tool_x to be disallowed")
	}
	if set.Len() != 3 {
		t.Errorf("expected len 3, got %d", set.Len())
	}
}

func TestAllowedToolSet_Empty(t *testing.T) {
	set := NewAllowedToolSet(nil)
	if set.IsAllowed("anything") {
		t.Error("empty set should disallow everything")
	}
	if set.Len() != 0 {
		t.Errorf("expected len 0, got %d", set.Len())
	}
}
