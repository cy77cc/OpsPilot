# 2026-04-17 AI Architecture Rebuild Design

## Context

This document defines a full rebuild plan for the AI module based on:

- `ai_review.md` findings (P0/P1/P2 issues)
- current codebase verification in:
  - `internal/bootstrap/modules.go`
  - `internal/modules/ai/logic/chat/chat.go`
  - `internal/modules/ai/logic/chat/projection.go`
  - `internal/modules/ai/handler/chat/handler.go`
  - `web/src/api/modules/ai.ts`
  - `web/src/components/AI/pendingRunStore.ts`

User constraints confirmed:

- Scope: `P0 + P1 + P2` full coverage.
- Strategy: "step-by-step but rebuild toward final architecture".
- Compatibility: no backward-compatibility requirement.
- Data: database is rebuildable; no historical preservation requirement.

## Problem Statement

Current AI architecture has these high-impact issues:

1. Uncontrolled lifecycle for background workers (`context.Background()` in bootstrap).
2. Projection/content persistence based on full event replay per run, causing long-session degradation.
3. Full session history loaded into model input without budget/select strategy.
4. Trace field exists but lacks strict end-to-end propagation.
5. Frontend API surface diverges from backend capabilities (`notImplementedByBackend` public methods).
6. SSE error branches still expose internal raw errors.
7. Module boundaries and naming drift from effective runtime ownership.
8. Pending run metadata persisted in localStorage with cross-tab/account contamination risk.

## Goals and Non-Goals

### Goals

1. Rebuild AI module into clean layers: `domain/app/runtime/infra/interfaces`.
2. Replace full projection rebuild with incremental projection updates.
3. Implement context engineering baseline: `select + compress + isolate`.
4. Enforce unified error-code contract for all API/SSE exits.
5. Establish trace and metrics closed-loop across request-run-tool-approval-worker.
6. Rebuild frontend AI module to expose only real backend capabilities.
7. Remove obsolete code paths after each rebuild stage.

### Non-Goals

1. Backward compatibility for existing API contracts.
2. One-time migration for historical AI data.
3. Non-AI module refactors unrelated to AI runtime, stream, approval, and frontend AI consumption.

## Approaches Considered

### Approach A: Big-bang rebuild

- Pros: cleanest final state quickly.
- Cons: high integration risk, poor fault isolation.

### Approach B: Multi-stage rebuild with aggressive cleanup (Recommended)

- Pros: controllable risk; each stage is runnable and testable; still converges to clean architecture.
- Cons: temporary transition code exists inside stage boundaries.

### Approach C: Skeleton-first then gradual migration

- Pros: structure first, low initial design ambiguity.
- Cons: function lag and delayed runtime validation.

Recommendation: Approach B.

## Target Architecture

### Backend

```text
internal/modules/ai/
  domain/
    run/
    session/
    approval/
    memory/
  app/
    command/
      chat_command_handler.go
      approval_command_handler.go
    query/
      session_query_service.go
      projection_query_service.go
  runtime/
    orchestrator/
    context/
      selector.go
      compressor.go
      isolator.go
    streaming/
      sse_adapter.go
      event_mapper.go
  infra/
    persistence/
      dao/
    observability/
      tracing.go
      metrics.go
    workers/
      approval_worker.go
      approval_expirer.go
  interfaces/
    http/
      chat_handler.go
      approval_handler.go
```

### Frontend

```text
web/src/features/ai/
  api/
    chatApi.ts
    sessionApi.ts
    runApi.ts
    approvalApi.ts
  stream/
    streamClient.ts
    eventDispatcher.ts
    reconnectController.ts
  state/
    runtimeStore.ts
    pendingRunStore.ts
  ui/
    CopilotSurface/
    AssistantReply/
```

## Core Runtime Design

### Main Flow

1. `interfaces/http/chat_handler` validates input and injects `trace_id`.
2. `app/command/chat_command_handler` creates run and starts execution.
3. `runtime/orchestrator` executes `deep_main` with on-demand readonly specialists.
4. `runtime/context` applies budgeted selection/compression/isolation before model invocation.
5. Runtime events append to `ai_run_events`.
6. Incremental projection updater updates `ai_run_projections` per event batch.
7. SSE outputs only mapped public event payloads and standardized errors.

### Incremental Projection Rules

1. Projection updates are event-driven, not rebuilt from full history on completion.
2. Projection version increments monotonically.
3. Projection writes are idempotent on `(run_id, version)` guards.
4. Content artifacts persist only for new blocks not previously written.

### Context Budget Rules

Default policy:

- `recent_n = 12` conversational turns.
- pinned constraints always included.
- semantic recall enabled for task-relevant memory.
- when token budget exceeded, oldest non-pinned context summarized first.

## Error Contract

### Outbound Policy

1. No handler may return raw `err.Error()` to client.
2. SSE errors map to `AI_STREAM_*`.
3. JSON API errors map to `AI_API_*`.
4. Public payload must include: `code`, `message`, `trace_id`, `retryable`.

### Internal Policy

- Internal logs and traces keep full detail.
- Public errors remain stable and non-sensitive.

## Observability and Governance

### Trace

`trace_id` required at:

1. request ingress
2. run row creation
3. tool call events
4. approval outbox record
5. worker execution
6. SSE terminal events

### Metrics

Minimum stage-gate metrics:

1. `ai_run_end_to_end_latency_ms`
2. `ai_projection_update_latency_ms`
3. `ai_approval_success_total`
4. `ai_approval_failure_total`
5. `ai_approval_retry_total`
6. `ai_stream_error_total`

## Multi-Stage Implementation Plan

### Stage 0: Freeze and baseline (0.5 day)

1. Freeze old AI feature additions.
2. Create explicit old-code deletion checklist.
3. Add baseline smoke checks for chat mainline.

Exit criteria:

- repository compiles
- baseline tests pass
- deletion checklist committed

### Stage 1: Backend skeleton rebuild (1-2 days)

1. Create target layered package structure.
2. Wire new `chat_handler -> chat_command_handler -> orchestrator` path.
3. Replace worker startup with controllable service lifecycle context.

Exit criteria:

- new path starts successfully
- worker exits on service shutdown

### Stage 2: Runtime core rebuild (2-3 days)

1. Implement context selector/compressor/isolator.
2. Replace projection full-replay with incremental updater.
3. Move approval worker/expirer under infra workers with retry/timeout policy.

Exit criteria:

- long-session latency no longer degrades linearly
- token budget enforcement works
- approval flow stable in integration tests

### Stage 3: Observability and error governance (1-2 days)

1. End-to-end `trace_id` propagation.
2. Unified error-code mapping for API/SSE.
3. Add metrics and alert thresholds for approval and stream failures.

Exit criteria:

- single `trace_id` can reconstruct full run chain
- no raw internal errors in client payload

### Stage 4: Frontend AI rebuild (2-3 days)

1. Split AI API by resource and remove non-implemented public methods.
2. Centralize stream/reconnect/event mapping.
3. Rework pending run state to in-memory default (no localStorage dependency).
4. Refactor oversized UI files into feature-local modules.

Exit criteria:

- frontend calls only implemented backend APIs
- reconnect and approval resume behavior passes regression checks

### Stage 5: Legacy path removal (1 day)

1. Delete obsolete routing and dead defaults.
2. Remove replaced logic under old chat/projection paths.
3. Remove transition-only flags and adapters.

Exit criteria:

- no legacy AI runtime paths referenced
- compile/test pass with new path only

### Stage 6: Stabilization and docs (1 day)

1. Complete unit/integration/regression coverage for critical scenarios.
2. Publish architecture, error-code, and troubleshooting docs.
3. Produce prioritized backlog for next iteration.

Exit criteria:

- CI green
- docs sufficient for independent continuation

## Testing Strategy

### Unit Tests

1. context budget selector and summarization trigger.
2. projection incremental versioning and idempotency.
3. error mapper internal-to-public conversion.

### Integration Tests

1. chat request to run completion.
2. tool approval wait/resume flow.
3. worker start/stop/retry timeout behavior.

### Regression Tests

1. SSE reconnect with `last_event_id`.
2. long-session run performance under synthetic load.
3. frontend pending run lifecycle and approval recovery.

## Risks and Mitigations

1. Risk: transition complexity across stages.
   - Mitigation: per-stage deletion list and strict exit gates.
2. Risk: hidden coupling in old chat/projection code.
   - Mitigation: build new vertical slice first, then prune old modules.
3. Risk: test gap during large movement.
   - Mitigation: baseline smoke before rebuild; mandatory stage-level integration checks.

## Acceptance Criteria

1. P0/P1/P2 findings in `ai_review.md` are either removed by deletion or covered by new implementation.
2. New architecture directories own runtime responsibilities with no shadow duplicate chain.
3. All client-facing AI errors use standardized code-based contract.
4. Worker lifecycle is service-controlled and observable.
5. Projection update is incremental and verifiably bounded for long runs.

