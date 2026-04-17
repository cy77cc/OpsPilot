package observability

import (
	"strings"

	"github.com/google/uuid"
)

func EnsureTraceID(in string) string {
	trimmed := strings.TrimSpace(in)
	if trimmed != "" {
		return trimmed
	}
	return uuid.NewString()
}
