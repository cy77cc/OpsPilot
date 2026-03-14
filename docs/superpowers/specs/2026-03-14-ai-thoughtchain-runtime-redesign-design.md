# AI ThoughtChain Runtime Redesign

## Context

The current AI module mixes multiple runtime models and approval paths:

- frontend renders both legacy `thoughtChain` data and newer `turn/block + phase/step` state
- backend emits overlapping SSE event families for chain progress, phase changes, and approval prompts
- approval can be triggered through multiple semantics and resume paths
- recommended prompts in a new conversation can enter an invalid empty-state race and show "AI assistant unavailable" even while the request is already in flight
- observability is fragmented and does not expose one consistent runtime lifecycle to Prometheus

This overlap has produced the four concrete problems reported by the user:

1. `thoughtChain` display does not correctly represent the real `plan -> execute -> replan` lifecycle and its styling is inconsistent.
2. Approval is not reliably triggered; when a tool requires approval the flow can exit instead of pausing and waiting.
3. Clicking a recommended prompt in a fresh conversation can briefly show a false unavailable state until the page is refreshed.
4. Callback-based telemetry is insufficient, reducing observability.

The user explicitly wants `thoughtChain` to become the new runtime contract, and wants legacy runtime structures removed first so they do not continue shaping new work.

## Goals

- make `thoughtChain` the only runtime protocol exposed to the AI chat frontend
- remove legacy `turn/block`, `phase/step`, and old approval-entry semantics from the primary chat path
- redesign the frontend runtime around a single `thoughtChain` store and an upgraded chain UI
- model approval as a first-class chain node and unify approval actions under one API
- add callback-driven metrics and tracing hooks that can be exported to the existing Prometheus integration
- prevent legacy protocol concepts from re-entering the implementation

## Non-Goals

- redesigning unrelated AI business prompts, expert routing, or RAG behavior
- broad data-model refactors outside what is needed to support the new chain runtime
- preserving long-term dual compatibility between new and old runtime protocols

## Design Principles

### 1. ThoughtChain Is The Source Of Truth

`thoughtChain` is the only runtime model for chat execution, replay, approval, and frontend rendering.

Everything else is derivative or removed.

### 2. Delete Old Guidance Before Building New Guidance

Legacy runtime entry points must be cut out of the main path before the new runtime is attached. The migration order is intentionally:

1. identify and isolate old entry points
2. remove old primary-path dependencies
3. build the new `thoughtChain` runtime on the cleaned boundary

This avoids implementing the new runtime under pressure from obsolete abstractions.

### 3. One Runtime, One Approval Story, One Observability Story

The frontend, backend, persistence, approval recovery, telemetry, and replay flow must all describe the same chain and the same node lifecycle.

## Runtime Architecture

Each assistant response owns one `thoughtChain`.

A chain contains ordered nodes that describe execution progress. Node kinds are:

- `plan`
- `step`
- `tool`
- `approval`
- `replan`
- `answer`
- `status` only when a system-level status note is needed; it must not carry core business flow

Each node has:

- stable identity: `chain_id`, `node_id`, optional `parent_node_id`
- display metadata: `kind`, `title`, `summary`
- lifecycle state: `pending`, `running`, `waiting`, `success`, `error`, `aborted`
- typed payload data for rendering and telemetry

The chain has:

- `chain_id`
- `session_id`
- `message_id`
- `trace_id`
- overall status: `idle`, `streaming`, `paused_for_approval`, `resuming`, `completed`, `failed`

## Streaming Contract

The chat stream exposes only `thoughtChain` lifecycle events.

### Chain Events

- `chain_started`
- `chain_meta`
- `chain_paused`
- `chain_resumed`
- `chain_completed`
- `chain_error`

### Node Events

- `node_open`
- `node_delta`
- `node_replace`
- `node_close`

### Payload Guidance

- `plan`: plan title, plan summary, proposed steps, constraints
- `step`: step order, current progress, completion notes
- `tool`: tool name, input summary, output summary, duration, risk summary
- `approval`: approval target, approval summary, preview, risk level, action token or action reference
- `replan`: trigger reason, affected prior nodes, new plan summary
- `answer`: final markdown content, optional follow-up recommendations

No legacy `turn_started`, `block_open`, `phase_started`, `step_started`, `approval_required`, or similar event types remain on the primary streaming path.

## Frontend Runtime And UX

The frontend keeps one `thoughtChain` store per assistant response.

The assistant message UI is rendered only from chain state. It must not merge or reconcile multiple runtime sources such as legacy `thoughtChain` arrays, `turn/block` projections, or separate approval cards from another store.

### Interaction Model

- the top of the assistant area shows a runtime timeline based on ordered nodes
- each node renders as a dedicated expandable card
- the `answer` node is visually separated from process nodes
- the `approval` node is visually emphasized and provides direct action controls
- `replan` clearly explains why the plan changed and what was replaced

### Recommended Prompt Fix

When the user clicks a recommended prompt in a fresh conversation, the frontend immediately creates:

- the user message
- a placeholder assistant chain container in `connecting` or `streaming` state

This avoids the transient "AI assistant unavailable" empty-state path before the first server event arrives.

### Presentation Requirement

The upgraded UI must present the real chain semantics instead of dumping serialized JSON or flattened text blobs. The display should make `plan -> step/tool -> approval -> replan -> answer` easy to scan as one continuous runtime.

## Approval Model

Approval is always represented as an independent `approval` node.

When a tool requires approval:

1. the current execution context pauses
2. an `approval` node opens in `waiting` state
3. the chain emits `chain_paused`
4. the frontend renders inline approval actions

When the user decides:

- the frontend calls one decision API
- the backend updates the approval node status
- the backend resumes the same chain execution context
- the stream emits `chain_resumed`

### Unified Approval API

Only one primary approval action should remain for chat runtime decisions:

`POST /api/v1/ai/chains/{chain_id}/approvals/{node_id}/decision`

Request body:

- `approved`
- `reason`

The chat frontend must not depend on parallel approval semantics such as old approval ticket confirmation endpoints or detached resume flows once migration is complete.

## Observability And Callback Design

Observability is built around chain and node lifecycle hooks, then exported to Prometheus through the system's existing monitoring integration.

### Callback Hooks

- `OnChainStarted`
- `OnNodeOpened`
- `OnNodeUpdated`
- `OnApprovalRequired`
- `OnApprovalResolved`
- `OnReplanTriggered`
- `OnChainCompleted`
- `OnChainFailed`

### Metrics

- chain total, completion, failure, and duration
- time to first chain event
- time to final answer
- approval trigger count, approval wait duration, approval approve/reject rate
- replan trigger count
- tool-node duration and outcome

Suggested labels:

- `scene`
- `node_kind`
- `tool`
- `status`
- `approval_outcome`

### Trace And Span Model

Each chain and each significant node lifecycle should emit trace/span context keyed by:

- `trace_id`
- `chain_id`
- `node_id`
- `session_id`
- `tool`
- `scene`
- `status`

Suggested span names:

- `chain.start`
- `node.plan`
- `node.step`
- `node.tool`
- `node.approval.wait`
- `node.replan`
- `chain.complete`

## Deletion Plan

Legacy architecture must be removed before the new runtime becomes the default path.

### Remove From Main Flow

- legacy `turn/block` chat rendering path
- legacy `phase/step` runtime lifecycle path
- old approval-trigger event handling on the chat primary path
- duplicate approval resume or confirmation flows used only to support the old runtime

### Remove From Frontend

- stores that derive runtime state from multiple protocol families
- rendering paths that treat serialized JSON blobs as fallback process UI
- empty-state transitions that can mask a live in-flight conversation

### Remove From Backend

- SSE emission paths whose only purpose is legacy phase or block compatibility
- approval bridge logic that bypasses the new chain pause/resume model
- compatibility-only DTOs and event types once migration is complete

## Testing Strategy

### Backend

- chain lifecycle event ordering tests
- approval pause/resume tests on the same chain
- replan event generation tests
- legacy event non-emission tests on the new primary path

### Frontend

- chain store state transition tests
- assistant message rendering tests for all node kinds
- recommended prompt new-conversation race regression tests
- approval inline action and resume tests
- serialized JSON regression tests to prevent raw payload dumping

### Observability

- callback invocation tests
- Prometheus metric emission tests with expected labels
- trace/span propagation tests across approval and replan boundaries

## Migration Strategy

This work should proceed in a deletion-first sequence:

1. inventory and mark all legacy runtime entry points and approval paths
2. detach legacy runtime structures from the chat primary path
3. delete obsolete frontend and backend protocol dependencies needed only by the old runtime
4. implement the new `thoughtChain` runtime contract
5. wire the unified approval decision API into the same chain lifecycle
6. add callback-based Prometheus observability
7. finalize test coverage and remove any temporary migration shims

Temporary compatibility should be minimized and explicitly deleted before completion. Long-lived dual-write or dual-render behavior is not acceptable for this redesign.

## Risks And Mitigations

### Risk: Deletion Breaks Hidden Dependencies

Mitigation:

- inventory legacy protocol usage before removal
- remove old primary-path wiring in small verified slices
- add tests asserting that new runtime behavior remains intact

### Risk: Approval Flow Regresses During Cutover

Mitigation:

- make approval a first-class node before deleting old resume logic completely
- keep approval lifecycle tests mandatory for every migration slice

### Risk: Frontend Replay And Live Stream Diverge

Mitigation:

- use the same `thoughtChain` node model for live updates and replay restoration
- keep one rendering path for both live and restored assistant responses

## Success Criteria

- the assistant UI renders one coherent `thoughtChain` and no longer depends on legacy runtime views
- approval-required tools pause the active chain and wait inline instead of exiting
- recommended prompts in a fresh conversation do not show a false unavailable state
- Prometheus receives consistent chain and node metrics through callback hooks
- the legacy runtime protocol family is removed from the main chat path and protected against accidental reintroduction
