package assist

import (
	"regexp"
	"strings"
)

var (
	fenceRegex   = regexp.MustCompile("(?s)^```(?:[a-zA-Z0-9]+)?\n?(.*?)\n?```$")
	leadInRegex  = regexp.MustCompile(`(?i)^.*:[\s\n]*`)
	sensitiveKey = map[string]bool{
		"password":   true,
		"secret":     true,
		"token":      true,
		"api_key":    true,
		"access_key": true,
	}
)

// NormalizeFormAssistOutput cleans the AI output by trimming, removing fences and lead-ins.
func NormalizeFormAssistOutput(raw string) string {
	raw = strings.TrimSpace(raw)

	// 1. Remove one-line lead-ins like "Here is the query:"
	lines := strings.SplitN(raw, "\n", 2)
	if len(lines) > 0 && strings.HasSuffix(strings.TrimSpace(lines[0]), ":") {
		if len(lines) > 1 {
			raw = strings.TrimSpace(lines[1])
		}
	}

	// 2. Remove markdown fences (now that lead-in is gone)
	if matches := fenceRegex.FindStringSubmatch(raw); len(matches) > 1 {
		raw = strings.TrimSpace(matches[1])
	}

	return strings.TrimSpace(raw)
}

// SanitizeFormContext removes sensitive information from the form context.
func SanitizeFormContext(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any)
	for k, v := range input {
		if _, ok := sensitiveKey[strings.ToLower(k)]; ok {
			continue
		}
		output[k] = v
	}
	return output
}
