// Package handler provides AI service HTTP handlers
package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
)

// jsonMarshal is an alias for json.Marshal that returns a string
func jsonMarshal(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// Type aliases for logic package types
type (
	AISession            = logic.AISession
	RecommendationRecord = logic.RecommendationRecord
	ApprovalTicket       = logic.ApprovalTicket
	ExecutionRecord      = logic.ExecutionRecord
)

// toolMemoryAccessor is an alias for logic.ToolMemoryAccessor
type toolMemoryAccessor = logic.ToolMemoryAccessor

// Helper functions

func toString(v any) string {
	return logic.ToString(v)
}

func toBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		ok, _ := strconv.ParseBool(strings.TrimSpace(x))
		return ok
	case json.Number:
		n, _ := x.Int64()
		return n != 0
	case int:
		return x != 0
	case int64:
		return x != 0
	case int32:
		return x != 0
	case float64:
		return x != 0
	case float32:
		return x != 0
	default:
		return false
	}
}

// normalizeSessionTitle normalizes a session title
func normalizeSessionTitle(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, trimmed)
	rs := []rune(strings.TrimSpace(trimmed))
	if len(rs) > 64 {
		rs = rs[:64]
	}
	return strings.TrimSpace(string(rs))
}
