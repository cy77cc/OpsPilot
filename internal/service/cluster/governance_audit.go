package cluster

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/service/governance"
	governanceapproval "github.com/cy77cc/OpsPilot/internal/service/governance/approval"
	governanceaudit "github.com/cy77cc/OpsPilot/internal/service/governance/audit"
	governancepolicy "github.com/cy77cc/OpsPilot/internal/service/governance/policy"
)

func (h *Handler) phase3Preflight(ctx context.Context, intent governance.OperationIntent) (governance.Decision, error) {
	if intent.OccurredAt.IsZero() {
		intent.OccurredAt = time.Now().UTC()
	}
	intent.Scope.Domain = "cluster"
	svc := h.phase3GovernanceService()
	return svc.Preflight(ctx, intent)
}

func (h *Handler) phase3Finalize(ctx context.Context, in governance.FinalizeInput) (governance.FinalizeOutput, error) {
	if in.Intent.OccurredAt.IsZero() {
		in.Intent.OccurredAt = time.Now().UTC()
	}
	in.Intent.Scope.Domain = "cluster"
	svc := h.phase3GovernanceService()
	return svc.Finalize(ctx, in)
}

func (h *Handler) phase3BuildEnvelope(decision governance.Decision, auditID uint, data any) governance.Envelope {
	return h.phase3GovernanceService().BuildEnvelope(decision, auditID, data)
}

func (h *Handler) phase3GovernanceService() governance.Service {
	rules := []governancepolicy.Rule{
		{
			Domain:           "cluster",
			Resource:         "admission",
			Action:           "admission.apply",
			ApprovalRequired: true,
			ApprovalReason:   "phase3 admission change requires approval",
			RiskLevel:        governance.RiskHigh,
		},
		{
			Domain:           "cluster",
			Resource:         "gitops",
			Action:           "gitops.sync",
			ApprovalRequired: true,
			ApprovalReason:   "phase3 gitops sync requires approval",
			RiskLevel:        governance.RiskHigh,
		},
		{
			Domain:           "cluster",
			Resource:         "runtime",
			Action:           "runtime.contain",
			ApprovalRequired: true,
			ApprovalReason:   "phase3 runtime containment requires approval",
			RiskLevel:        governance.RiskCritical,
		},
	}
	resolver := governancepolicy.NewResolver(rules...)
	resolver.AllowFallback = true
	resolver.DefaultPolicy = &governance.Policy{
		RiskLevel:        governance.RiskMedium,
		ApprovalRequired: false,
	}

	approvalSvc := governanceapproval.NewService(h.svcCtx.DB)
	auditSvc := governanceaudit.NewService(h.svcCtx.DB, nil)
	return governance.NewService(resolver, approvalSvc, auditSvc, nil)
}
