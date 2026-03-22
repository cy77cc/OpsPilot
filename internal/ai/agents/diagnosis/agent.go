// Package diagnosis 实现基于 Plan-Execute-Replan 架构的 K8s 只读诊断助手。
//
// 架构：
//
//	PlanExecute
//	  ├── Planner   — 将诊断请求分解为只读调查步骤
//	  ├── Executor  — 仅使用 K8s 只读工具 + 监控工具执行步骤
//	  └── Replanner — 根据执行结果动态调整剩余步骤
//
// 该 Agent 严禁执行任何写操作，所有工具调用均为只读。
package diagnosis

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
)

// NewDiagnosisAgent 创建只读诊断 Agent 实例（PlanExecute 架构）。
//
// 工具集限定为只读 K8s 工具和监控工具，不包含任何写操作工具。
// 最大迭代轮次为 20，防止无限循环。
//
// 参数:
//   - ctx: 上下文（可携带 common.PlatformDeps 供工具使用）
//
// 返回: ResumableAgent 和初始化错误
func NewDiagnosisAgent(ctx context.Context) (adk.ResumableAgent, error) {
	planner, err := newDiagnosisPlanner(ctx)
	if err != nil {
		return nil, fmt.Errorf("diagnosis agent: init planner: %w", err)
	}

	executor, err := newDiagnosisExecutor(ctx)
	if err != nil {
		return nil, fmt.Errorf("diagnosis agent: init executor: %w", err)
	}

	replanner, err := newDiagnosisReplanner(ctx)
	if err != nil {
		return nil, fmt.Errorf("diagnosis agent: init replanner: %w", err)
	}
	loop, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          "execute_replan",
		Description:   "OpsPilot diagnosis execution loop for Kubernetes troubleshooting: run read-only investigation steps with cluster and monitoring tools, then iteratively replan remaining checks based on observed evidence until completion or max iterations.",
		SubAgents:     []adk.Agent{executor, replanner},
		MaxIterations: 20,
	})
	if err != nil {
		return nil, err
	}

	return adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "DiagnosisAgent",
		Description: "OpsPilot Kubernetes diagnosis orchestrator: first build a structured, read-only investigation plan, then drive it through an execute-and-replan loop to identify likely root causes without performing any write operations.",
		SubAgents:   []adk.Agent{planner, loop},
	})
}

// newDiagnosisPlanner 创建诊断专用规划子 Agent。
//
// 使用低温度（0.1）确保规划步骤的确定性。
func newDiagnosisPlanner(ctx context.Context) (adk.Agent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.1,
	})
	if err != nil {
		return nil, err
	}
	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: model,
		GenInputFn:           genPlannerInputFn,
	})
}

// newDiagnosisExecutor 创建诊断专用执行子 Agent。
//
// 工具集仅限只读 K8s 工具 + 监控工具，不包含任何写操作。
func newDiagnosisExecutor(ctx context.Context) (adk.Agent, error) {
	toolset := tools.NewDiagnosisTools(ctx)
	normalizerMW, err := tools.ShadowArgNormalizationToolMiddleware(ctx, toolset)
	if err != nil {
		return nil, fmt.Errorf("diagnosis agent: init tool normalization middleware: %w", err)
	}

	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
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
				Tools:               toolset,
				ToolCallMiddlewares: []compose.ToolMiddleware{normalizerMW},
			},
		},
		GenInputFn: genExecutorInputFn,
	})
}

// newDiagnosisReplanner 创建诊断专用重规划子 Agent。
//
// 使用中等温度（0.3）允许在保守范围内灵活调整诊断步骤。
func newDiagnosisReplanner(ctx context.Context) (adk.Agent, error) {
	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
		Temp:     0.3,
	})
	if err != nil {
		return nil, err
	}
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: model,
	})
}

func genPlannerInputFn(ctx context.Context, userInput []adk.Message) ([]adk.Message, error) {
	msgs, err := prompt.DiagnosisPlannerPrompt.Format(ctx, map[string]any{
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

	msgs, err_ := prompt.DiagnosisExecutorPrompt.Format(ctx, map[string]any{
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
