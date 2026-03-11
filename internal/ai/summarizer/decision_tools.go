// Package summarizer 实现 AI 编编的总结阶段。
//
// 本文件定义总结器的决策工具，用于输出结构化总结。
package summarizer

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// decisionTool 是决策工具的基础实现。
type decisionTool struct {
	info *schema.ToolInfo
}

// Info 返回工具信息。
func (t decisionTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return t.info, nil
}

// InvokableRun 执行工具调用，直接返回参数作为结果。
func (t decisionTool) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	return argumentsInJSON, nil
}

// summaryDecisionTool 创建总结决策工具。
func summaryDecisionTool() tool.BaseTool {
	return decisionTool{
		info: &schema.ToolInfo{
			Name: "emit_summary",
			Desc: "Emit the final structured summary output for the current execution iteration.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"summary": {
					Type:     schema.String,
					Required: true,
					Desc:     "short structured summary for the frontend summary stage",
				},
				"conclusion": {
					Type: schema.String,
					Desc: "high-level conclusion for the user",
				},
				"headline": {
					Type: schema.String,
					Desc: "one-line final headline for user-facing rendering",
				},
				"key_findings": {
					Type: schema.Array,
					ElemInfo: &schema.ParameterInfo{
						Type: schema.String,
					},
				},
				"resource_summaries": {
					Type: schema.Array,
					ElemInfo: &schema.ParameterInfo{
						Type: schema.String,
					},
				},
				"recommendations": {
					Type: schema.Array,
					ElemInfo: &schema.ParameterInfo{
						Type: schema.String,
					},
				},
				"raw_output_policy": {
					Type: schema.String,
					Desc: "expected raw output rendering policy such as summary_only or excerpt_allowed",
				},
				"next_actions": {
					Type: schema.Array,
					ElemInfo: &schema.ParameterInfo{
						Type: schema.String,
					},
				},
				"need_more_investigation": {
					Type:     schema.Boolean,
					Required: true,
				},
				"narrative": {
					Type:     schema.String,
					Required: true,
				},
				"replan_hint": {
					Type: schema.Object,
				},
			}),
		},
	}
}
