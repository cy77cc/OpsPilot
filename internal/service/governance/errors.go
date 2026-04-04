package governance

const (
	CodeSuccess               = "success"
	CodeApprovalRequired      = "approval_required"
	CodeApprovalRejected      = "approval_rejected"
	CodeApprovalTokenInvalid  = "approval_token_invalid"
	CodeApprovalTokenExpired  = "approval_token_expired"
	CodeApprovalTokenReplay   = "approval_token_replayed"
	CodeApprovalScopeMismatch = "approval_token_scope_mismatch"
	CodeApprovalNotApproved   = "approval_token_not_approved"
	CodePermissionDenied      = "permission_denied"
	CodePolicyNotFound        = "policy_not_found"
	CodeInternalError         = "internal_error"
)

type GovError struct {
	Code    string
	Message string
}

func (e *GovError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func NewGovError(code, message string) *GovError {
	return &GovError{Code: code, Message: message}
}

func IsGovError(err error) (*GovError, bool) {
	if err == nil {
		return nil, false
	}
	govErr, ok := err.(*GovError)
	return govErr, ok
}
