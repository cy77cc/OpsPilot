package host

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicyEngine_FailClosedOnParserError(t *testing.T) {
	engine := NewHostCommandPolicyEngine(DefaultReadonlyAllowlist())
	got := engine.Evaluate(PolicyInput{ToolName: "host_exec_readonly", CommandRaw: "echo $("})
	require.Equal(t, DecisionRequireApprovalInterrupt, got.DecisionType)
	require.Contains(t, got.ReasonCodes, "parse_error")
}

func TestPolicyEngine_FailClosedOnCommandTooLong(t *testing.T) {
	engine := NewHostCommandPolicyEngine(DefaultReadonlyAllowlist())
	got := engine.Evaluate(PolicyInput{
		ToolName:   "host_exec_readonly",
		CommandRaw: strings.Repeat("a", 4097),
	})
	require.Equal(t, DecisionRequireApprovalInterrupt, got.DecisionType)
	require.Contains(t, got.ReasonCodes, "command_too_long")
}

func TestPolicyEngine_RequireApprovalWhenReadonlyValidationFails(t *testing.T) {
	engine := NewHostCommandPolicyEngine(DefaultReadonlyAllowlist())
	got := engine.Evaluate(PolicyInput{ToolName: "host_exec_readonly", CommandRaw: "systemctl status nginx"})
	require.Equal(t, DecisionRequireApprovalInterrupt, got.DecisionType)
	require.Contains(t, got.ReasonCodes, "command_not_allowlisted")
}
