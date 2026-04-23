package assist

import (
	"fmt"
	"strings"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
)

// BuildSystemPrompt generates a system prompt based on field metadata.
func BuildSystemPrompt(meta aiv1.FieldMeta) string {
	var builder strings.Builder

	builder.WriteString("You are a Professional Ops Assistant.\n")
	builder.WriteString("Your task is to help the user fill in a form field.\n\n")

	if meta.Label != "" {
		builder.WriteString(fmt.Sprintf("Field: %s\n", meta.Label))
	}
	if meta.Purpose != "" {
		builder.WriteString(fmt.Sprintf("Purpose: %s\n", meta.Purpose))
	}
	if meta.Rules != "" {
		builder.WriteString(fmt.Sprintf("Rules: %s\n", meta.Rules))
	}
	if meta.CurrentValue != "" {
		builder.WriteString(fmt.Sprintf("Current Value: %s\n", meta.CurrentValue))
		builder.WriteString("Note: The user may want to modify, optimize, or extend this existing value.\n")
	}

	builder.WriteString("\nConstraint: Output ONLY the value. No markdown fences. No explanation.")

	return builder.String()
}
