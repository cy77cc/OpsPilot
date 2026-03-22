package host

import (
	"strings"
)

const maxPolicyCommandLength = 4096

// HostCommandPolicyEngine enforces host command policy rules.
type HostCommandPolicyEngine struct {
	readonlyAllowlist map[string]struct{}
	policyVersion     string
}

// NewHostCommandPolicyEngine creates a policy engine with readonly allowlist.
func NewHostCommandPolicyEngine(readonlyAllowlist []string) *HostCommandPolicyEngine {
	set := make(map[string]struct{}, len(readonlyAllowlist))
	for _, item := range readonlyAllowlist {
		name := strings.TrimSpace(strings.ToLower(item))
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return &HostCommandPolicyEngine{
		readonlyAllowlist: set,
		policyVersion:     "host_policy/v1",
	}
}

// DefaultReadonlyAllowlist returns the initial readonly command allowlist.
func DefaultReadonlyAllowlist() []string {
	return []string{"cat", "ls", "grep", "top", "free", "df", "tail"}
}

// Evaluate performs fail-closed policy evaluation.
func (e *HostCommandPolicyEngine) Evaluate(input PolicyInput) PolicyDecision {
	cmd := strings.TrimSpace(input.CommandRaw)
	if len(cmd) > maxPolicyCommandLength {
		return e.requireApproval("command_too_long")
	}

	parsed, err := ParseCommand(cmd)
	if err != nil {
		return e.requireApproval("parse_error")
	}

	if strings.TrimSpace(input.ToolName) == "host_exec_change" {
		return e.requireApproval("change_requires_approval")
	}
	return PolicyDecision{
		DecisionType:  DecisionAllowReadonlyExecute,
		ReasonCodes:   []string{"readonly_allowed"},
		PolicyVersion: e.policyVersion,
		ASTSummary:    strings.Join(parsed.BaseCommands, ","),
	}
}

func (e *HostCommandPolicyEngine) requireApproval(reason string) PolicyDecision {
	return PolicyDecision{
		DecisionType:  DecisionRequireApprovalInterrupt,
		ReasonCodes:   []string{reason},
		PolicyVersion: e.policyVersion,
	}
}
