# Hybrid MOE Agent

## Overview

The AI agent routing and execution path has been refactored to a Hybrid MOE design:

1. `ExpertRegistry` loads expert definitions from `configs/experts.yaml`
2. `HybridRouter` selects experts using scene -> keyword -> domain -> default fallback
3. `Orchestrator` builds and executes multi-expert plans
4. `ResultAggregator` merges expert outputs into a final response

## Configuration

- Experts: `configs/experts.yaml`
- Scene mappings: `configs/scene_mappings.yaml`

Both files are resolved from current working directory with upward path fallback, so tests and runtime can load them from different process roots.

## Runtime Integration

`PlatformAgent` now owns:

- `registry` (`experts.ExpertRegistry`)
- `router` (`*experts.HybridRouter`)
- `orchestrator` (`*experts.Orchestrator`)

`Stream()` and `Generate()` route with `HybridRouter` and execute with `Orchestrator`.

## Testing

- Unit tests: registry/router/orchestrator/aggregator behavior
- Integration tests: end-to-end pipeline with real YAML config
- Regression tests: key scene mappings compatibility checks
- Performance baseline: benchmark for routing path (`BenchmarkHybridRouterRoute`)
