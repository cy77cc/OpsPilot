# Change: refactor-ai-core-for-aiops

## Why

The current AI system is still shaped like a handler-driven chat assistant with tool usage layered onto HTTP and SSE flows. That is not the right foundation for the product direction we have chosen. The target is an AIOps platform with application-card style interactions, operational evidence, approvals, resumable execution, and future domain-specific execution paths. That requires the backend AI core to live in `internal/ai`, not in gateway-side glue code.

We have already identified the desired architectural direction:

- frontend interactions should evolve toward application cards rather than document-style long-form output
- backend orchestration should follow ADK `plan-execute-replan` patterns inspired by the official `eino-examples`
- approvals, execution evidence, and resumable workflows should be part of the AI control plane rather than handler-local behavior

To make that possible, the first backend refactor must re-center the system around `internal/ai` as the owner of orchestration, execution semantics, interrupt handling, and platform-level AI events. `internal/service/ai` should remain the API and SSE gateway, but it should stop being the de facto host for AI orchestration behavior.

## What Changes

- Introduce an AIOps-oriented orchestration core under `internal/ai` as the primary host for AI task orchestration.
- Refactor the current AI execution path so HTTP handlers delegate to the `internal/ai` core instead of effectively coordinating orchestration themselves.
- Establish backend structures in `internal/ai` that can evolve into ADK `plan-execute-replan` orchestration without requiring another handler-centric rewrite later.
- Move execution semantics, interrupt interpretation, and platform event production toward the AI core so they are no longer conceptually owned by gateway code.
- Keep `internal/service/ai` focused on request parsing, auth/session shell concerns, SSE transport, and route compatibility.
- Preserve current `/api/v1/ai` API compatibility in this phase unless a later protocol change explicitly updates the contract.
- Prepare the backend for later application-card streaming and domain executors, but do not require full frontend protocol migration or complete domain decomposition in this change.

## Capabilities

### New Capabilities

- `aiops-orchestrator-core`: defines the AI core responsibility for AIOps-oriented orchestration, interrupt/resume handling, execution semantics, and platform event production inside `internal/ai`

### Modified Capabilities

- `ai-assistant-adk-architecture`: change the ownership model so ADK orchestration is hosted by the AI core rather than by handler-local execution flows
- `ai-control-plane-baseline`: clarify that the AI gateway and the AI core have distinct responsibilities, with orchestration and control-plane semantics centered in `internal/ai`
- `ai-assistant-drawer`: preserve existing assistant compatibility while allowing the backend to transition toward AI core owned operational events

## Impact

- Affected backend code:
  - `internal/ai/**`
  - `internal/service/ai/**`
  - selected wiring in `internal/svc/**`
- Architectural impact:
  - `internal/ai` becomes the true host for orchestration, execution semantics, interrupt/resume coordination, and future domain routing
  - `internal/service/ai` becomes a thinner gateway over the AI core
- External impact:
  - `/api/v1/ai` routes remain compatible in this phase
  - SSE behavior stays backward-compatible where required, while later AIOps event evolution is handled in follow-up changes
- Strategic impact:
  - creates the backend foundation required for future application-card streaming, domain executors, and full AIOps platform behavior
