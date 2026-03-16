// Package agents 提供 AI 助手的统一 Agent 入口。
//
// 架构说明：
//
//	统一入口（HTTP Handler）
//	  └── IntentRouter（三层策略：规则层 → 模型层 → 策略层）
//	        ├── qa         → QAAgent      （ChatModelAgent + RAG 检索工具）
//	        ├── diagnosis  → DiagnosisAgent（PlanExecute + 只读 K8s 工具）
//	        ├── change     → ChangeAgent   （PlanExecute + 写工具 + HITL 审批）
//	        └── unknown    → 降级 → QAAgent
//
//	定时调度（非 IntentRouter 触发）
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
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
)

// NewRouterAgent 创建根路由 Agent（基于 ROUTERPROMPT，仅做意图 Transfer）。
//
// 该 Agent 保留用于 PRE 子 Agent 委托场景，生产主链路请使用 NewPhase1IntentRouter。
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

// NewRouter 创建包含 PlanExecute 子 Agent 的根路由（旧版，保留兼容）。
func NewRouter(ctx context.Context) (adk.ResumableAgent, error) {
	routerAgent, err := NewRouterAgent(ctx)
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

	inspectionAgent, err := inspection.NewInspectionAgent(ctx)
	if err != nil {
		return nil, err
	}
	subagents := []adk.Agent{changeAgent, diagnosisAgent, inspectionAgent}
	return adk.SetSubAgents(ctx, routerAgent, subagents)
}
