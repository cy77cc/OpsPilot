// Package agents 构建 OpsPilot Agent。
//
// 本文件使用 adk.NewChatModelAgent 创建单一 Agent，
// 直接管理工具并自动处理 tool calling loop。
package agents

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	aitools "github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

// Deps 是 Agent 所需的外部依赖。
type Deps struct {
	PlatformDeps  common.PlatformDeps              // 平台服务依赖（数据库、外部 API 等）
	DecisionMaker *airuntime.ApprovalDecisionMaker // 审批决策器
}

// NewAgent 构建并返回 ChatModelAgent。
//
// Agent 直接管理工具，变更类工具通过 Gate 包装实现统一审批拦截。
func NewAgent(ctx context.Context, deps Deps) (adk.ResumableAgent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.2,
	})
	if err != nil {
		return nil, err
	}

	registry := aitools.NewRegistry(deps.PlatformDeps)
	adapter := aitools.NewADKToolAdapter(registry, deps.DecisionMaker)
	tools := adapter.AdaptAll()

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "OpsPilotAgent",
		Instruction: "{" + airuntime.SessionKeyInstruction + "}",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations: 20,
	})
}
