// Package skill provides a filesystem-based backend for loading Eino ADK skill definitions.
package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloudwego/eino/adk/middlewares/skill"
	"gopkg.in/yaml.v3"
)

const skillFileName = "SKILL.md"

// Backend implements skill.Backend using the OS filesystem.
type Backend struct {
	baseDir string

	cacheMu sync.RWMutex
	cached  []skill.Skill
	loaded  bool
}

// New creates a new filesystem-based skill backend.
// baseDir is the absolute path to the directory containing skill subdirectories.
func New(baseDir string) (*Backend, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("skill baseDir is required")
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("invalid skill baseDir %q: %w", baseDir, err)
	}
	return &Backend{baseDir: abs}, nil
}

// List returns all available skill front matters from the base directory.
// Only first-level subdirectories containing SKILL.md are scanned.
func (b *Backend) List(_ context.Context) ([]skill.FrontMatter, error) {
	skills, err := b.loadAll()
	if err != nil {
		return nil, err
	}
	matters := make([]skill.FrontMatter, 0, len(skills))
	for _, s := range skills {
		matters = append(matters, s.FrontMatter)
	}
	return matters, nil
}

// Get returns a specific skill by name.
func (b *Backend) Get(_ context.Context, name string) (skill.Skill, error) {
	if name == "" {
		return skill.Skill{}, fmt.Errorf("skill name is required")
	}
	skills, err := b.loadAll()
	if err != nil {
		return skill.Skill{}, fmt.Errorf("failed to load skills: %w", err)
	}
	for _, s := range skills {
		if s.Name == name {
			return s, nil
		}
	}
	return skill.Skill{}, fmt.Errorf("skill not found: %s", name)
}

func (b *Backend) loadAll() ([]skill.Skill, error) {
	// Fast path: read from cache
	b.cacheMu.RLock()
	if b.loaded {
		defer b.cacheMu.RUnlock()
		return b.cached, nil
	}
	b.cacheMu.RUnlock()

	// Slow path: load from disk
	b.cacheMu.Lock()
	defer b.cacheMu.Unlock()

	// Double-check after acquiring write lock
	if b.loaded {
		return b.cached, nil
	}

	skills, err := b.loadAllFromDisk()
	if err != nil {
		return nil, err
	}
	b.cached = skills
	b.loaded = true
	return skills, nil
}

// loadAllFromDisk reads skills from the filesystem.
func (b *Backend) loadAllFromDisk() ([]skill.Skill, error) {
	entries, err := os.ReadDir(b.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill directory %q: %w", b.baseDir, err)
	}

	var skills []skill.Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(b.baseDir, entry.Name(), skillFileName)
		s, err := b.loadSkill(skillPath)
		if err != nil {
			continue
		}
		skills = append(skills, s)
	}
	return skills, nil
}

func (b *Backend) loadSkill(path string) (skill.Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return skill.Skill{}, fmt.Errorf("failed to read %s: %w", path, err)
	}

	frontmatter, content, err := parseFrontmatter(string(data))
	if err != nil {
		return skill.Skill{}, fmt.Errorf("failed to parse frontmatter in %s: %w", path, err)
	}

	var fm skill.FrontMatter
	if err := yaml.Unmarshal([]byte(frontmatter), &fm); err != nil {
		return skill.Skill{}, fmt.Errorf("failed to unmarshal frontmatter in %s: %w", path, err)
	}

	return skill.Skill{
		FrontMatter:   fm,
		Content:       strings.TrimSpace(content),
		BaseDirectory: filepath.Dir(path),
	}, nil
}

// parseFrontmatter extracts YAML frontmatter delimited by --- from the content.
func parseFrontmatter(data string) (frontmatter string, content string, err error) {
	const delimiter = "---"

	data = strings.TrimSpace(data)
	if !strings.HasPrefix(data, delimiter) {
		return "", "", fmt.Errorf("file does not start with frontmatter delimiter")
	}

	rest := data[len(delimiter):]
	before, after, found := strings.Cut(rest, "\n"+delimiter)
	if !found {
		return "", "", fmt.Errorf("frontmatter closing delimiter not found")
	}

	frontmatter = strings.TrimSpace(before)
	content = strings.TrimPrefix(after, "\n")

	return frontmatter, content, nil
}
