// Package logic 实现 AI 模块的业务逻辑层。
//
// 本文件定义审批相关的错误类型和上下文工具函数。
package logic

import (
	"context"
	"fmt"
	"strings"
)

// approvalSubmitIdempotencyKeyContextKey 幂等键上下文键。
type approvalSubmitIdempotencyKeyContextKey struct{}

// WithApprovalSubmitIdempotencyKey 将幂等键存入上下文。
//
// 用于支持审批提交的幂等性，防止重复提交。
//
// 参数:
//   - ctx: 上下文
//   - key: 幂等键（通常由客户端生成）
//
// 返回: 包含幂等键的新上下文
func WithApprovalSubmitIdempotencyKey(ctx context.Context, key string) context.Context {
	trimmed := strings.TrimSpace(key)
	if ctx == nil || trimmed == "" {
		return ctx
	}
	return context.WithValue(ctx, approvalSubmitIdempotencyKeyContextKey{}, trimmed)
}

// ApprovalSubmitIdempotencyKeyFromContext 从上下文获取幂等键。
func ApprovalSubmitIdempotencyKeyFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	key, _ := ctx.Value(approvalSubmitIdempotencyKeyContextKey{}).(string)
	return strings.TrimSpace(key)
}

// ApprovalNotFoundError 审批任务不存在错误。
type ApprovalNotFoundError struct {
	ApprovalID string
}

// Error 实现错误接口。
func (e *ApprovalNotFoundError) Error() string {
	if e == nil || strings.TrimSpace(e.ApprovalID) == "" {
		return "approval not found"
	}
	return fmt.Sprintf("approval %q not found", e.ApprovalID)
}

// ApprovalForbiddenError 审批权限不足错误。
//
// 当用户尝试操作不属于自己会话的审批任务时返回。
type ApprovalForbiddenError struct {
	ApprovalID string
	UserID     uint64
}

// Error 实现错误接口。
func (e *ApprovalForbiddenError) Error() string {
	if e == nil || strings.TrimSpace(e.ApprovalID) == "" {
		return "approval does not belong to current user"
	}
	return fmt.Sprintf("approval %q does not belong to current user", e.ApprovalID)
}

// ApprovalConflictError 审批冲突错误。
//
// 当审批任务已被其他用户处理或状态已变更时返回。
type ApprovalConflictError struct {
	ApprovalID string
	Message    string
}

// Error 实现错误接口。
func (e *ApprovalConflictError) Error() string {
	if e == nil {
		return "approval already handled"
	}
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if strings.TrimSpace(e.ApprovalID) == "" {
		return "approval already handled"
	}
	return fmt.Sprintf("approval %q already handled", e.ApprovalID)
}
