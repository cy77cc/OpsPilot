package logic

import "strings"

// nilIfEmpty returns nil if string is empty or whitespace only, otherwise returns pointer to string.
func nilIfEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
