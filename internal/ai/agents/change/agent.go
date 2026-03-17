// Package change 实现基于 Plan-Execute-Replan 架构的 K8s 变更助手。
//
// 架构：
//
//	PlanExecute（Resumable）
//	  ├── Planner   — 规划变更步骤，必须在写操作前插入预检和审批说明
//	  ├── Executor  — 挂载只读工具 + 写操作工具（写操作内置 approvalGate 审批中断）
//	  └── Replanner — 变更后动态调整验证步骤
//
// 写操作工具通过 approvalGate 在执行前触发 adk.Interrupt，
// 暂停 Agent 并创建 AIApprovalTask，等待人工批准后通过 ResumeWithParams 恢复。
//
// Phase 1 说明：写操作工具（scale/restart/rollback/delete_pod）待 Phase 2 实现，
// 当前仅挂载只读工具作为占位，Agent 架构已就绪。
package change

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/ai/tools"
)

// NewChangeAgent 创建变更 Agent 实例（PlanExecute Resumable 架构）。
//
// 该 Agent 必须以 Resumable 模式运行，以支持 HITL 审批中断与恢复：
//   - 写操作工具内置 approvalGate，触发 adk.Interrupt 暂停执行
//   - ResumeWithParams 携带审批结果时恢复
//
// 参数:
//   - ctx: 上下文（应携带 common.PlatformDeps 供工具使用）
//
// 返回: ResumableAgent 和初始化错误
func NewChangeAgent(ctx context.Context) (adk.ResumableAgent, error) {
	planner, err := newChangePlanner(ctx)
	if err != nil {
		return nil, fmt.Errorf("change agent: init planner: %w", err)
	}

	executor, err := newChangeExecutor(ctx)
	if err != nil {
		return nil, fmt.Errorf("change agent: init executor: %w", err)
	}

	replanner, err := newChangeReplanner(ctx)
	if err != nil {
		return nil, fmt.Errorf("change agent: init replanner: %w", err)
	}

	loop, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          "execute_replan",
		Description:   "OpsPilot change execution loop for Kubernetes operations: execute planned steps with available tools and iteratively replan verification or fallback actions based on runtime outcomes until completion or max iterations.",
		SubAgents:     []adk.Agent{executor, replanner},
		MaxIterations: 20,
	})
	if err != nil {
		return nil, err
	}

	return adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "ChangeAgent",
		Description: "OpsPilot Kubernetes change orchestrator: first produce a safe, approval-aware execution plan, then drive the plan through an execute-and-replan loop in a resumable human-in-the-loop workflow.",
		SubAgents:   []adk.Agent{planner, loop},
	})
}

// newChangePlanner 创建变更专用规划子 Agent。
//
// 变更规划要求更严格的确定性（Temp=0.0），避免规划出不必要的高风险步骤。
func newChangePlanner(ctx context.Context) (adk.Agent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.0,
	})
	if err != nil {
		return nil, err
	}
	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: model,
	})
}

// newChangeExecutor 创建变更专用执行子 Agent。
//
// 工具集包含：
//   - 只读 K8s 工具（用于预检和验证步骤）
//   - 写操作工具（Phase 2 实现，每个工具内置 approvalGate）
//
// Phase 1：仅挂载只读工具，写工具待 Phase 2 接入。
func newChangeExecutor(ctx context.Context) (adk.Agent, error) {
	// Phase 2 将调用 tools.NewChangeTools(ctx)，其中包含写操作工具
	// 当前仅使用只读工具，确保 Phase 1 架构验证可通过
	toolset := tools.NewChangeTools(ctx)

	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  120 * time.Second,
		Thinking: false,
		Temp:     0.0,
	})
	if err != nil {
		return nil, err
	}

	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolset,
			},
		},
	})
}

// newChangeReplanner 创建变更专用重规划子 Agent。
//
// 变更后的重规划使用中高温度（0.5），允许根据实际执行结果灵活调整验证步骤。
func newChangeReplanner(ctx context.Context) (adk.Agent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.5,
	})
	if err != nil {
		return nil, err
	}
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model,
	})
}
