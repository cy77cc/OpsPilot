// Package change 实现基于 Plan-Execute-Replan 架构的 K8s 变更助手。
//
// 架构：
//
//	PlanExecute（Resumable）
//	  ├── Planner   — 规划变更步骤，必须在写操作前插入预检和审批说明
//	  ├── Executor  — 挂载只读工具 + 写操作工具（写操作内置 approvalGate 审批中断）
//	  └── Replanner — 变更后动态调整验证步骤
//
// 写操作工具通过 ApprovalMiddleware 在执行前触发 tool.StatefulInterrupt，
// 暂停 Agent 并发送 tool_approval 事件，等待人工批准后通过 ResumeWithParams 恢复。
//
// HITL (Human-in-the-Loop) 工作流:
//  1. Executor 调用高风险工具（如 host_batch, k8s_delete_pod）
//  2. ApprovalMiddleware 拦截调用，触发 StatefulInterrupt
//  3. Runner 检测到中断，通过 SSE 发送 tool_approval 事件给前端
//  4. 用户在前端审批界面确认或拒绝
//  5. 审批结果通过 API 携带 ApprovalResult 调用 ResumeWithParams 恢复执行
//  6. ApprovalMiddleware 根据审批结果决定继续执行或返回拒绝消息
package change

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/prompt"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/middleware"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
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
		GenInputFn:           genPlannerInputFn,
	})
}

// newChangeExecutor 创建变更专用执行子 Agent。
//
// 工具集包含：
//   - 只读 K8s 工具（用于预检和验证步骤）
//   - 写操作工具（Phase 2 实现，每个工具内置 approvalGate）
//
// 审批中间件集成说明:
//   - 高风险工具（host_batch, k8s_delete_pod 等）通过 ApprovalMiddleware 拦截
//   - 拦截后触发 StatefulInterrupt，暂停执行并等待人工审批
//   - 审批通过后通过 ResumeWithParams 恢复执行
//
// Phase 1：仅挂载只读工具，写工具待 Phase 2 接入。
func newChangeExecutor(ctx context.Context) (adk.Agent, error) {
	// Phase 2 将调用 tools.NewChangeTools(ctx)，其中包含写操作工具
	// 当前仅使用只读工具，确保 Phase 1 架构验证可通过
	toolset := tools.NewChangeTools(ctx)
	normalizerMW, err := tools.ShadowArgNormalizationToolMiddleware(ctx, toolset)
	if err != nil {
		return nil, fmt.Errorf("change agent: init tool normalization middleware: %w", err)
	}

	// 创建审批中间件，用于拦截高风险工具调用
	approvalMW := tools.ApprovalToolMiddleware(nil)
	if svcCtx, ok := runtimectx.ServicesAs[*svc.ServiceContext](ctx); ok && svcCtx != nil && svcCtx.DB != nil {
		approvalMW = tools.ApprovalToolMiddleware(&middleware.ApprovalMiddlewareConfig{
			Orchestrator: common.NewApprovalOrchestrator(svcCtx.DB),
		})
	}

	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  120 * time.Second,
		Thinking: false,
		Temp:     0.0,
	})
	if err != nil {
		return nil, err
	}

	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model:         model,
		MaxIterations: 24,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: toolset,
				// 注册审批中间件，拦截高风险工具调用
				ToolCallMiddlewares: []compose.ToolMiddleware{normalizerMW, approvalMW},
			},
		},
		GenInputFn: genExecutorInputFn,
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

func genPlannerInputFn(ctx context.Context, userInput []adk.Message) ([]adk.Message, error) {
	msgs, err := prompt.ChangePlannerPrompt.Format(ctx, map[string]any{
		"input": userInput,
	})
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func genExecutorInputFn(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
	planContent, err_ := in.Plan.MarshalJSON()
	if err_ != nil {
		return nil, err_
	}

	firstStep := in.Plan.FirstStep()

	msgs, err_ := prompt.ChangeExecutorPrompt.Format(ctx, map[string]any{
		"input":          in.UserInput[0].Content,
		"plan":           string(planContent),
		"executed_steps": formatExecutedSteps(in.ExecutedSteps),
		"step":           firstStep,
	})
	if err_ != nil {
		return nil, err_
	}

	return msgs, nil
}

func formatExecutedSteps(in []planexecute.ExecutedStep) string {
	const (
		maxPromptSteps       = 5
		maxResultRunes       = 600
		truncatedResultLabel = "...<truncated>"
	)

	var sb strings.Builder
	total := len(in)
	if total == 0 {
		return "Completed 0 step(s). No previous step results."
	}

	start := 0
	if total > maxPromptSteps {
		start = total - maxPromptSteps
	}

	_, _ = fmt.Fprintf(&sb, "Completed %d step(s). Showing the latest %d step(s).\n\n", total, total-start)
	for idx, m := range in[start:] {
		_, _ = fmt.Fprintf(&sb, "## %d. Step: %v\n  Result: %s\n\n", start+idx+1, m.Step, truncateForPrompt(m.Result, maxResultRunes, truncatedResultLabel))
	}
	return sb.String()
}

func truncateForPrompt(input string, maxRunes int, suffix string) string {
	if maxRunes <= 0 || utf8.RuneCountInString(input) <= maxRunes {
		return input
	}
	if maxRunes <= len(suffix) {
		return suffix[:maxRunes]
	}

	runes := []rune(input)
	return string(runes[:maxRunes-len([]rune(suffix))]) + suffix
}
