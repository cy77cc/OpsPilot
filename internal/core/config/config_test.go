package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewConfigReturnsErrorWhenFileMissing(t *testing.T) {
	SetConfigFile(filepath.Join(t.TempDir(), "missing-config.yaml"))
	if err := NewConfig(); err == nil {
		t.Fatalf("expected error when config file does not exist")
	}
}

func TestValidateConfigRejectsMissingSecrets(t *testing.T) {
	cfg := Config{
		App:    App{Name: "opspilot"},
		Server: Server{Port: 8080, ReadTimeout: time.Second, WriteTimeout: time.Second, IdleTimeout: time.Second, Salt: "server-salt"},
		JWT:    JWT{Secret: "${SERVER_SECRET}"},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatalf("expected error for unresolved jwt secret placeholder")
	}
	if !strings.Contains(err.Error(), "jwt.secret") {
		t.Fatalf("expected jwt.secret validation error, got: %v", err)
	}
}

func TestValidateConfigRejectsInvalidPort(t *testing.T) {
	cfg := Config{
		App:    App{Name: "opspilot"},
		Server: Server{Port: 0, ReadTimeout: time.Second, WriteTimeout: time.Second, IdleTimeout: time.Second, Salt: "salt"},
		JWT:    JWT{Secret: "secret"},
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Fatalf("expected error for invalid server.port")
	}
	if !strings.Contains(err.Error(), "server.port") {
		t.Fatalf("expected server.port validation error, got: %v", err)
	}
}

func TestValidateConfigSuccess(t *testing.T) {
	cfg := Config{
		App: App{Name: "opspilot"},
		Server: Server{
			Port:         8080,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
			Salt:         "server-salt",
		},
		JWT:   JWT{Secret: "jwt-secret"},
		Redis: Redis{Enable: true, Addr: "127.0.0.1:6379"},
	}

	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}
