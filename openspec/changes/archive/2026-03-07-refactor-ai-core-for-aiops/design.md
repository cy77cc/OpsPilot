## Context

The product direction has shifted from a chat assistant toward an AIOps platform. The backend therefore needs to optimize for operational planning, execution evidence, approval and resume flows, and future card-oriented event streaming. The current implementation still concentrates too much AI behavior around gateway-facing chat execution paths, while the real long-term host for AI orchestration should be `internal/ai`.

This codebase already contains substantial AI substrate inside `internal/ai`, including ADK runner wiring, tools, and agent infrastructure. At the same time, `internal/service/ai` owns HTTP routes, SSE delivery, approval and execution endpoints, and the current assistant transport surface. The refactor must preserve those public surfaces while moving orchestration ownership into the AI core.

Constraints:
- Existing `/api/v1/ai` routes and SSE consumption need to remain compatible during this phase.
- Mutating execution still requires approval and existing control-plane flows must continue to work.
- This change is a foundational refactor, not the full rollout of application-card protocol or domain executors.

## Goals / Non-Goals

**Goals:**
- Make `internal/ai` the primary host for orchestration responsibilities.
- Introduce an explicit AIOps-oriented orchestration core in `internal/ai` that can evolve into ADK `plan-execute-replan`.
- Move platform event semantics and interrupt interpretation toward the AI core instead of gateway-local glue.
- Keep `internal/service/ai` focused on HTTP, auth, request mapping, and SSE transport.
- Preserve current API compatibility while improving architecture ownership boundaries.

**Non-Goals:**
- Fully implement domain-specific executors in this phase.
- Replace the current frontend stream contract with a final card-first protocol.
- Redesign all approval, preview, or execution APIs.
- Complete every future `plan-execute-replan` behavior in one change.

## Decisions

### Decision: `internal/ai` becomes the orchestration host

The primary orchestration host SHALL live in `internal/ai`, not under `internal/service/ai`.

Rationale:
- `internal/ai` is the correct long-term location for task orchestration, interrupt handling, evidence semantics, and ADK integration.
- The gateway package should not be the owner of AI planning or execution behavior.
- Future domain routing, planner/executor/replanner decomposition, and platform event production all belong closer to the AI runtime substrate than to HTTP handlers.

Alternatives considered:
- Keep orchestration in handler-adjacent service packages.
  - Rejected because it keeps the architecture handler-centered and makes later AIOps evolution harder.
- Move everything into a new top-level package outside `internal/ai`.
  - Rejected for now because the repository already treats `internal/ai` as the AI subsystem root.

### Decision: gateway remains transport-only and delegates to the AI core

`internal/service/ai` SHALL remain the compatibility gateway for `/api/v1/ai`, auth/session shell concerns, and SSE writing, but SHALL delegate orchestration to `internal/ai`.

Rationale:
- Preserves existing route structure and frontend integration while changing backend ownership.
- Prevents transport code from continuing to accumulate AI behavior.
- Keeps rollback and compatibility risk lower than a route-level redesign.

Alternatives considered:
- Rewrite public AI APIs alongside the internal refactor.
  - Rejected because it mixes architectural cleanup with user-facing protocol migration.

### Decision: platform event semantics move toward the AI core while SSE framing stays in the gateway

The AI core SHALL own platform-level event meaning, while the gateway SHALL continue to own transport framing and compatibility serialization.

Rationale:
- Event meaning such as plan progress, tool execution semantics, interrupt meaning, and eventual card-oriented events are business concepts.
- SSE flushing, heartbeat, connection lifecycle, and transport compatibility are gateway concerns.

Alternatives considered:
- Keep event translation completely in the gateway.
  - Rejected because that leaves business semantics attached to transport.
- Immediately expose a brand new card-first event family.
  - Rejected for this phase to preserve compatibility and limit rollout risk.

### Decision: refactor toward `plan-execute-replan` shape without requiring complete domain decomposition now

The AI core SHALL be reorganized so that future planner, executor, and replanner roles have a clear host and data flow, but this change SHALL not require the full final AIOps executor split.

Rationale:
- The system needs the right backbone first.
- The current risk is ownership and architectural drift, not the absence of every final execution mode.
- Domain executors are easier to add once orchestration and event ownership live in the AI core.

Alternatives considered:
- Implement host, k8s, service, and monitoring executors immediately.
  - Rejected because it broadens scope before the orchestration host is stable.

### Decision: existing control-plane services remain valid but become AI-core dependencies

Approval, preview, session persistence, and execution record capabilities SHALL remain available, but their usage model SHALL shift so the AI core consumes them instead of gateway code implicitly coordinating them.

Rationale:
- These capabilities are already part of the platform baseline.
- Reusing them avoids unnecessary API churn.
- It creates a clearer control-plane relationship for later orchestration growth.

Alternatives considered:
- Replace all control-plane services during the same change.
  - Rejected because it adds too much migration risk to a boundary refactor.

## Risks / Trade-offs

- [Risk] Architectural refactor without immediate user-facing gains may look like churn. → Mitigation: keep API behavior stable and tie the refactor to the next AIOps-oriented orchestration phase.
- [Risk] Moving orchestration ownership could break existing SSE traces or approval flows. → Mitigation: preserve current route and stream compatibility in this phase and add regression coverage around chat, approval, and resume flows.
- [Risk] The AI core could become another monolith if planner/executor/replanner boundaries remain implicit. → Mitigation: explicitly shape the refactor around future `plan-execute-replan` ownership, even if the first implementation is transitional.
- [Risk] Partial refactor may leave duplicate behavior across gateway and AI core. → Mitigation: treat gateway logic as transport-only and move semantic ownership in a single direction toward `internal/ai`.

## Migration Plan

1. Introduce the AIOps-oriented orchestration host under `internal/ai` and route existing AI entrypoints through it.
2. Re-home orchestration semantics, interrupt interpretation, and platform event production so `internal/service/ai` no longer acts as the orchestration owner.
3. Preserve existing `/api/v1/ai` routes and SSE compatibility while the internal ownership shifts.
4. Verify chat, approvals, preview/execute, and resume flows continue to work through the gateway.
5. Use follow-up changes to introduce explicit planner/executor/replanner behavior, domain executors, and card-first event protocol evolution.

Rollback strategy:
- Because this phase preserves public compatibility, rollback can revert the delegation from gateway to AI core without changing public routes or frontend contracts.

## Open Questions

- Which existing responsibilities in `internal/ai/agent` should become the first-class host for planner/executor/replanner orchestration?
- Should platform event translation live in a dedicated `internal/ai` orchestration package or directly beside runner/agent abstractions?
- Which current SSE event family members should remain compatibility shims versus become first-class AI core events in the next phase?
