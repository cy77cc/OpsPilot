package observability

import "testing"

func TestEnsureTraceID_GeneratesWhenEmpty(t *testing.T) {
	got := EnsureTraceID("")
	if got == "" {
		t.Fatal("trace id must not be empty")
	}
}
