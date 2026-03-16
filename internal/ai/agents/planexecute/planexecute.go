package planexecute

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/planexecute/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/planexecute/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/planexecute/replan"
)

func NewPlanExecute(ctx context.Context) (adk.ResumableAgent, error) {

	planner, err := planner.NewPlanner(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create planner: %w", err)
	}

	executor, err := executor.NewExecutor(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	replanner, err := replan.NewReplanner(ctx)
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
