package logic

import (
	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
)

// FormAssistInput defines the input for form assistance.
type FormAssistInput struct {
	Scene       string
	UserPrompt  string
	FieldMeta   aiv1.FieldMeta
	FormContext map[string]any
	UserID      uint64
}
