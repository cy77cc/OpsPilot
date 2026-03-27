// Package ai 提供 AI 模块统一入口。
//
// 统一对外暴露 DeepAgents 初始化能力，避免业务层直接依赖内部 agent 实现目录。
package ai

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/change"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/governance"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/host"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/infrastructure"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/kubernetes"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/monitor"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/orchestrator"
)

// InitDeepAgent 初始化 DeepAgents 主入口。
func InitDeepAgent(ctx context.Context) (adk.ResumableAgent, error) {
	return orchestrator.NewOpsPilotAgent(ctx)
}

// AgentFactory 构造单个子 Agent。
type AgentFactory func(ctx context.Context) (adk.Agent, error)

// Registry 返回可直接构建的子 Agent 注册表。
// 该注册表用于集中暴露 agent 入口，避免可用 agent 在包结构中“隐形”。
func Registry() map[string]AgentFactory {
	return map[string]AgentFactory{
		"change":         change.New,
		"host":           host.New,
		"kubernetes":     kubernetes.New,
		"monitor":        monitor.New,
		"governance":     governance.New,
		"infrastructure": infrastructure.New,
	}
}
