package monitor

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/agent/contracts"
)

func TestShouldDelegateToIsolationWorker(t *testing.T) {
	if !ShouldDelegateToIsolationWorker(contracts.Scope{TimeRange: "24h"}, 1200) {
		t.Fatal("expected large result set to require worker delegation")
	}
	if ShouldDelegateToIsolationWorker(contracts.Scope{TimeRange: "5m"}, 20) {
		t.Fatal("did not expect small result set to require worker delegation")
	}
}
