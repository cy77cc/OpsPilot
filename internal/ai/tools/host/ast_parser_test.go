package host

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCommand_CollectsPipelineCommands(t *testing.T) {
	parsed, err := ParseCommand("cat /var/log/syslog | grep error")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"cat", "grep"}, parsed.BaseCommands)
}
