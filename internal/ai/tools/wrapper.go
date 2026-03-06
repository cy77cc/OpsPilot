package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ApprovalInfo is emitted when a high-risk tool requires approval.
type ApprovalInfo struct {
	ToolName        string         `json:"tool_name"`
	ArgumentsInJSON string         `json:"arguments"`
	Risk            ToolRisk       `json:"risk"`
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
	risk      ToolRisk
	previewFn ApprovalPreviewFn
}

func NewApprovableTool(base tool.InvokableTool, risk ToolRisk, previewFn ApprovalPreviewFn) *ApprovableTool {
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
		return map[string]any{}, nil
	}
	preview, err := t.previewFn(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("build approval preview: %w", err)
	}
	if preview == nil {
		return map[string]any{}, nil
	}
	return preview, nil
}

type ReviewableTool struct {
	tool.InvokableTool
}

func NewReviewableTool(base tool.InvokableTool) *ReviewableTool {
	return &ReviewableTool{InvokableTool: base}
}

func (t *ReviewableTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	info, err := t.Info(ctx)
	if err != nil {
		return "", err
	}

	wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
	if !wasInterrupted {
		return "", tool.StatefulInterrupt(ctx, &ReviewEditInfo{
			ToolName:        info.Name,
			ArgumentsInJSON: args,
		}, args)
	}

	isResumeTarget, hasData, result := tool.GetResumeContext[*ReviewEditResult](ctx)
	if !isResumeTarget {
		return "", tool.StatefulInterrupt(ctx, &ReviewEditInfo{
			ToolName:        info.Name,
			ArgumentsInJSON: storedArgs,
		}, storedArgs)
	}
	if !hasData || result == nil {
		return "", fmt.Errorf("missing review result for tool %q", info.Name)
	}
	if result.Disapproved {
		if result.DisapproveReason != nil {
			return fmt.Sprintf("tool %q disapproved: %s", info.Name, *result.DisapproveReason), nil
		}
		return fmt.Sprintf("tool %q disapproved", info.Name), nil
	}

	callArgs := storedArgs
	if result.EditedArgumentsInJSON != nil {
		callArgs = *result.EditedArgumentsInJSON
	} else if !result.NoNeedToEdit {
		return "", fmt.Errorf("invalid review result for tool %q", info.Name)
	}

	return t.InvokableTool.InvokableRun(ctx, callArgs, opts...)
}
