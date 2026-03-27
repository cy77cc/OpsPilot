// Package ai 提供 AI 模块统一入口。
//
// 统一对外暴露 DeepAgents 初始化能力，避免业务层直接依赖内部 agent 实现目录。
package ai

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/orchestrator"
)

// InitDeepAgent 初始化 DeepAgents 主入口。
func InitDeepAgent(ctx context.Context) (adk.ResumableAgent, error) {
	return orchestrator.NewOpsPilotAgent(ctx)
}
