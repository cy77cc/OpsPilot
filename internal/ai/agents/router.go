// Package agents 提供 AI 助手的统一 Agent 入口。
//
// 架构说明：
//
//	统一入口（HTTP Handler）
//	  └── RouterAgent（LLM 路由）
//	        ├── QAAgent        （ChatModelAgent + RAG 检索工具）
//	        ├── DiagnosisAgent （PlanExecute + 只读 K8s 工具）
//	        └── ChangeAgent    （PlanExecute + 写工具 + HITL 审批）
//
//	定时调度（非用户聊天路由）
//	  └── InspectionAgent（ChatModelAgent + 巡检工具集）
package agents

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/change"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/diagnosis"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/inspection"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/prompt"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/qa"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
)

// NewRouterAgent 创建根路由 Agent（基于 ROUTERPROMPT，仅做意图 Transfer）。
func NewRouterAgent(ctx context.Context) (*adk.ChatModelAgent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.2,
	})
	if err != nil {
		return nil, err
	}
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          "OpsPilotAgent",
		Description:   "OpsPilot infrastructure operations assistant with approval-gated tool execution",
		Instruction:   prompt.ROUTERPROMPT,
		Model:         model,
		MaxIterations: 3,
	})
}

// NewRouter 创建用户聊天主路由。
//
// 注意：InspectionAgent 为调度专用，不参与用户聊天路由，避免巡检任务误被普通问答触发。
func NewRouter(ctx context.Context) (adk.ResumableAgent, error) {
	routerAgent, err := NewRouterAgent(ctx)
	if err != nil {
		return nil, err
	}

	qaAgent, err := qa.NewQAAgent(ctx, nil)
	if err != nil {
		return nil, err
	}

	changeAgent, err := change.NewChangeAgent(ctx)
	if err != nil {
		return nil, err
	}

	diagnosisAgent, err := diagnosis.NewDiagnosisAgent(ctx)
	if err != nil {
		return nil, err
	}

	subagents := []adk.Agent{qaAgent, diagnosisAgent, changeAgent}
	return adk.SetSubAgents(ctx, routerAgent, subagents)
}

// NewInspectionAgent 创建定时巡检 Agent（供调度任务调用）。
func NewInspectionAgent(ctx context.Context) (adk.Agent, error) {
	return inspection.NewInspectionAgent(ctx)
}
