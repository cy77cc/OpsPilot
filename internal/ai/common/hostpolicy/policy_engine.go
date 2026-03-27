package hostpolicy

import "strings"

const maxPolicyCommandLength = 4096

// HostCommandPolicyEngine enforces host command policy rules.
type HostCommandPolicyEngine struct {
	readonlyAllowlist map[string]struct{}
	validator         *HostCommandValidator
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
		validator:         NewHostCommandValidator(readonlyAllowlist),
		policyVersion:     "host_policy/v1",
	}
}

// DefaultReadonlyAllowlist returns the initial readonly command allowlist.
func DefaultReadonlyAllowlist() []string {
	return []string{
		"ls", "pwd", "tree", "stat", "file", "wc", "du",
		"cat", "head", "tail", "grep", "egrep", "fgrep", "zcat", "zgrep", "sort", "uniq", "cut", "tr", "column", "jq",
		"top", "free", "df", "uptime", "uname", "dmesg", "vmstat", "iostat", "sar", "lsblk", "lscpu", "lsmod", "lspci",
		"ping", "ss", "netstat", "ip", "route", "arp", "host", "nslookup", "dig",
		"ps", "pstree", "whoami", "id", "w", "who", "last", "journalctl",
	}
}

// Evaluate performs fail-closed policy evaluation.
func (e *HostCommandPolicyEngine) Evaluate(input PolicyInput) PolicyDecision {
	cmd := strings.TrimSpace(input.CommandRaw)
	if len(cmd) > maxPolicyCommandLength {
		return e.requireApproval([]string{"command_too_long"}, nil, "")
	}

	parsed, err := ParseCommand(cmd)
	if err != nil {
		return e.requireApproval([]string{"parse_error"}, nil, "")
	}

	violations := e.validator.Validate(parsed)
	if len(violations) > 0 {
		return e.requireApproval(reasonCodesFromViolations(violations), violations, strings.Join(parsed.BaseCommands, ","))
	}

	if strings.TrimSpace(input.ToolName) == "host_exec_change" {
		return e.requireApproval([]string{"change_requires_approval"}, nil, strings.Join(parsed.BaseCommands, ","))
	}
	return PolicyDecision{
		DecisionType:  DecisionAllowReadonlyExecute,
		ReasonCodes:   []string{"readonly_allowed"},
		PolicyVersion: e.policyVersion,
		ASTSummary:    strings.Join(parsed.BaseCommands, ","),
	}
}

func (e *HostCommandPolicyEngine) requireApproval(reasonCodes []string, violations []PolicyViolation, astSummary string) PolicyDecision {
	return PolicyDecision{
		DecisionType:  DecisionRequireApprovalInterrupt,
		ReasonCodes:   reasonCodes,
		Violations:    violations,
		PolicyVersion: e.policyVersion,
		ASTSummary:    astSummary,
	}
}

func reasonCodesFromViolations(violations []PolicyViolation) []string {
	if len(violations) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	codes := make([]string, 0, len(violations))
	for _, violation := range violations {
		code := strings.TrimSpace(violation.Type)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	if len(codes) == 0 {
		codes = append(codes, "policy_violation")
	}
	return codes
}
