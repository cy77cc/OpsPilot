// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件提供审批编排器的服务层封装，代理调用 tools/common 包中的实现。
package logic

import (
	"context"

	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	common "github.com/cy77cc/OpsPilot/internal/modules/ai/common/approval"
	"gorm.io/gorm"
)

// ApprovalOrchestrator 审批编排器别名，代理 common 包实现。
type ApprovalOrchestrator = common.ApprovalOrchestrator

// approvalPolicyStore 审批策略存储接口。
type approvalPolicyStore interface {
	ListEnabledByToolName(ctx context.Context, toolName string) ([]ai.AIToolRiskPolicy, error)
}

// approvalTaskWriter 审批任务写入接口。
type approvalTaskWriter interface {
	Create(ctx context.Context, task *ai.AIApprovalTask) error
}

// approvalOutboxWriter 审批 Outbox 写入接口。
type approvalOutboxWriter interface {
	EnqueueOrTouch(ctx context.Context, event *ai.AIApprovalOutboxEvent) error
}

// NewApprovalOrchestrator 创建审批编排器实例。
func NewApprovalOrchestrator(db *gorm.DB) *ApprovalOrchestrator {
	return common.NewApprovalOrchestrator(db)
}

func NewApprovalOrchestratorWithStores(policy approvalPolicyStore, approval approvalTaskWriter, outbox approvalOutboxWriter) *ApprovalOrchestrator {
	return common.NewApprovalOrchestratorWithStores(policy, approval, outbox)
}
