package streaming

import (
	"errors"

	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
)

const (
	// PublicErrorCodeInternal is the fallback stream error code.
	PublicErrorCodeInternal = "AI_STREAM_INTERNAL"
	// PublicErrorCodeCursorExpired is returned when a replay cursor is stale.
	PublicErrorCodeCursorExpired = "AI_STREAM_CURSOR_EXPIRED"
)

// PublicError is the public-facing AI stream/API error contract.
type PublicError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// MapStreamError converts internal errors into public stream errors.
func MapStreamError(err error) PublicError {
	if errors.Is(err, aidao.ErrRunEventCursorExpired) {
		return PublicError{
			Code:      PublicErrorCodeCursorExpired,
			Message:   "last_event_id is too old; refresh the stream snapshot",
			Retryable: false,
		}
	}
	return PublicError{
		Code:      PublicErrorCodeInternal,
		Message:   "An internal error occurred",
		Retryable: true,
	}
}
