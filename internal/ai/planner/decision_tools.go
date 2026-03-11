// Package planner 实现 AI 编排的规划阶段。
//
// 本文件定义规划器的决策工具，用于输出结构化决策。
// 包括 clarify、reject、direct_reply、plan 四种决策类型。
package planner

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

// decisionTools 返回所有决策工具。
func decisionTools() []tool.BaseTool {
	return []tool.BaseTool{
		newDecisionTool("clarify", "Emit a final clarify decision when resource targets remain unresolved or ambiguous.", map[string]*schema.ParameterInfo{
			"type": {
				Type:     schema.String,
				Required: true,
				Enum:     []string{"clarify"},
				Desc:     "decision type, must be clarify",
			},
			"message": {
				Type:     schema.String,
				Required: true,
				Desc:     "clarification message for the user",
			},
			"candidates": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
				},
				Desc: "optional candidate resources for clarification",
			},
			"narrative": {
				Type:     schema.String,
				Required: true,
				Desc:     "natural language explanation of why clarify is needed",
			},
		}),
		newDecisionTool("reject", "Emit a final reject decision only when planning determines the request should not proceed at all.", map[string]*schema.ParameterInfo{
			"type": {
				Type:     schema.String,
				Required: true,
				Enum:     []string{"reject"},
				Desc:     "decision type, must be reject",
			},
			"reason": {
				Type:     schema.String,
				Required: true,
				Desc:     "reason for rejection",
			},
			"narrative": {
				Type:     schema.String,
				Required: true,
				Desc:     "natural language explanation of the rejection",
			},
		}),
		newDecisionTool("direct_reply", "Emit a final direct reply decision when no execution plan is needed.", map[string]*schema.ParameterInfo{
			"type": {
				Type:     schema.String,
				Required: true,
				Enum:     []string{"direct_reply"},
				Desc:     "decision type, must be direct_reply",
			},
			"message": {
				Type:     schema.String,
				Required: true,
				Desc:     "final direct reply message",
			},
			"narrative": {
				Type:     schema.String,
				Required: true,
				Desc:     "natural language explanation of why direct reply is sufficient",
			},
		}),
		newDecisionTool("plan", "Emit the final execution plan after resource IDs and constraints are resolved.", map[string]*schema.ParameterInfo{
			"type": {
				Type:     schema.String,
				Required: true,
				Enum:     []string{"plan"},
				Desc:     "decision type, must be plan",
			},
			"narrative": {
				Type:     schema.String,
				Required: true,
				Desc:     "plan-level narrative",
			},
			"plan": {
				Type:     schema.Object,
				Required: true,
				Desc:     "structured execution plan object",
			},
		}),
	}
}

func newDecisionTool(name, desc string, params map[string]*schema.ParameterInfo) tool.BaseTool {
	return decisionTool{
		info: &schema.ToolInfo{
			Name:        name,
			Desc:        desc,
			ParamsOneOf: schema.NewParamsOneOfByParams(params),
		},
	}
}
