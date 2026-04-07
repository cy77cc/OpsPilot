package handler

import (
	"context"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/service/governance/logic"
)

type Service interface {
	Preflight(ctx context.Context, intent logic.OperationIntent) (logic.Decision, error)
	Finalize(ctx context.Context, in logic.FinalizeInput) (logic.FinalizeOutput, error)
	BuildEnvelope(decision logic.Decision, auditID uint, data any) logic.Envelope
}

type PolicyResolver interface {
	Resolve(ctx context.Context, scope logic.Scope) (logic.Policy, error)
}

type ApprovalService interface {
	Issue(ctx context.Context, intent logic.OperationIntent, reason string) (*logic.ApprovalInfo, error)
	Consume(ctx context.Context, intent logic.OperationIntent) error
	Confirm(ctx context.Context, ticket string, approverID uint, approved bool, note string) error
}

type AuditService interface {
	Record(ctx context.Context, in logic.FinalizeInput) (uint, error)
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

func (s *coreService) Preflight(ctx context.Context, intent logic.OperationIntent) (logic.Decision, error) {
	if s == nil || s.policyResolver == nil {
		return logic.Decision{State: logic.StateFailed, Code: CodeInternalError, Message: "policy resolver not configured"}, NewGovError(CodeInternalError, "policy resolver not configured")
	}

	intent.Scope = logic.MergeScopeFromContext(ctx, intent.Scope)
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
				return logic.Decision{State: logic.StateFailed, Code: CodeInternalError, Message: "approval service not configured"}, NewGovError(CodeInternalError, "approval service not configured")
			}
			approval, err := s.approvalService.Issue(ctx, intent, reason)
			if err != nil {
				return s.decisionFromError(err), err
			}
			return logic.Decision{
				Allowed:  false,
				State:    logic.StateApprovalRequired,
				Code:     CodeApprovalRequired,
				Message:  reason,
				Approval: approval,
			}, nil
		}

		if s.approvalService == nil {
			return logic.Decision{State: logic.StateFailed, Code: CodeInternalError, Message: "approval service not configured"}, NewGovError(CodeInternalError, "approval service not configured")
		}
		if err := s.approvalService.Consume(ctx, intent); err != nil {
			return s.decisionFromError(err), err
		}
	}

	code := CodeSuccess
	if policy.ApprovalRequired {
		code = CodeSuccess
	}
	return logic.Decision{
		Allowed: true,
		State:   logic.StateCompleted,
		Code:    code,
	}, nil
}

func (s *coreService) Finalize(ctx context.Context, in logic.FinalizeInput) (logic.FinalizeOutput, error) {
	if s == nil || s.auditService == nil {
		return logic.FinalizeOutput{}, NewGovError(CodeInternalError, "audit service not configured")
	}
	auditID, err := s.auditService.Record(ctx, in)
	if err != nil {
		return logic.FinalizeOutput{}, err
	}
	return logic.FinalizeOutput{AuditID: auditID}, nil
}

func (s *coreService) BuildEnvelope(decision logic.Decision, auditID uint, data any) logic.Envelope {
	return BuildEnvelope(decision, auditID, data)
}

func (s *coreService) decisionFromError(err error) logic.Decision {
	if err == nil {
		return logic.Decision{State: logic.StateFailed, Code: CodeInternalError, Message: CodeInternalError}
	}
	govErr, ok := IsGovError(err)
	if !ok {
		return logic.Decision{State: logic.StateFailed, Code: CodeInternalError, Message: err.Error()}
	}
	switch govErr.Code {
	case CodeApprovalRequired:
		return logic.Decision{State: logic.StateApprovalRequired, Code: govErr.Code, Message: govErr.Message}
	case CodeApprovalRejected, CodePermissionDenied:
		return logic.Decision{State: logic.StateRejected, Code: govErr.Code, Message: govErr.Message}
	default:
		return logic.Decision{State: logic.StateFailed, Code: govErr.Code, Message: govErr.Message}
	}
}

func BuildEnvelope(decision logic.Decision, auditID uint, data any) logic.Envelope {
	state := decision.State
	if state == "" {
		state = logic.StateCompleted
	}
	code := strings.TrimSpace(decision.Code)
	if code == "" {
		switch state {
		case logic.StateApprovalRequired:
			code = CodeApprovalRequired
		case logic.StateRejected:
			code = CodeApprovalRejected
		case logic.StateFailed:
			code = CodeInternalError
		default:
			code = CodeSuccess
		}
	}

	env := logic.Envelope{
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