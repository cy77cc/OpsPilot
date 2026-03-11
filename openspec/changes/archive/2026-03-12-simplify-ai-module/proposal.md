# Proposal: Simplify AI Module

**Date**: 2026-03-12
**Status**: Draft
**Change**: simplify-ai-module

---

## Why

The AI module (`internal/ai/`) has accumulated three categories of unnecessary complexity:

1. **Duplicate type definitions**: `executor.EventMeta` and `events.EventMeta` are structurally identical, forcing manual field-by-field conversion in `orchestrator.go` every time an executor event is bridged to the SSE stream.

2. **Redundant data model**: `ResolvedResources` carries both flat scalar fields (`ServiceName`, `ServiceID`, `ClusterName`, `ClusterID`, `HostNames []string`, `HostIDs []int`, `PodName`) and list fields (`Services []ResourceRef`, `Clusters []ResourceRef`, `Hosts []ResourceRef`, `Pods []PodRef`) for the same resources. This forces `canonicalizePlan` to manage 11 separate fallback blocks instead of 5, and causes `populateStepInput` to use inconsistent access patterns.

3. **Single 1302-line file**: `planner/planner.go` conflates five distinct responsibilities — type definitions, core planning logic, LLM output parsing, plan normalization/validation, and resource collection from rewrite output — making it hard to navigate and extend.

None of these issues represent missing functionality. They are structural inefficiencies that increase cognitive load and the risk of bugs when the module evolves.

---

## What Changes

### A — Unify EventMeta (B in exploration)

Remove `executor.EventMeta`. The `executor` package will directly use `events.EventMeta`.
`executor.EventEmitter` signature updates to accept `events.EventMeta`.
The manual conversion block in `orchestrator.go:522–546` is deleted.

**Affected files**: `executor/executor.go`, `executor/events.go`, `orchestrator.go`

### B — Simplify ResolvedResources (A in exploration)

Remove flat scalar fields from `ResolvedResources`:
- `ServiceName string`, `ServiceID int`
- `ClusterName string`, `ClusterID int`
- `HostNames []string`, `HostIDs []int`
- `PodName string`

Keep only: `Services []ResourceRef`, `Clusters []ResourceRef`, `Hosts []ResourceRef`, `Pods []PodRef`, `Namespace string`, `Scope *ResourceScope`.

Add package-level helpers (`primaryID`, `primaryName`, `allIDs`) to replace the flat-field access pattern.

`parseResolvedResources` continues to parse both flat and list fields from LLM JSON output, then normalises to the list-only struct.
`canonicalizePlan` reduces from 11 fallback blocks to 5.
`populateStepInput` uses the helper functions consistently.

**Affected files**: `planner/planner.go` (types + parse + normalize + validate + collect functions), any callers of `ResolvedResources` fields

### C — Split planner.go (C in exploration)

Move code from `planner/planner.go` into focused files within the same package:

| File | Responsibility | Approx. lines |
|------|---------------|---------------|
| `planner.go` | `Planner` struct, `Plan`, `PlanStream`, `plan` | ~100 |
| `types.go` | All exported types (`Decision`, `ExecutionPlan`, `PlanStep`, `ResolvedResources`, etc.) | ~200 |
| `parse.go` | `ParseDecision`, `parseExecutionPlan`, `parseResolvedResources`, `looseStringValue`, etc. | ~350 |
| `normalize.go` | `normalizeDecision`, `canonicalizePlan`, `populateStepInput`, `validatePlanPrerequisites`, helper funcs | ~350 |
| `collect.go` | `buildBasePlanContext`, `collectHostNames`, `collectPods`, `detectScope`, etc. | ~200 |

No logic changes. Same package. Existing tests continue to pass without modification.

---

## Capabilities

No new capabilities. This change touches internal implementation only.
Spec files are not required (no behavior change visible outside `internal/ai/`).

---

## Impact

| Path | Change |
|------|--------|
| `internal/ai/executor/executor.go` | Remove `EventMeta` struct and `EventEmitter` type; import `events` |
| `internal/ai/executor/events.go` | Update function signatures to use `events.EventMeta` |
| `internal/ai/orchestrator.go` | Remove manual EventMeta conversion (~24 lines); update call sites |
| `internal/ai/planner/planner.go` | Types + logic moved to new files; becomes ~100 lines |
| `internal/ai/planner/types.go` | New file (split from planner.go) |
| `internal/ai/planner/parse.go` | New file (split from planner.go) |
| `internal/ai/planner/normalize.go` | New file (split from planner.go); `canonicalizePlan` shrinks ~30 lines |
| `internal/ai/planner/collect.go` | New file (split from planner.go) |
| `internal/ai/planner/planner_test.go` | No changes needed (same package) |
| `internal/ai/planner/support.go` | Possibly absorb into `normalize.go` or `collect.go` |

**Risk**: Low. No external API contracts change. No Redis schema changes (ExecutionState and SessionSnapshot are unaffected; ResolvedResources is inside ExecutionPlan which is stored, but we remove fields that were redundant copies — downstream readers of old stored state that used the flat fields will see zero-values, which is the same as "not resolved".)
