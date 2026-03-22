package logic

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

type ApprovalOrchestrator = common.ApprovalOrchestrator

type approvalPolicyStore interface {
	ListEnabledByToolName(ctx context.Context, toolName string) ([]model.AIToolRiskPolicy, error)
}

type approvalTaskWriter interface {
	Create(ctx context.Context, task *model.AIApprovalTask) error
}

type approvalOutboxWriter interface {
	EnqueueOrTouch(ctx context.Context, event *model.AIApprovalOutboxEvent) error
}

func NewApprovalOrchestrator(db *gorm.DB) *ApprovalOrchestrator {
	return common.NewApprovalOrchestrator(db)
}

func NewApprovalOrchestratorWithStores(policy approvalPolicyStore, approval approvalTaskWriter, outbox approvalOutboxWriter) *ApprovalOrchestrator {
	return common.NewApprovalOrchestratorWithStores(policy, approval, outbox)
}
