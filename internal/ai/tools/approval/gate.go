package approval

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

type interruptState struct {
	ArgumentsInJSON  string `json:"arguments_json"`
	EditedArguments  string `json:"edited_arguments,omitempty"`
	Approved         bool   `json:"approved,omitempty"`
	Reason           string `json:"reason,omitempty"`
}

type Gate struct {
	inner           tool.InvokableTool
	meta            airuntime.ApprovalToolSpec
	decisionMaker   *airuntime.ApprovalDecisionMaker
	summaryRenderer *SummaryRenderer
}

func NewGate(inner tool.InvokableTool, meta airuntime.ApprovalToolSpec, decisionMaker *airuntime.ApprovalDecisionMaker, summaryRenderer *SummaryRenderer) *Gate {
	if summaryRenderer == nil {
		summaryRenderer = NewSummaryRenderer()
	}
	return &Gate{
		inner:           inner,
		meta:            meta,
		decisionMaker:   decisionMaker,
		summaryRenderer: summaryRenderer,
	}
}

func (g *Gate) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return g.inner.Info(ctx)
}

func (g *Gate) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	params, err := parseArguments(argumentsInJSON)
	if err != nil {
		return "", err
	}
	runtimeCtx := airuntime.RuntimeContext{}
	if g.decisionMaker != nil {
		runtimeCtx = g.runtimeContext(ctx)
	}
	decision, err := g.decide(ctx, params, runtimeCtx)
	if err != nil {
		return "", err
	}
	if !decision.NeedApproval {
		return g.inner.InvokableRun(ctx, argumentsInJSON, opts...)
	}

	wasInterrupted, hasState, state := tool.GetInterruptState[interruptState](ctx)
	if wasInterrupted {
		return g.handleResume(ctx, hasState, state, opts...)
	}
	return g.triggerInterrupt(ctx, argumentsInJSON, decision, params)
}

func (g *Gate) handleResume(ctx context.Context, hasState bool, state interruptState, opts ...tool.Option) (string, error) {
	if !hasState {
		return "", fmt.Errorf("approval gate resume state is missing")
	}
	isTarget, hasData, resume := tool.GetResumeContext[interruptState](ctx)
	if !isTarget {
		return "", tool.StatefulInterrupt(ctx, nil, state)
	}
	if hasData {
		state.Approved = resume.Approved
		state.Reason = resume.Reason
		// 保存编辑后的参数
		if resume.EditedArguments != "" {
			state.EditedArguments = resume.EditedArguments
		}
	}
	return g.executeResumed(ctx, state, opts...)
}

func (g *Gate) executeResumed(ctx context.Context, state interruptState, opts ...tool.Option) (string, error) {
	if !state.Approved {
		message := "approval rejected"
		if strings.TrimSpace(state.Reason) != "" {
			message = message + ": " + strings.TrimSpace(state.Reason)
		}
		return message, nil
	}
	// 优先使用编辑后的参数，否则使用原始参数
	argumentsToUse := state.ArgumentsInJSON
	if strings.TrimSpace(state.EditedArguments) != "" {
		argumentsToUse = state.EditedArguments
	}
	return g.inner.InvokableRun(ctx, argumentsToUse, opts...)
}

func (g *Gate) triggerInterrupt(ctx context.Context, argumentsInJSON string, decision airuntime.ApprovalDecision, params map[string]any) (string, error) {
	info := airuntime.ApprovalInterruptInfo{
		ToolName:        decision.Tool.Name,
		ToolDisplayName: firstValue(decision.Tool.DisplayName, decision.Tool.Name),
		Mode:            decision.Tool.Mode,
		RiskLevel:       decision.Tool.Risk,
		Summary:         g.summaryRenderer.Render(decision, params),
		Params:          params,
		ArgumentsInJSON: argumentsInJSON,
		Environment:     decision.Environment,
		Namespace:       firstValue(stringValue(params["namespace"]), namespaceFromResources(g.runtimeContext(ctx).SelectedResources)),
	}
	return "", tool.StatefulInterrupt(ctx, info, interruptState{ArgumentsInJSON: argumentsInJSON})
}

func (g *Gate) decide(ctx context.Context, params map[string]any, runtimeCtx airuntime.RuntimeContext) (airuntime.ApprovalDecision, error) {
	if g.decisionMaker == nil {
		return airuntime.ApprovalDecision{
			NeedApproval: strings.EqualFold(g.meta.Mode, "mutating"),
			Reason:       "default mutating tool approval",
			Tool:         g.meta,
		}, nil
	}
	return g.decisionMaker.Decide(ctx, airuntime.ApprovalCheckRequest{
		ToolName:       g.meta.Name,
		Mode:           g.meta.Mode,
		Risk:           g.meta.Risk,
		Scene:          runtimeCtx.Scene,
		Environment:    stringValue(runtimeCtx.Metadata["environment"]),
		Namespace:      firstValue(stringValue(params["namespace"]), namespaceFromResources(runtimeCtx.SelectedResources)),
		Params:         params,
		RuntimeContext: runtimeCtx,
	})
}

func (g *Gate) runtimeContext(ctx context.Context) airuntime.RuntimeContext {
	return airuntime.RuntimeContextFromContext(ctx)
}

func parseArguments(argumentsInJSON string) (map[string]any, error) {
	argumentsInJSON = strings.TrimSpace(argumentsInJSON)
	if argumentsInJSON == "" {
		return map[string]any{}, nil
	}
	params := make(map[string]any)
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return nil, fmt.Errorf("parse approval gate arguments: %w", err)
	}
	return params, nil
}

func namespaceFromResources(resources []airuntime.SelectedResource) string {
	for _, resource := range resources {
		if strings.TrimSpace(resource.Namespace) != "" {
			return strings.TrimSpace(resource.Namespace)
		}
	}
	return ""
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func firstValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
