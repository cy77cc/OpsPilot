package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestServiceModulesUseStructuredLayout(t *testing.T) {
	t.Parallel()

	modules := []string{
		"automation",
		"cmdb",
		"dashboard",
		"jobs",
		"monitoring",
		"topology",
		"ai",
		"cicd",
		"cluster",
		"deployment",
		"service",
	}

	for _, module := range modules {
		module := module
		t.Run(module, func(t *testing.T) {
			t.Parallel()

			for _, name := range []string{"handler", "logic"} {
				path := filepath.Join(module, name)
				info, err := os.Stat(path)
				if err != nil {
					t.Fatalf("expected %s to exist: %v", path, err)
				}
				if !info.IsDir() {
					t.Fatalf("expected %s to be a directory", path)
				}
			}
		})
	}
}
