package host

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidator_RejectsNonAllowlistedCommand(t *testing.T) {
	parsed, err := ParseCommand("systemctl status nginx")
	require.NoError(t, err)

	validator := NewHostCommandValidator(DefaultReadonlyAllowlist())
	violations := validator.Validate(parsed)
	require.NotEmpty(t, violations)
	require.Equal(t, "command_not_allowlisted", violations[0].Type)
}

func TestValidator_RejectsAwkInInitialAllowlist(t *testing.T) {
	parsed, err := ParseCommand("awk '{print $1}' /etc/hosts")
	require.NoError(t, err)

	validator := NewHostCommandValidator(DefaultReadonlyAllowlist())
	violations := validator.Validate(parsed)
	require.NotEmpty(t, violations)
	require.Equal(t, "command_not_allowlisted", violations[0].Type)
}

func TestValidator_AllowsPipelineWhenEachCommandAllowlisted(t *testing.T) {
	parsed, err := ParseCommand("cat /var/log/syslog | grep error")
	require.NoError(t, err)

	validator := NewHostCommandValidator(DefaultReadonlyAllowlist())
	violations := validator.Validate(parsed)
	require.Empty(t, violations)
}

func TestValidator_RejectsRedirectionAndBackground(t *testing.T) {
	validator := NewHostCommandValidator(DefaultReadonlyAllowlist())

	parsedRedirection, err := ParseCommand("cat /etc/hosts > /tmp/hosts")
	require.NoError(t, err)
	require.NotEmpty(t, validator.Validate(parsedRedirection))

	parsedBackground, err := ParseCommand("cat /etc/hosts &")
	require.NoError(t, err)
	require.NotEmpty(t, validator.Validate(parsedBackground))
}

func TestValidator_CommandChainRequiresEachSegmentAllowlisted(t *testing.T) {
	parsed, err := ParseCommand("cat /etc/hosts; uname -a")
	require.NoError(t, err)

	validator := NewHostCommandValidator(DefaultReadonlyAllowlist())
	violations := validator.Validate(parsed)
	require.NotEmpty(t, violations)
	require.Equal(t, "command_not_allowlisted", violations[0].Type)
}
