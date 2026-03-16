package runtime

import (
	"encoding/json"
	"fmt"
)

type StreamEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

var publicEventNames = map[string]struct{}{
	"init":         {},
	"intent":       {},
	"status":       {},
	"delta":        {},
	"progress":     {},
	"report_ready": {},
	"error":        {},
	"done":         {},
}

func EncodePublicEvent(event string, data any) ([]byte, error) {
	if _, ok := publicEventNames[event]; !ok {
		return nil, fmt.Errorf("unsupported public event %q", event)
	}
	return json.Marshal(StreamEvent{
		Event: event,
		Data:  data,
	})
}
