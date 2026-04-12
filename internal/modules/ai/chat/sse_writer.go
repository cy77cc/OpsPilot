package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SSEWriter writes server-sent events.
type SSEWriter struct {
	writer io.Writer
}

func NewSSEWriter(writer io.Writer) *SSEWriter {
	return &SSEWriter{writer: writer}
}

func (w *SSEWriter) WriteEvent(event string, payload any) error {
	return w.WriteEventWithID("", event, payload)
}

func (w *SSEWriter) WriteEventWithID(eventID, event string, payload any) error {
	if w == nil || w.writer == nil {
		return fmt.Errorf("sse writer is nil")
	}
	payload, resolvedEventID := sanitizeSSEPayload(payload, eventID)
	if resolvedEventID != "" {
		if _, err := fmt.Fprintf(w.writer, "id: %s\n", resolvedEventID); err != nil {
			return err
		}
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w.writer, "event: %s\ndata: %s\n\n", event, data)
	return err
}

func (w *SSEWriter) WritePing() error {
	_, err := io.WriteString(w.writer, ": ping\n\n")
	return err
}

func sanitizeSSEPayload(payload any, eventID string) (any, string) {
	if trimmed := strings.TrimSpace(eventID); trimmed != "" {
		return payload, trimmed
	}

	data, ok := payload.(map[string]any)
	if !ok {
		return payload, ""
	}

	candidate, _ := data["event_id"].(string)
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return payload, ""
	}

	copyPayload := make(map[string]any, len(data))
	for key, value := range data {
		if key == "event_id" {
			continue
		}
		copyPayload[key] = value
	}
	return copyPayload, candidate
}
