// Package logic 提供 AI 命令审计上下文的传递工具。
//
// 本文件定义了用于在请求链中传递 AI 命令审计信息的上下文工具，
// 支持 AI 触发的 CI/CD 操作审计追踪。
package logic

import (
	"context"
	"encoding/json"

	"github.com/cy77cc/OpsPilot/internal/runtimectx"
)

// CommandAuditContext 是 AI 命令审计上下文。
//
// 用于记录 AI 触发 CI/CD 操作的审计信息，包括命令 ID、意图、计划哈希等。
// 该上下文通过 HTTP 请求链传递，最终写入审计事件表。
type CommandAuditContext struct {
	CommandID       string         `json:"command_id"`                 // AI 命令 ID
	Intent          string         `json:"intent"`                     // AI 意图
	PlanHash        string         `json:"plan_hash"`                  // 执行计划哈希
	TraceID         string         `json:"trace_id"`                   // 追踪 ID
	ApprovalContext map[string]any `json:"approval_context,omitempty"` // 审批上下文
	Summary         string         `json:"summary,omitempty"`          // 执行摘要
}

// commandAuditContextKey 是上下文键类型。
type commandAuditContextKey struct{}

// WithCommandAuditContext 将命令审计上下文注入到 context 中。
//
// 参数:
//   - ctx: 原始上下文
//   - meta: 命令审计上下文
//
// 返回: 包含审计上下文的新上下文
func WithCommandAuditContext(ctx context.Context, meta CommandAuditContext) context.Context {
	return runtimectx.WithValue(ctx, commandAuditContextKey{}, meta)
}

// commandAuditContextFromContext 从 context 中提取命令审计上下文。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 命令审计上下文和是否存在标志
func commandAuditContextFromContext(ctx context.Context) (CommandAuditContext, bool) {
	if ctx == nil {
		return CommandAuditContext{}, false
	}
	meta, ok := runtimectx.Value(ctx, commandAuditContextKey{}).(CommandAuditContext)
	if !ok {
		return CommandAuditContext{}, false
	}
	return meta, true
}

// mustJSONOrEmpty 将对象序列化为 JSON 字符串，失败时返回空字符串。
//
// 参数:
//   - v: 待序列化的对象
//
// 返回: JSON 字符串，失败时返回空字符串
func mustJSONOrEmpty(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
