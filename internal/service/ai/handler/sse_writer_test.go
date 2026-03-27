package handler

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/service/ai/chat"
)

func TestSSEWriter_WriteEvent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := chat.NewSSEWriter(&buf)

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

func TestSSEWriter_WriteEventWithID(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := chat.NewSSEWriter(&buf)

	if err := writer.WriteEvent("run_state", map[string]any{
		"event_id": "evt-42",
		"status":   "completed",
		"agent":    "executor",
	}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	raw := buf.String()
	if !strings.HasPrefix(raw, "id: evt-42\nevent: run_state\ndata: ") || !strings.HasSuffix(raw, "\n\n") {
		t.Fatalf("unexpected SSE framing: %q", raw)
	}

	payload := strings.TrimSuffix(strings.TrimPrefix(raw, "id: evt-42\nevent: run_state\ndata: "), "\n\n")
	var got map[string]any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if _, ok := got["event_id"]; ok {
		t.Fatalf("expected event_id to be stripped from SSE payload, got %#v", got)
	}
	if got["status"] != "completed" || got["agent"] != "executor" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestSSEWriter_WriteRunStateEvent(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := chat.NewSSEWriter(&buf)

	if err := writer.WriteEvent("run_state", map[string]any{
		"status": "executing",
		"agent":  "K8sAgent",
	}); err != nil {
		t.Fatalf("write event: %v", err)
	}

	raw := buf.String()
	if !strings.HasPrefix(raw, "event: run_state\ndata: ") || !strings.HasSuffix(raw, "\n\n") {
		t.Fatalf("unexpected SSE framing: %q", raw)
	}

	payload := strings.TrimSuffix(strings.TrimPrefix(raw, "event: run_state\ndata: "), "\n\n")
	var got map[string]any
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if got["status"] != "executing" || got["agent"] != "K8sAgent" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestSSEWriter_WritePing(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	writer := chat.NewSSEWriter(&buf)

	if err := writer.WritePing(); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	if got := buf.String(); got != ": ping\n\n" {
		t.Fatalf("unexpected ping frame: %q", got)
	}
}
