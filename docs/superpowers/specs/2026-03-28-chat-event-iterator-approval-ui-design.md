# Chat Event Iterator Unification & Approval UI Enrichment Design

- Date: 2026-03-28
- Scope:
  - Backend: `internal/service/ai/logic`
  - Frontend: `web/src/components/AI`, `web/src/api/modules/ai.ts`
- Goal:
  - Unify duplicated backend chat/approval iterator consumption loops.
  - Upgrade approval UI so users can review the concrete operation before approving.

## 1. Background

Current backend has repeated `for + iterator.Next() + stream recv` logic in two paths:

1. `Logic.Chat`
2. `ApprovalWorker.resumeApprovedTask`

These paths independently maintain interrupt handling, recoverable tool errors, stream receive failures, and projector flush/finalization logic. This creates drift risk.

On frontend, `tool_approval` currently stores only minimal info in runtime activity. The approval card mostly exposes approve/reject actions, while the operation preview payload is not surfaced in a user-readable way.

## 2. Goals and Non-Goals

### 2.1 Goals

1. Remove duplicated iterator consumption core from the two backend paths.
2. Keep existing path-specific orchestration semantics intact.
3. Render approval-required operation details in UI:
   - key parameter summary table
   - expandable raw JSON preview
4. Preserve SSE contract compatibility.

### 2.2 Non-Goals

1. No SSE protocol rename or schema breaking changes.
2. No redesign of approval business workflow.
3. No broad UI redesign outside approval-related components.

## 3. Options Considered

### Option A (Selected): Shared Event Iterator Processor + Per-Path Orchestration

- Extract one reusable iterator processor for core consumption.
- Keep per-path pre/post logic (run status updates, terminalization policy, emit behavior).

Pros:

- Highest de-duplication with controlled risk.
- Keeps business boundaries clear.
- Enables shared tests for iterator behavior.

Cons:

- Requires callback/policy interface design.

### Option B: Only Extract Small Loop Helpers

Pros:

- Minimal code movement.

Cons:

- Leaves substantial duplication.
- Drift risk remains.

### Option C: Full Execution Engine Unification

Pros:

- Maximum structural unification.

Cons:

- Overly invasive for current target.
- Large regression surface.

## 4. Selected Design

## 4.1 Backend Architecture

Introduce a shared processor (name tentative: `processAgentIterator`) for iterator traversal and stream chunk ingestion.

### Input Contract

- `iterator`: `*adk.AsyncIterator[*adk.AgentEvent]`
- `projector`: `*airuntime.StreamProjector`
- `emit`: `EventEmitter` (or no-op)
- callbacks/policies:
  - projected-event consume callback
  - flush callback
  - assistant/intent update callback
  - recoverable-failure callback
  - fatal-failure callback

### Output Contract

Return a unified result (`IteratorProcessResult`, tentative):

- `Interrupted bool`
- `HasToolErrors bool`
- `CircuitBroken bool`
- `SummaryText string`
- `AssistantSnapshot string`
- `FatalErr error`

This result is interpreted by each caller to perform path-specific finalize behavior.

### Responsibilities Inside Shared Processor

1. Main iterator loop (`iterator.Next()`).
2. `event.Err` classification:
   - recoverable interrupt
   - recoverable tool error
   - fatal error
3. Message stream sub-loop (`MessageStream.Recv()`).
4. Projection pipeline handling:
   - `projector.Consume`
   - `projector.FlushBuffer`
   - completion boundary checks
5. Summary accumulation and tool-failure tracking updates.

### Responsibilities Kept in Callers

1. `Logic.Chat`
   - live SSE emission semantics
   - terminal failure conversion via existing chat path policy
2. `ApprovalWorker.resumeApprovedTask`
   - run status transitions (`resuming`, retryable failure, terminal failure)
   - outbox/write-model side effects

## 4.2 Frontend Approval Experience

### Data Model Changes

Extend `AssistantReplyActivity` with approval presentation fields:

- `approvalPreview?: Record<string, unknown>`
- `approvalPreviewSummary?: Array<{ key: string; label: string; value: string }>`

In `applyToolApproval`, persist `payload.preview` and derive a summary list.

### Summary Extraction Rules

Prefer common fields if present:

- `cluster`
- `namespace`
- `resource` / `resourceType`
- `kind`
- `name`
- `action`
- `risk` / `riskLevel`

If absent, fallback to a flattened preview subset with bounded length/value formatting.

### UI Rendering Changes

`tool_approval` card must include:

1. Tool label and current approval state.
2. "Operation to approve" key parameter table (2-8 rows).
3. Risk/timeout hints.
4. Expandable raw JSON section ("View raw approval payload").

### Approval Action Behavior

Restore and harden approve/reject interaction flow:

1. `pending` state: approve/reject enabled.
2. `submitting` state: disable controls, show in-progress.
3. Success: move to terminal approval state and broadcast update event.
4. Conflict response: auto-refresh ticket via `getApproval`.
5. Refresh failure: move to `refresh-needed` readonly state.

### Backward Compatibility

If historical `tool_approval` lacks `preview`, render fallback text:

- "No structured preview available"

Actions remain available.

## 5. Data Flow

## 5.1 Backend Iterator Path (Unified Core)

1. Caller sets up shell/projector/policies.
2. Shared processor consumes iterator and emits/persists projected events via callbacks.
3. Shared processor returns `IteratorProcessResult`.
4. Caller finalizes run state based on path-specific rules.

## 5.2 Frontend Approval Path

1. SSE `tool_approval` received by `PlatformChatProvider`.
2. `applyToolApproval` stores preview + summary into runtime activity.
3. `AssistantReply`/`ToolReference` render approval card with details.
4. User chooses approve/reject after reviewing operation content.

## 6. Error Handling

## 6.1 Backend

1. Recoverable tool errors:
   - continue stream, mark tool error state
2. Interrupt events:
   - transition to waiting approval path
3. Fatal iterator/stream errors:
   - return `FatalErr` to caller
   - caller applies path-specific terminalization

## 6.2 Frontend

1. Submit approval network failures:
   - show actionable error and readonly fallback when needed
2. Conflict:
   - refresh canonical approval status
3. Missing preview:
   - degrade gracefully without blocking user action

## 7. Testing Plan

## 7.1 Backend

1. Shared processor unit tests:
   - normal flow
   - interrupt flow
   - recoverable tool error
   - stream recv failure
   - fatal iterator failure
2. Regression tests for:
   - `Logic.Chat`
   - `ApprovalWorker.resumeApprovedTask`
3. Assert event ordering and run state parity with current contract.

## 7.2 Frontend

1. `replyRuntime` tests for preview persistence + summary derivation.
2. `AssistantReply`/`ToolReference` tests:
   - key parameter rendering
   - raw JSON expand/collapse
   - approve/reject state transitions
   - conflict refresh + refresh-needed branch
3. Keep existing SSE/provider tests green.

## 8. Acceptance Criteria

1. Backend no longer maintains duplicated full iterator loops across paths.
2. Frontend approval card clearly shows operation details before user decision.
3. Legacy events without preview remain actionable.
4. Existing SSE event names and compatibility remain intact.

## 9. Implementation Notes

1. This change intentionally preserves existing route and API surfaces.
2. Follow-up refactors can extend the shared processor to additional iterator consumers if needed.
