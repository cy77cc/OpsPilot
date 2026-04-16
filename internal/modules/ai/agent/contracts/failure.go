package contracts

type FailureCode string

const (
	FailureCodeUnsupportedScope      FailureCode = "unsupported_scope"
	FailureCodeInsufficientData      FailureCode = "insufficient_data"
	FailureCodeToolFailed            FailureCode = "tool_failed"
	FailureCodeTimeout               FailureCode = "timeout"
	FailureCodeArtifactWriteFailed   FailureCode = "artifact_write_failed"
	FailureCodeSummaryGenerateFailed FailureCode = "summary_generation_failed"
	FailureCodePolicyDenied          FailureCode = "policy_denied"
)

func (c FailureCode) Known() bool {
	switch c {
	case FailureCodeUnsupportedScope,
		FailureCodeInsufficientData,
		FailureCodeToolFailed,
		FailureCodeTimeout,
		FailureCodeArtifactWriteFailed,
		FailureCodeSummaryGenerateFailed,
		FailureCodePolicyDenied:
		return true
	default:
		return false
	}
}
