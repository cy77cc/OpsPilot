// Package spec 定义专家系统的核心规范和接口。
//
// 本文件提供专家接口定义、工具能力描述和工具导出结构，
// 是所有专家实现的基础契约。
package spec

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
)

// ToolCapability 描述单个工具的能力特征。
//
// 用于向规划器声明工具的操作模式和风险等级，支持智能决策。
type ToolCapability struct {
	Name        string `json:"name"`                  // 工具能力名称
	Description string `json:"description,omitempty"` // 能力描述
	Mode        string `json:"mode"`                  // 操作模式: readonly（只读）/ mutating（变更）
	Risk        string `json:"risk"`                  // 风险等级: low（低）/ medium（中）/ high（高）
}

// ToolExport 专家的工具导出信息，用于规划器决策。
//
// 包含专家名称、描述和能力清单，使规划器能够选择合适的专家。
type ToolExport struct {
	Name         string           `json:"name"`         // 专家名称（如 "hostops_expert"）
	Description  string           `json:"description"`  // 专家描述
	Capabilities []ToolCapability `json:"capabilities"` // 能力清单
}

// Expert 定义专家的核心接口。
//
// 专家是具备特定领域工具集的智能代理，负责：
//   - 声明自己的名称、描述和能力
//   - 提供可调用的工具集
//   - 导出工具目录供规划器使用
type Expert interface {
	// Name 返回专家的唯一标识名称。
	Name() string

	// Description 返回专家的功能描述。
	Description() string

	// Capabilities 返回专家提供的工具能力清单。
	Capabilities() []ToolCapability

	// Tools 返回专家的可调用工具列表。
	Tools(ctx context.Context) []tool.InvokableTool

	// AsTool 将专家导出为工具目录条目。
	AsTool() ToolExport
}

// FilterToolsByName 从工具列表中排除指定名称的工具。
//
// 用于精细化控制专家暴露的工具集，例如排除危险操作或重复工具。
//
// 参数:
//   - ctx: 上下文
//   - tools: 原始工具列表
//   - excluded: 要排除的工具名称
//
// 返回:
//   - 过滤后的工具列表
func FilterToolsByName(ctx context.Context, tools []tool.InvokableTool, excluded ...string) []tool.InvokableTool {
	if len(excluded) == 0 {
		return tools
	}
	// 构建排除名称集合
	blocked := make(map[string]struct{}, len(excluded))
	for _, name := range excluded {
		if name != "" {
			blocked[name] = struct{}{}
		}
	}
	// 过滤工具列表
	out := make([]tool.InvokableTool, 0, len(tools))
	for _, invokable := range tools {
		if invokable == nil {
			continue
		}
		info, err := invokable.Info(ctx)
		if err != nil || info == nil {
			// 无法获取信息的工具保留
			out = append(out, invokable)
			continue
		}
		if _, skip := blocked[info.Name]; skip {
			// 在排除列表中，跳过
			continue
		}
		out = append(out, invokable)
	}
	return out
}
