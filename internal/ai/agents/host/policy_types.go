package host

// DecisionType represents policy engine outcomes.
type DecisionType string

const (
	DecisionAllowReadonlyExecute     DecisionType = "allow_readonly_execute"
	DecisionRequireApprovalInterrupt DecisionType = "require_approval_interrupt"
)

// PolicyInput captures command execution context for policy evaluation.
type PolicyInput struct {
	ToolName     string
	AgentRole    string
	Target       string
	CommandRaw   string
	SessionID    string
	RunID        string
	CallID       string
	CheckpointID string
}

// PolicyDecision is the policy engine output.
type PolicyDecision struct {
	DecisionType  DecisionType
	ReasonCodes   []string
	Violations    []PolicyViolation
	PolicyVersion string
	ASTSummary    string
}

// PolicyViolation describes one policy validation issue.
type PolicyViolation struct {
	Type    string
	Detail  string
	Segment string
}
