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

type Deps struct {
	PlatformDeps     common.PlatformDeps
	ContextProcessor *airuntime.ContextProcessor
}

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
