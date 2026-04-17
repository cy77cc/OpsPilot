package context

import "fmt"

// CompressOverflow keeps a budgeted slice and replaces dropped context with a summary marker.
func CompressOverflow(history []Message, budget Budget) []Message {
	selected := SelectBudgeted(history, budget)
	if len(selected) == len(history) {
		return append([]Message(nil), selected...)
	}

	overflow := len(history) - len(selected)
	summary := Message{
		Role:    "system",
		Content: fmt.Sprintf("compressed %d overflow messages", overflow),
		Pinned:  true,
	}

	out := make([]Message, 0, len(selected)+1)
	out = append(out, summary)
	out = append(out, selected...)
	return out
}
