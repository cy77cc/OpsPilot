package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBackend_CachesSkills(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: test-skill
description: A test skill
---
Skill content here.
`), 0644)

	backend, err := New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	// First call — loads from disk
	skills1, err := backend.List(context.Background())
	if err != nil {
		t.Fatalf("first List failed: %v", err)
	}
	if len(skills1) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills1))
	}

	// Remove the file — cache should still return it
	os.RemoveAll(skillDir)

	// Second call — should return cached result
	skills2, err := backend.List(context.Background())
	if err != nil {
		t.Fatalf("second List failed: %v", err)
	}
	if len(skills2) != 1 {
		t.Errorf("expected cached 1 skill after file deletion, got %d", len(skills2))
	}
}

func TestBackend_GetFromCache(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: my-skill
description: My skill
---
Content.
`), 0644)

	backend, err := New(tmpDir)
	if err != nil {
		t.Fatalf("failed to create backend: %v", err)
	}

	// Warm the cache
	_, _ = backend.List(context.Background())

	// Remove file, Get should still work
	os.RemoveAll(skillDir)

	s, err := backend.Get(context.Background(), "my-skill")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if s.FrontMatter.Name != "my-skill" {
		t.Errorf("expected name my-skill, got %q", s.FrontMatter.Name)
	}
}
