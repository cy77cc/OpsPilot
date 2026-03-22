package common

import "testing"

func TestFallbackRequiresApproval_CoversHostExecChange(t *testing.T) {
	if !fallbackRequiresApproval("host_exec_change", "service_control") {
		t.Fatal("expected host_exec_change to require approval in fallback policy")
	}
}
