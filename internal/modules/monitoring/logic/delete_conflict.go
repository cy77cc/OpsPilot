package logic

import "fmt"

type DeleteBlocker struct {
	Type    string   `json:"type"`
	Count   int      `json:"count"`
	Samples []string `json:"samples,omitempty"`
}

type DeleteConflictError struct {
	Resource string          `json:"resource"`
	Blockers []DeleteBlocker `json:"blockers"`
}

func (e *DeleteConflictError) Error() string {
	return fmt.Sprintf("%s has references", e.Resource)
}
