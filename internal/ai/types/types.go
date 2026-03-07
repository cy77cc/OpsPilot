package types

// AgentResult represents the result of an agent operation
type AgentResult struct {
	Type     string
	Content  string
	ToolName string
	ToolData map[string]any
	Ask      *AskRequest
}

// AskRequest represents a request for user input/approval
type AskRequest struct {
	ID          string
	Title       string
	Description string
	Risk        string
	Details     map[string]any
}
