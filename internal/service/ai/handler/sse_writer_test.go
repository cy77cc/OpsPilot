package handler

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSSEWriter_WriteEvent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	if err := writer.WriteEvent("meta", map[string]any{
		"session_id": "sess-1",
		"run_id":     "run-1",
		"turn":       1,
	}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	raw := buf.String()
	if !strings.HasPrefix(raw, "event: meta\ndata: ") || !strings.HasSuffix(raw, "\n\n") {
		t.Fatalf("unexpected SSE framing: %q", raw)
	}

	payload := strings.TrimSuffix(strings.TrimPrefix(raw, "event: meta\ndata: "), "\n\n")
	var got map[string]any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	if got["session_id"] != "sess-1" || got["run_id"] != "run-1" || got["turn"] != float64(1) {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestSSEWriter_WritePing(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	if err := writer.WritePing(); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	if got := buf.String(); got != ": ping\n\n" {
		t.Fatalf("unexpected ping frame: %q", got)
	}
}
