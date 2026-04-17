package projection

import "strings"

// Event is a compact projection input event.
type Event struct {
	Type string
	Text string
}

// Block is a compact rendered projection block.
type Block struct {
	Text string `json:"text"`
}

// State stores the current incremental projection snapshot.
type State struct {
	Version int     `json:"version"`
	Blocks  []Block `json:"blocks"`
}

// ApplyEvent applies one event to the current state.
func ApplyEvent(state State, event Event) State {
	next := state
	next.Version++
	if strings.TrimSpace(event.Text) != "" {
		next.Blocks = append(append([]Block(nil), state.Blocks...), Block{Text: event.Text})
	}
	return next
}
