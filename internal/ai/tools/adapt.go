// Package tools 提供 AI 工具的注册、适配和执行能力。
//
// 本文件实现 ToolSpec 到 ADK InvokableTool 的适配，
// 并通过 Gate 包装实现变更类工具的统一审批拦截。
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	approvaltools "github.com/cy77cc/OpsPilot/internal/ai/tools/approval"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

// ADKToolAdapter 将 ToolSpec 转换为 ADK InvokableTool。
//
// 同时处理只读工具和变更工具的 Gate 包装。
type ADKToolAdapter struct {
	registry      *Registry
	decisionMaker *airuntime.ApprovalDecisionMaker
}

// NewADKToolAdapter 创建工具适配器。
func NewADKToolAdapter(registry *Registry, decisionMaker *airuntime.ApprovalDecisionMaker) *ADKToolAdapter {
	return &ADKToolAdapter{
		registry:      registry,
		decisionMaker: decisionMaker,
	}
}

// AdaptAll 将注册表中的所有工具转换为 ADK tool 列表。
//
// 变更类工具自动通过 Gate 包装实现审批拦截。
func (a *ADKToolAdapter) AdaptAll() []tool.BaseTool {
	if a == nil || a.registry == nil {
		return nil
	}

	caps := a.registry.List()
	tools := make([]tool.BaseTool, 0, len(caps))
	for _, cap := range caps {
		spec, ok := a.registry.Get(cap.Name)
		if !ok {
			continue
		}
		tools = append(tools, a.adaptTool(spec))
	}

	return tools
}

// adaptTool 转换单个工具，根据模式决定是否包装 Gate。
func (a *ADKToolAdapter) adaptTool(spec ToolSpec) tool.BaseTool {
	baseTool := &invokableToolWrapper{
		name:        spec.Name,
		description: spec.Description,
		params:      paramInfoMap(spec.Schema),
		deps:        a.registry.deps,
		execute:     spec.Execute,
		preview:     spec.Preview,
	}

	needApproval := spec.Mode == ModeMutating || spec.Risk == RiskHigh
	if !needApproval {
		return baseTool
	}

	approvalSpec := airuntime.ApprovalToolSpec{
		Name:        spec.Name,
		DisplayName: spec.DisplayName,
		Description: spec.Description,
		Mode:        string(spec.Mode),
		Risk:        string(spec.Risk),
		Category:    spec.Category,
	}

	return approvaltools.NewGate(baseTool, approvalSpec, a.decisionMaker, nil)
}

// invokableToolWrapper 实现 tool.InvokableTool 接口。
type invokableToolWrapper struct {
	name        string
	description string
	params      map[string]*schema.ParameterInfo
	deps        common.PlatformDeps
	execute     func(context.Context, common.PlatformDeps, map[string]any) (*Execution, error)
	preview     func(context.Context, common.PlatformDeps, map[string]any) (any, error)
}

// Info 返回工具元信息和参数描述。
func (w *invokableToolWrapper) Info(context.Context) (*schema.ToolInfo, error) {
	info := &schema.ToolInfo{
		Name:  w.name,
		Desc:  w.description,
		Extra: map[string]any{},
	}
	if len(w.params) > 0 {
		info.ParamsOneOf = schema.NewParamsOneOfByParams(w.params)
	}
	return info, nil
}

// InvokableRun 执行底层 ToolSpec.Execute。
func (w *invokableToolWrapper) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	params := make(map[string]any)
	if strings.TrimSpace(argumentsInJSON) != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
			return "", fmt.Errorf("parse arguments: %w", err)
		}
	}

	deps := common.PlatformDepsFromContext(ctx)
	if deps == nil {
		deps = &w.deps
	}

	if w.execute == nil {
		if w.preview == nil {
			return "", fmt.Errorf("tool %q has no execute function", w.name)
		}
		result, err := w.preview(ctx, *deps, params)
		if err != nil {
			return "", err
		}
		output, err := json.Marshal(result)
		if err != nil {
			return "", fmt.Errorf("marshal preview result: %w", err)
		}
		return string(output), nil
	}

	result, err := w.execute(ctx, *deps, params)
	if err != nil {
		return "", err
	}

	output, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}

	return string(output), nil
}

func paramInfoMap(hints map[string]ParamHint) map[string]*schema.ParameterInfo {
	if len(hints) == 0 {
		return nil
	}
	out := make(map[string]*schema.ParameterInfo, len(hints))
	for name, hint := range hints {
		out[name] = &schema.ParameterInfo{
			Type:     dataTypeFromHint(hint.Type),
			Desc:     strings.TrimSpace(hint.Hint),
			Required: hint.Required,
			Enum:     enumStrings(hint.Options),
		}
	}
	return out
}

func dataTypeFromHint(value string) schema.DataType {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "boolean":
		return schema.Boolean
	case "integer":
		return schema.Integer
	case "number":
		return schema.Number
	case "array":
		return schema.Array
	case "object":
		return schema.Object
	default:
		return schema.String
	}
}

func enumStrings(options []ParamOption) []string {
	if len(options) == 0 {
		return nil
	}
	out := make([]string, 0, len(options))
	for _, option := range options {
		value := strings.TrimSpace(fmt.Sprint(option.Value))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
