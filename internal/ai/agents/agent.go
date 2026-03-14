// Package agents 组装 plan-execute Agent 管线。
//
// NewAgent 将 Planner、Executor、Replanner 三个子 Agent 通过
// eino ADK planexecute.New 拼装为可恢复执行的 ResumableAgent。
package agents

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/replan"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

// Deps 是 Agent 管线所需的外部依赖。
type Deps struct {
	PlatformDeps     common.PlatformDeps          // 平台服务依赖（数据库、外部 API 等）
	ContextProcessor *airuntime.ContextProcessor  // 为各阶段 LLM 调用注入上下文信息
}

// NewAgent 构建并返回完整的 plan-execute ResumableAgent。
// 内部依次创建 Planner、Executor、Replanner，任一失败则返回错误。
func NewAgent(ctx context.Context, deps Deps) (adk.ResumableAgent, error) {
	planner, err := planner.NewPlanner(ctx, deps.ContextProcessor)
	if err != nil {
		return nil, fmt.Errorf("failed to create planner: %w", err)
	}

	executor, err := executor.NewExecutor(ctx, deps.PlatformDeps, deps.ContextProcessor)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	replanner, err := replan.NewReplanner(ctx, deps.ContextProcessor)
	if err != nil {
		return nil, fmt.Errorf("failed to create replanner: %w", err)
	}

	return planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      executor,
		Replanner:     replanner,
		MaxIterations: 20,
	})
}
