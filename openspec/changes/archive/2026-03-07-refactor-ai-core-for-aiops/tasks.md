## 1. Establish AI core orchestration host

- [x] 1.1 Inventory the current AI execution path across `internal/ai` and `internal/service/ai`, and identify the orchestration responsibilities that must move under `internal/ai`
- [x] 1.2 Introduce an AIOps-oriented orchestration entrypoint in `internal/ai` that becomes the primary host for AI task coordination
- [x] 1.3 Re-home AI execution semantics, interrupt interpretation, and platform event meaning so they are produced from the AI core instead of gateway-local glue
- [x] 1.4 Shape the new AI core entrypoint so future planner, executor, and replanner roles can evolve there without another handler-centered refactor

## 2. Re-scope the gateway around transport compatibility

- [x] 2.1 Refactor `internal/service/ai` chat and resume flows to delegate orchestration into the AI core entrypoint
- [x] 2.2 Keep `/api/v1/ai` routes and required SSE compatibility behavior stable while the internal ownership changes
- [x] 2.3 Rewire existing approval, preview, execution, and session-oriented control-plane capabilities so they are consumable by the AI core
- [x] 2.4 Remove or simplify gateway-local orchestration behavior that would otherwise duplicate AI core responsibilities

## 3. Verify compatibility and AIOps-readiness

- [x] 3.1 Add or update backend tests to cover gateway delegation, AI core orchestration entry, approval/resume compatibility, and streamed event compatibility
- [x] 3.2 Verify that existing frontend-facing AI flows continue to work without a simultaneous protocol rewrite
- [x] 3.3 Validate that the refactor leaves a clear host in `internal/ai` for follow-up plan-execute-replan and domain executor changes
