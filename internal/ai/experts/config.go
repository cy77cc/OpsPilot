package experts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadExpertsConfig(path string) (*ExpertsFile, error) {
	path = resolveConfigPath(path)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ExpertsFile
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if err := ValidateExpertsConfig(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func ValidateExpertsConfig(cfg *ExpertsFile) error {
	if cfg == nil {
		return fmt.Errorf("experts config is nil")
	}
	if len(cfg.Experts) == 0 {
		return fmt.Errorf("experts config has no experts")
	}
	seen := make(map[string]struct{}, len(cfg.Experts))
	for _, exp := range cfg.Experts {
		name := strings.TrimSpace(exp.Name)
		if name == "" {
			return fmt.Errorf("expert name is required")
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate expert name: %s", name)
		}
		seen[name] = struct{}{}
		if strings.TrimSpace(exp.Persona) == "" {
			return fmt.Errorf("expert %s persona is required", name)
		}
		if len(exp.ToolPatterns) == 0 {
			return fmt.Errorf("expert %s tool_patterns is required", name)
		}
	}
	return nil
}

func LoadSceneMappings(path string) (*SceneMappingsFile, error) {
	path = resolveConfigPath(path)
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg SceneMappingsFile
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.Mappings) == 0 {
		return nil, fmt.Errorf("scene mappings config has no mappings")
	}
	return &cfg, nil
}

func resolveConfigPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return path
	}
	if filepath.IsAbs(path) {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	wd, err := os.Getwd()
	if err != nil {
		return path
	}
	dir := wd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return path
}
