package ai

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ApprovalConfig struct {
	Version    string                          `yaml:"version"`
	Defaults   ApprovalDefaultConfig           `yaml:"defaults"`
	RiskPolicy map[string]ApprovalRiskPolicy   `yaml:"risk_policy"`
	Tools      map[string]ApprovalToolOverride `yaml:"tools"`
}

type ApprovalDefaultConfig struct {
	RequireConfirmationForReadonly bool   `yaml:"require_confirmation_for_readonly"`
	ConfirmationTimeoutSeconds     int    `yaml:"confirmation_timeout_seconds"`
	ApprovalTimeoutSeconds         int    `yaml:"approval_timeout_seconds"`
	RiskLevel                      string `yaml:"risk_level"`
}

type ApprovalRiskPolicy struct {
	RequiresApproval bool `yaml:"requires_approval"`
	TimeoutSeconds   int  `yaml:"timeout_seconds"`
}

type ApprovalToolOverride struct {
	RiskLevel              string `yaml:"risk_level"`
	ConfirmationTimeoutSec int    `yaml:"confirmation_timeout_seconds"`
	ApprovalTimeoutSec     int    `yaml:"approval_timeout_seconds"`
}

type ApprovalDecision struct {
	RiskLevel           string
	RequiresApproval    bool
	ConfirmationTimeout time.Duration
	ApprovalTimeout     time.Duration
}

func LoadApprovalConfig(path string) (*ApprovalConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg ApprovalConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *ApprovalConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("approval config is nil")
	}
	if c.Defaults.ConfirmationTimeoutSeconds <= 0 {
		return fmt.Errorf("defaults.confirmation_timeout_seconds must be > 0")
	}
	if c.Defaults.ApprovalTimeoutSeconds <= 0 {
		return fmt.Errorf("defaults.approval_timeout_seconds must be > 0")
	}
	if !isRiskLevel(c.Defaults.RiskLevel) {
		return fmt.Errorf("defaults.risk_level must be one of: low, medium, high")
	}
	for level, rule := range c.RiskPolicy {
		if !isRiskLevel(level) {
			return fmt.Errorf("risk_policy key must be one of: low, medium, high")
		}
		if rule.TimeoutSeconds <= 0 {
			return fmt.Errorf("risk_policy.%s.timeout_seconds must be > 0", level)
		}
	}
	for toolName, item := range c.Tools {
		if strings.TrimSpace(toolName) == "" {
			return fmt.Errorf("tool name must not be empty")
		}
		if item.RiskLevel != "" && !isRiskLevel(item.RiskLevel) {
			return fmt.Errorf("tools.%s.risk_level must be one of: low, medium, high", toolName)
		}
		if item.ConfirmationTimeoutSec < 0 || item.ApprovalTimeoutSec < 0 {
			return fmt.Errorf("tools.%s timeout must be >= 0", toolName)
		}
	}
	return nil
}

func isRiskLevel(s string) bool {
	l := strings.ToLower(strings.TrimSpace(s))
	return l == "low" || l == "medium" || l == "high"
}
