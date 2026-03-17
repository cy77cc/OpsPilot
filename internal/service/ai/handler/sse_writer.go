package handler

import (
	"encoding/json"
	"fmt"
	"io"
)

type SSEWriter struct {
	writer io.Writer
}

func NewSSEWriter(writer io.Writer) *SSEWriter {
	return &SSEWriter{writer: writer}
}

func (w *SSEWriter) WriteEvent(event string, payload any) error {
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
