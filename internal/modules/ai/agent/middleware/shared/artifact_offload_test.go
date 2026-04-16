package shared

import "testing"

func TestShouldOffloadResult_UsesOutputModeAndSize(t *testing.T) {
	if !ShouldOffloadResult("summary_plus_artifact", 1200) {
		t.Fatal("expected large summary_plus_artifact result to offload")
	}
	if ShouldOffloadResult("inline", 120) {
		t.Fatal("did not expect small inline result to offload")
	}
}
