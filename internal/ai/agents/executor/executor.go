// Package executor 封装 plan-execute 架构中的执行子 Agent。
//
// NewExecutor 创建 Executor Agent，其职责是按照 Planner 输出的计划，
// 调用领域工具逐步执行每个步骤并汇报结果。
// 敏感工具（高风险/变更类）在执行前会触发审批中断。
package executor

import (
	"context"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/compose"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/ai/tools"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

// NewExecutor 创建执行 Agent 实例，挂载所有领域工具（含审批 Gate 包装）。
// processor 为 nil 时使用 defaultExecutorInput 构建 LLM 输入。
func NewExecutor(ctx context.Context, deps common.PlatformDeps, processor *airuntime.ContextProcessor) (adk.Agent, error) {
	toolset := tools.NewAllTools(ctx, deps)

	model, err := chatmodel.NewChatModel(ctx, chatmodel.ChatModelConfig{
		Timeout:  60 * time.Second,
		Thinking: false,
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
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			if processor != nil {
				return processor.BuildExecutorInput(ctx, in, toolset)
			}
			return defaultExecutorInput(ctx, in)
		},
	})
}

// defaultExecutorInput 在 ContextProcessor 不可用时构建 Executor 的 LLM 输入。
// 将计划 JSON、已完成步骤和当前步骤格式化后注入 Prompt 模板。
func defaultExecutorInput(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
	planContent, err := in.Plan.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return executorPrompt.Format(ctx, map[string]any{
		"input":          in.UserInput,
		"plan":           string(planContent),
		"executed_steps": in.ExecutedSteps,
		"step":           in.Plan.FirstStep(),
	})
}
