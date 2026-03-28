# Tool Inline Narrative + Approval Fold + Agent Flow Design

Date: 2026-03-28  
Status: Draft Approved by User (conversation-level)

## 1. Context

Current chat runtime UI shows many tool-call/approval cards above or outside the narrative text, which breaks reading continuity.  
Target UX is narrative-first:

- Assistant text and tool references appear in one continuous body flow.
- Approval details are visible while waiting, then auto-collapse after approval/rejection.
- Tool completion state is visible inline (`✓` / `✕`).
- Agent flow uses real runtime events only (no assumed planner/executor/replanner roles).

## 2. Product Decisions (Locked)

From user confirmations:

1. Inline behavior scope: **global** (both step-based and non-plan assistant replies).
2. Approval post-submit UI: **collapsed one-line token** (`[tool_name ✓/✕]`), expandable on demand.
3. Agent-flow source: **existing runtime events only** (`agent_handoff` / `run_state.agent` if available in stream).
4. Tool inline status: **show success/failure inline** (`[tool_name ✓]` / `[tool_name ✕]`).

## 3. Goals and Non-Goals

### Goals

- Keep tool/approval/agent markers in narrative order.
- Ensure consistent rendering between live stream and history projection.
- Preserve approval action UX while waiting.
- Minimize regressions in current runtime state machine and reconnect logic.

### Non-Goals

- No backend protocol redesign in this change.
- No synthetic/guessed subagent timeline.
- No migration of old stored messages; fallback rendering remains for legacy payloads.

## 4. Architecture

### 4.1 Unified Narrative Segment Flow

Introduce/standardize a single ordered segment flow for assistant rendering:

- `text`
- `tool_ref`
- `agent_ref` (new)

Notes:

- Approval stays represented by activity kind (`tool_approval`) referenced by the same `tool_ref`.
- Plan and non-plan rendering use the same segment interpretation pipeline.

### 4.2 Activity Table as Source of Truth

Keep `runtime.activities` for mutable state:

- tool status (`active/done/error`)
- approval state (`waiting-approval/submitting/approved/rejected/...`)
- tool args/raw content/approval preview
- agent handoff metadata

Segments store references only (e.g., `callId`, `agentId`) and never duplicate mutable details.

## 5. Data Model Changes

## 5.1 Types

Update `web/src/components/AI/types.ts`:

- Extend `AssistantReplySegment`:
  - add `type: 'agent_ref'`
  - add `agentId?: string`
- Ensure `AssistantReplyRuntime` has narrative-level `segments?: AssistantReplySegment[]` for non-plan flow.

### 5.2 Runtime Builders

Update `web/src/components/AI/replyRuntime.ts`:

- `applyDelta`: append/merge `text` segment for non-plan mode.
- `applyToolCall`: append `tool_ref(call_id)` for current narrative flow.
- `applyToolApproval`/`applyToolResult`: update referenced activity only (segment position unchanged).
- `applyAgentHandoff`: append `agent_ref` segment and create/update activity.

### 5.3 History Projection

Update `web/src/components/AI/historyProjection.ts` to emit equivalent segment flow for hydrated history so historical display equals live display.

## 6. Rendering Design

## 6.1 One Renderer for All Flows

In `web/src/components/AI/AssistantReply.tsx`:

- Build a single `renderSegmentFlow(segments, activities, context)` path.
- Reuse for:
  - active step rendering
  - completed-step lazy content rendering
  - non-plan standalone assistant rendering

## 6.2 Inline Token Style Rules

Inline tokens:

- Tool active: `[tool_name …]`
- Tool success: `[tool_name ✓]`
- Tool error: `[tool_name ✕]`
- Agent: `[Agent: <name>]` (from real event payload)

Layout behavior:

- Default inline with text flow (`inline-block`).
- Auto fallback to block-safe layout around markdown block structures to avoid broken typography.

## 6.3 Approval Card Behavior

`ToolReference` / `ToolResultCard` behavior:

- `waiting-approval/submitting/refresh-needed`: expanded inline detail card with actions.
- `approved/rejected/expired`: auto-collapse to one-line token with state marker.
- Collapsed token remains clickable to expand details.

## 7. Event Mapping Rules

Ordered mapping:

1. `delta` -> `text` segment
2. `tool_call` -> `tool_ref`
3. `tool_approval` -> mutate activity state to approval
4. `tool_result` -> mutate tool status/content
5. `agent_handoff` / `run_state.agent` -> `agent_ref`

Idempotency:

- Upsert activities by stable IDs (`call_id`, `handoff:<agent>` or equivalent).
- Prevent duplicate segment insertions for replay/reconnect where possible.

## 8. Error Handling and Fallbacks

- Missing activity for a segment ref: render lightweight placeholder token, do not break message rendering.
- Legacy runtime without segments: keep backward-compatible fallback path (`content + activities`).
- Reconnect duplicate events: dedupe via reference IDs and append guards.

## 9. Testing Plan

## 9.1 Unit Tests

- `replyRuntime.test.ts`
  - non-plan `delta -> tool_call -> tool_result` yields inline order.
  - tool error maps to `[tool ✕]`.
  - approval transitions to auto-collapsed state after final decision.
  - agent handoff adds `agent_ref`.

- `AssistantReply.test.tsx`
  - narrative inline rendering (`text + [tool] + text`).
  - approval waiting shows expanded details.
  - approval approved/rejected auto-collapses token.
  - collapsed token can reopen details.

- `historyProjection.test.ts`
  - hydrated history segment order matches live runtime behavior.

## 9.2 Regression Coverage

- Existing approval submit/conflict/retry behavior remains unchanged.
- Existing reconnect behavior for waiting approval and resume statuses remains unchanged.

## 10. Acceptance Criteria

1. Tool/approval markers are no longer globally stacked above正文; they appear in narrative order.
2. Approval detail is visible while pending and auto-collapses after terminal approval state.
3. Tool status is visible inline with success/failure markers (`✓` / `✕`).
4. Agent flow is shown only from real runtime events.
5. Live stream and history projection present the same ordering semantics.

## 11. Implementation Scope (Next Step)

Expected touched files:

- `web/src/components/AI/types.ts`
- `web/src/components/AI/replyRuntime.ts`
- `web/src/components/AI/historyProjection.ts`
- `web/src/components/AI/AssistantReply.tsx`
- `web/src/components/AI/ToolReference.tsx`
- `web/src/components/AI/ToolResultCard.tsx` (if needed for collapse behavior)
- corresponding test files in `web/src/components/AI/__tests__/` and runtime/history tests.
