package approvalhandler

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// Service provides approval-related use cases.
type Service struct {
	logic *logic.Logic
}

func NewService(svcCtx *svc.ServiceContext) *Service {
	return NewServiceWithLogic(logic.NewAILogic(svcCtx))
}

func NewServiceWithLogic(l *logic.Logic) *Service {
	return &Service{logic: l}
}

func (s *Service) SubmitApproval(ctx context.Context, input logic.SubmitApprovalInput) (*logic.SubmitApprovalOutput, error) {
	return s.logic.SubmitApproval(ctx, input)
}

func (s *Service) RetryResumeApproval(ctx context.Context, input logic.RetryResumeApprovalInput) (*logic.RetryResumeApprovalOutput, error) {
	return s.logic.RetryResumeApproval(ctx, input)
}

func (s *Service) GetApproval(ctx context.Context, approvalID string, userID uint64) (*ai.AIApprovalTask, error) {
	return s.logic.GetApproval(ctx, approvalID, userID)
}

func (s *Service) ListPendingApprovals(ctx context.Context, userID uint64) ([]ai.AIApprovalTask, error) {
	return s.logic.ListPendingApprovals(ctx, userID)
}
