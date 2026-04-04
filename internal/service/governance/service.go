package governance

import (
	"context"
	"strings"
)

type Service interface {
	Preflight(ctx context.Context, intent OperationIntent) (Decision, error)
	Finalize(ctx context.Context, in FinalizeInput) (FinalizeOutput, error)
	BuildEnvelope(decision Decision, auditID uint, data any) Envelope
}

type PolicyResolver interface {
	Resolve(ctx context.Context, scope Scope) (Policy, error)
}

type ApprovalService interface {
	Issue(ctx context.Context, intent OperationIntent, reason string) (*ApprovalInfo, error)
	Consume(ctx context.Context, intent OperationIntent) error
	Confirm(ctx context.Context, ticket string, approverID uint, approved bool, note string) error
}

type AuditService interface {
	Record(ctx context.Context, in FinalizeInput) (uint, error)
}

type Redactor interface {
	Redact(v any) any
}

type coreService struct {
	policyResolver  PolicyResolver
	approvalService ApprovalService
	auditService    AuditService
	redactor        Redactor
}

func NewService(policyResolver PolicyResolver, approvalService ApprovalService, auditService AuditService, redactor Redactor) Service {
	return &coreService{
		policyResolver:  policyResolver,
		approvalService: approvalService,
		auditService:    auditService,
		redactor:        redactor,
	}
}

func (s *coreService) Preflight(ctx context.Context, intent OperationIntent) (Decision, error) {
	if s == nil || s.policyResolver == nil {
		return Decision{State: StateFailed, Code: CodeInternalError, Message: "policy resolver not configured"}, NewGovError(CodeInternalError, "policy resolver not configured")
	}

	intent.Scope = MergeScopeFromContext(ctx, intent.Scope)
	policy, err := s.policyResolver.Resolve(ctx, intent.Scope)
	if err != nil {
		return s.decisionFromError(err), err
	}

	if policy.ApprovalRequired {
		reason := strings.TrimSpace(policy.ApprovalReason)
		if reason == "" {
			reason = CodeApprovalRequired
		}

		if strings.TrimSpace(intent.ApprovalToken) == "" {
			if s.approvalService == nil {
				return Decision{State: StateFailed, Code: CodeInternalError, Message: "approval service not configured"}, NewGovError(CodeInternalError, "approval service not configured")
			}
			approval, err := s.approvalService.Issue(ctx, intent, reason)
			if err != nil {
				return s.decisionFromError(err), err
			}
			return Decision{
				Allowed:  false,
				State:    StateApprovalRequired,
				Code:     CodeApprovalRequired,
				Message:  reason,
				Approval: approval,
			}, nil
		}

		if s.approvalService == nil {
			return Decision{State: StateFailed, Code: CodeInternalError, Message: "approval service not configured"}, NewGovError(CodeInternalError, "approval service not configured")
		}
		if err := s.approvalService.Consume(ctx, intent); err != nil {
			return s.decisionFromError(err), err
		}
	}

	code := CodeSuccess
	if policy.ApprovalRequired {
		code = CodeSuccess
	}
	return Decision{
		Allowed: true,
		State:   StateCompleted,
		Code:    code,
	}, nil
}

func (s *coreService) Finalize(ctx context.Context, in FinalizeInput) (FinalizeOutput, error) {
	if s == nil || s.auditService == nil {
		return FinalizeOutput{}, NewGovError(CodeInternalError, "audit service not configured")
	}
	auditID, err := s.auditService.Record(ctx, in)
	if err != nil {
		return FinalizeOutput{}, err
	}
	return FinalizeOutput{AuditID: auditID}, nil
}

func (s *coreService) BuildEnvelope(decision Decision, auditID uint, data any) Envelope {
	return BuildEnvelope(decision, auditID, data)
}

func (s *coreService) decisionFromError(err error) Decision {
	if err == nil {
		return Decision{State: StateFailed, Code: CodeInternalError, Message: CodeInternalError}
	}
	govErr, ok := IsGovError(err)
	if !ok {
		return Decision{State: StateFailed, Code: CodeInternalError, Message: err.Error()}
	}
	switch govErr.Code {
	case CodeApprovalRequired:
		return Decision{State: StateApprovalRequired, Code: govErr.Code, Message: govErr.Message}
	case CodeApprovalRejected, CodePermissionDenied:
		return Decision{State: StateRejected, Code: govErr.Code, Message: govErr.Message}
	default:
		return Decision{State: StateFailed, Code: govErr.Code, Message: govErr.Message}
	}
}

func BuildEnvelope(decision Decision, auditID uint, data any) Envelope {
	state := decision.State
	if state == "" {
		state = StateCompleted
	}
	code := strings.TrimSpace(decision.Code)
	if code == "" {
		switch state {
		case StateApprovalRequired:
			code = CodeApprovalRequired
		case StateRejected:
			code = CodeApprovalRejected
		case StateFailed:
			code = CodeInternalError
		default:
			code = CodeSuccess
		}
	}

	env := Envelope{
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

func IsCode(err error, code string) bool {
	govErr, ok := IsGovError(err)
	return ok && govErr.Code == code
}
