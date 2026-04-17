package chathandler

import "testing"

func TestLegacyRoutingPathRemoved(t *testing.T) {
	if LegacyRoutingEnabled() {
		t.Fatal("legacy routing must be removed")
	}
}
