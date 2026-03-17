package runtime

import (
	"encoding/json"
	"strings"
)

// decodeStepsEnvelope parses planner/replanner step payloads.
func decodeStepsEnvelope(raw string) ([]string, bool) {
	var payload struct {
		Steps []string `json:"steps"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || len(payload.Steps) == 0 {
		return nil, false
	}
	return payload.Steps, true
}

// decodeResponseEnvelope parses replanner final response payloads.
func decodeResponseEnvelope(raw string) (string, bool) {
	var payload struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil || strings.TrimSpace(payload.Response) == "" {
		return "", false
	}
	return payload.Response, true
}
