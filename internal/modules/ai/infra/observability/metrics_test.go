package observability

import "testing"

func TestCounterInc_IncrementsValue(t *testing.T) {
	c := NewCounter("ai_stream_error_total")
	c.Inc()
	if c.Value() != 1 {
		t.Fatalf("expected 1, got %d", c.Value())
	}
}
