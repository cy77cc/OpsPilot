// Package sceneutil provides shared scene normalization and tool-allowance utilities.
package sceneutil

import "strings"

const DefaultScene = "ai"

// NormalizeScene normalizes a scene name to lowercase trimmed form.
// Returns "ai" for empty input.
func NormalizeScene(scene string) string {
	scene = strings.TrimSpace(scene)
	if scene == "" {
		return DefaultScene
	}
	return strings.ToLower(scene)
}

// AllowedToolSet provides O(1) tool-name membership checks.
type AllowedToolSet struct {
	allowed map[string]struct{}
}

// NewAllowedToolSet creates a set from a slice of tool names.
func NewAllowedToolSet(names []string) *AllowedToolSet {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	return &AllowedToolSet{allowed: set}
}

// IsAllowed returns true if the tool name is in the set.
func (s *AllowedToolSet) IsAllowed(name string) bool {
	if s == nil || s.allowed == nil {
		return false
	}
	_, ok := s.allowed[name]
	return ok
}

// Len returns the number of allowed tools.
func (s *AllowedToolSet) Len() int {
	if s == nil || s.allowed == nil {
		return 0
	}
	return len(s.allowed)
}

// Names returns the allowed tool names in no guaranteed order.
func (s *AllowedToolSet) Names() []string {
	if s == nil || s.allowed == nil {
		return nil
	}
	names := make([]string, 0, len(s.allowed))
	for n := range s.allowed {
		names = append(names, n)
	}
	return names
}
