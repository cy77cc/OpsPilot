package experts

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Orchestrator struct {
	registry   ExpertRegistry
	executor   *ExpertExecutor
	aggregator *ResultAggregator
}

func NewOrchestrator(registry ExpertRegistry, aggregator *ResultAggregator) *Orchestrator {
	if aggregator == nil {
		aggregator = NewResultAggregator(AggregationTemplate, nil)
	}
	return &Orchestrator{
		registry:   registry,
		executor:   NewExpertExecutor(registry),
		aggregator: aggregator,
	}
}

func (o *Orchestrator) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
	if req == nil || req.Decision == nil {
		return nil, fmt.Errorf("route decision is required")
	}
	runCtx := ctx
	if timeoutMS, ok := runtimeTimeout(req.RuntimeContext); ok {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, timeoutMS)
		defer cancel()
	}

	plan := o.buildPlan(req.Decision)
	results, err := o.executePlan(runCtx, plan, req)
	resp, aggErr := o.aggregateResults(runCtx, results, req)
	if aggErr != nil && err == nil {
		err = aggErr
	}
	traces := make([]ExpertTrace, 0, len(results))
	for _, item := range results {
		status := "ok"
		if item.Error != nil {
			status = "failed"
		}
		traces = append(traces, ExpertTrace{
			ExpertName: item.ExpertName,
			Output:     item.Output,
			Duration:   item.Duration,
			Status:     status,
		})
	}
	return &ExecuteResult{
		Response: resp,
		Traces:   traces,
		Metadata: map[string]any{
			"strategy": req.Decision.Strategy,
			"source":   req.Decision.Source,
		},
	}, err
}

func (o *Orchestrator) buildPlan(decision *RouteDecision) *ExecutionPlan {
	steps := make([]ExecutionStep, 0, 1+len(decision.HelperExperts))
	steps = append(steps, ExecutionStep{
		ExpertName: decision.PrimaryExpert,
		Task:       "primary analysis",
	})
	switch decision.Strategy {
	case StrategyParallel:
		for _, helper := range decision.HelperExperts {
			steps = append(steps, ExecutionStep{
				ExpertName: helper,
				Task:       "helper analysis",
			})
		}
	case StrategySequential:
		for i, helper := range decision.HelperExperts {
			steps = append(steps, ExecutionStep{
				ExpertName:  helper,
				Task:        "helper analysis",
				DependsOn:   []int{i},
				ContextFrom: []int{i},
			})
		}
	default:
	}
	return &ExecutionPlan{Steps: steps}
}

func (o *Orchestrator) executePlan(ctx context.Context, plan *ExecutionPlan, req *ExecuteRequest) ([]ExpertResult, error) {
	if o == nil || o.executor == nil {
		return nil, fmt.Errorf("executor is nil")
	}
	if plan == nil || len(plan.Steps) == 0 {
		return nil, fmt.Errorf("execution plan is empty")
	}
	if req == nil || req.Decision == nil {
		return nil, fmt.Errorf("request decision is empty")
	}

	results := make([]ExpertResult, 0, len(plan.Steps))
	switch req.Decision.Strategy {
	case StrategyParallel:
		type resultWithIndex struct {
			idx int
			val ExpertResult
		}
		ch := make(chan resultWithIndex, len(plan.Steps))
		var wg sync.WaitGroup
		for idx := range plan.Steps {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				step := plan.Steps[i]
				res, err := o.executor.ExecuteStep(ctx, &step, nil, req.Message)
				if res == nil {
					res = &ExpertResult{ExpertName: step.ExpertName, Error: err}
				}
				if err != nil {
					res.Error = err
				}
				ch <- resultWithIndex{idx: i, val: *res}
			}(idx)
		}
		wg.Wait()
		close(ch)
		ordered := make([]ExpertResult, len(plan.Steps))
		for item := range ch {
			ordered[item.idx] = item.val
		}
		results = append(results, ordered...)
		return results, nil
	default:
		var firstErr error
		for i := range plan.Steps {
			step := plan.Steps[i]
			res, err := o.executor.ExecuteStep(ctx, &step, results, req.Message)
			if res == nil {
				res = &ExpertResult{ExpertName: step.ExpertName, Error: err}
			}
			if err != nil && firstErr == nil {
				firstErr = err
				res.Error = err
			}
			results = append(results, *res)
		}
		return results, firstErr
	}
}

func (o *Orchestrator) aggregateResults(ctx context.Context, results []ExpertResult, req *ExecuteRequest) (string, error) {
	if o == nil || o.aggregator == nil {
		return "", fmt.Errorf("aggregator is nil")
	}
	query := ""
	if req != nil {
		query = req.Message
	}
	return o.aggregator.Aggregate(ctx, results, query)
}

func runtimeTimeout(runtime map[string]any) (time.Duration, bool) {
	if len(runtime) == 0 {
		return 0, false
	}
	raw, ok := runtime["timeout_ms"]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case int:
		if v > 0 {
			return time.Duration(v) * time.Millisecond, true
		}
	case int64:
		if v > 0 {
			return time.Duration(v) * time.Millisecond, true
		}
	case float64:
		if v > 0 {
			return time.Duration(v) * time.Millisecond, true
		}
	}
	return 0, false
}
