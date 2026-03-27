package host

import "strings"

// HostCommandValidator validates parsed command AST summaries.
type HostCommandValidator struct {
	readonlyAllowlist map[string]struct{}
}

// NewHostCommandValidator creates a validator with readonly command allowlist.
func NewHostCommandValidator(allowlist []string) *HostCommandValidator {
	set := make(map[string]struct{}, len(allowlist))
	for _, item := range allowlist {
		name := strings.TrimSpace(strings.ToLower(item))
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	return &HostCommandValidator{readonlyAllowlist: set}
}

// Validate checks whether a parsed command is readonly-safe.
func (v *HostCommandValidator) Validate(parsed *ParsedCommand) []PolicyViolation {
	if parsed == nil {
		return []PolicyViolation{{Type: "parse_error", Detail: "parsed command is nil"}}
	}

	violations := make([]PolicyViolation, 0)
	if parsed.HasRedirection {
		violations = append(violations, PolicyViolation{Type: "disallowed_operator", Detail: "redirection"})
	}
	if parsed.HasBackground {
		violations = append(violations, PolicyViolation{Type: "disallowed_operator", Detail: "background"})
	}
	if parsed.HasCommandSubstitution {
		violations = append(violations, PolicyViolation{Type: "disallowed_operator", Detail: "command_substitution"})
	}
	for _, cmd := range parsed.BaseCommands {
		if _, ok := v.readonlyAllowlist[cmd]; ok {
			continue
		}
		violations = append(violations, PolicyViolation{Type: "command_not_allowlisted", Detail: cmd, Segment: cmd})
	}
	return violations
}
