package envelope

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/service/governance"
)

type Mapper struct{}

func NewMapper() *Mapper {
	return &Mapper{}
}

func (m *Mapper) Build(decision governance.Decision, auditID uint, data any) governance.Envelope {
	return Build(decision, auditID, data)
}

func Build(decision governance.Decision, auditID uint, data any) governance.Envelope {
	state := decision.State
	if state == "" {
		state = governance.StateCompleted
	}
	code := strings.TrimSpace(decision.Code)
	if code == "" {
		switch state {
		case governance.StateApprovalRequired:
			code = governance.CodeApprovalRequired
		case governance.StateRejected:
			code = governance.CodeApprovalRejected
		case governance.StateFailed:
			code = governance.CodeInternalError
		default:
			code = governance.CodeSuccess
		}
	}
	env := governance.Envelope{
		State:   state,
		AuditID: auditID,
		Code:    code,
		Message: strings.TrimSpace(decision.Message),
		Data:    data,
	}
	if decision.Approval != nil {
		env.Approval = decision.Approval
	}
	return env
}
