package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/tools/core"
)

type ToolResult struct {
	OK        bool   `json:"ok"`
	ErrorCode string `json:"error_code,omitempty"`
	Data      any    `json:"data,omitempty"`
	Error     string `json:"error,omitempty"`
	Source    string `json:"source"`
	LatencyMS int64  `json:"latency_ms"`
}

func WithToolUser(ctx context.Context, userID uint64, approvalToken string) context.Context {
	return core.WithToolUser(ctx, userID, approvalToken)
}

func WithToolRuntimeContext(ctx context.Context, runtime map[string]any) context.Context {
	return core.WithToolRuntimeContext(ctx, runtime)
}

func WithToolMemoryAccessor(ctx context.Context, accessor core.ToolMemoryAccessor) context.Context {
	return core.WithToolMemoryAccessor(ctx, accessor)
}

func WithToolEventEmitter(ctx context.Context, emitter core.ToolEventEmitter) context.Context {
	return core.WithToolEventEmitter(ctx, emitter)
}

func ToolUserFromContext(ctx context.Context) (uint64, string) {
	return core.ToolUserFromContext(ctx)
}

func MarshalToolResult(result ToolResult) (string, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// ApprovalInfo is emitted when a high-risk tool requires approval.
type ApprovalInfo struct {
	ToolName        string         `json:"tool_name"`
	ArgumentsInJSON string         `json:"arguments"`
	Risk            string         `json:"risk"`
	Preview         map[string]any `json:"preview"`
}

// ApprovalResult carries user decision for a high-risk tool.
type ApprovalResult struct {
	Approved         bool    `json:"approved"`
	DisapproveReason *string `json:"disapprove_reason,omitempty"`
}

// ReviewEditInfo is emitted when a medium-risk tool requires parameter review.
type ReviewEditInfo struct {
	ToolName        string `json:"tool_name"`
	ArgumentsInJSON string `json:"arguments"`
}

// ReviewEditResult carries the reviewed/edited arguments.
type ReviewEditResult struct {
	EditedArgumentsInJSON *string `json:"edited_arguments,omitempty"`
	NoNeedToEdit          bool    `json:"no_need_to_edit"`
	Disapproved           bool    `json:"disapproved"`
	DisapproveReason      *string `json:"disapprove_reason,omitempty"`
}

func init() {
	schema.Register[*ApprovalInfo]()
	schema.Register[*ApprovalResult]()
	schema.Register[*ReviewEditInfo]()
	schema.Register[*ReviewEditResult]()
	schema.Register[map[string]any]()
	schema.Register[[]any]()
	schema.Register[[]map[string]any]()
}

type ApprovalPreviewFn func(ctx context.Context, args string) (map[string]any, error)

type ApprovableTool struct {
	tool.InvokableTool
	risk      string
	previewFn ApprovalPreviewFn
}

func NewApprovableTool(base tool.InvokableTool, risk string, previewFn ApprovalPreviewFn) *ApprovableTool {
	return &ApprovableTool{InvokableTool: base, risk: risk, previewFn: previewFn}
}

func (t *ApprovableTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	info, err := t.Info(ctx)
	if err != nil {
		return "", err
	}

	wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
	if !wasInterrupted {
		preview, err := t.preview(ctx, args)
		if err != nil {
			return "", err
		}
		return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
			ToolName:        info.Name,
			ArgumentsInJSON: args,
			Risk:            t.risk,
			Preview:         preview,
		}, args)
	}

	isResumeTarget, hasData, result := tool.GetResumeContext[*ApprovalResult](ctx)
	if !isResumeTarget {
		preview, err := t.preview(ctx, storedArgs)
		if err != nil {
			return "", err
		}
		return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
			ToolName:        info.Name,
			ArgumentsInJSON: storedArgs,
			Risk:            t.risk,
			Preview:         preview,
		}, storedArgs)
	}
	if !hasData || result == nil {
		return "", fmt.Errorf("missing approval result for tool %q", info.Name)
	}
	if !result.Approved {
		if result.DisapproveReason != nil {
			return fmt.Sprintf("tool %q disapproved: %s", info.Name, *result.DisapproveReason), nil
		}
		return fmt.Sprintf("tool %q disapproved", info.Name), nil
	}
	return t.InvokableTool.InvokableRun(ctx, storedArgs, opts...)
}

func (t *ApprovableTool) preview(ctx context.Context, args string) (map[string]any, error) {
	if t.previewFn == nil {
		return nil, nil
	}
	return t.previewFn(ctx, args)
}
