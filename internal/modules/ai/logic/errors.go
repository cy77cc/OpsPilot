// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件定义审批相关的错误类型和上下文工具函数，
// 从 approval 子包重新导出以保持向后兼容。
package logic

import (
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/approval"
)

// Re-export approval types for backward compatibility.
type (
	ApprovalNotFoundError  = approval.ApprovalNotFoundError
	ApprovalForbiddenError = approval.ApprovalForbiddenError
	ApprovalConflictError  = approval.ApprovalConflictError
)

// Re-export approval context functions.
var (
	WithApprovalSubmitIdempotencyKey   = approval.WithApprovalSubmitIdempotencyKey
	ApprovalSubmitIdempotencyKeyFromContext = approval.ApprovalSubmitIdempotencyKeyFromContext
)
