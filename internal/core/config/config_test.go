package config

import (
	"path/filepath"
	"testing"
)

func TestNewConfigReturnsErrorWhenFileMissing(t *testing.T) {
	SetConfigFile(filepath.Join(t.TempDir(), "missing-config.yaml"))
	if err := NewConfig(); err == nil {
		t.Fatalf("expected error when config file does not exist")
	}
}
