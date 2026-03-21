# AI Session Interaction Repair Design

**Date**: 2026-03-21
**Status**: Approved

## Background

The recent step lazy-loading refactor improved historical session hydration cost, but it introduced three regressions in the AI assistant session surface:

1. Active-session completed steps can be collapsed but cannot be expanded again.
2. Historical sessions no longer show the step list consistently.
3. Conversation follow mode does not recover when the user sends a new message while scrolled away from the bottom.

This change must preserve the current UI appearance. Only interaction logic and state wiring are in scope.

## Goals

1. Restore expand/collapse behavior for completed steps in active sessions.
2. Ensure historical assistant replies always show summary plus step titles when projection data contains steps.
3. Make sending a new user message force the viewport back to the bottom and re-enter follow mode for the new run.
4. Keep failures local to the affected step or message instead of hiding the whole reply.

## Non-Goals

1. No visual redesign of the AI surface.
2. No backend contract changes.
3. No broad runtime architecture rewrite.

## Approach Options

### Option 1: Minimal behavior repair in existing components

Repair the broken state boundaries inside `historyProjection.ts`, `AssistantReply.tsx`, and `CopilotSurface.tsx` while preserving the current component structure.

Pros:

- Lowest regression risk
- Keeps current UI unchanged
- Directly targets the three observed failures

Cons:

- Existing stateful logic remains somewhat dense

### Option 2: Extract shared step disclosure logic into a hook

Move step expansion, lazy loading, and retry state into a dedicated hook shared across active and historical replies.

Pros:

- Cleaner long-term ownership of step state
- Easier future extension

Cons:

- More refactoring than this bugfix requires
- Higher regression surface for a behavior-only repair

### Option 3: Rewrite scroll and reply interactions as explicit state machines

Formalize both follow-mode transitions and reply-step disclosure into separate state machines.

Pros:

- Strongest long-term clarity

Cons:

- Too invasive for the current issue
- Conflicts with the requirement to keep the current UI and interaction shell stable

## Recommended Approach

Use Option 1.

The current failures are caused by mismatched state ownership and incorrect runtime assumptions, not by a fundamentally wrong component split. The smallest reliable fix is to repair the existing interaction boundaries and add regression tests around them.

## Design

### 1. Step interaction rules

Active sessions:

- The active step remains expanded.
- Completed steps remain individually collapsible.
- Collapsing and re-expanding a completed step must work repeatedly.

Historical sessions:

- The assistant reply initially renders summary content plus step titles only.
- Expanding a historical step triggers lazy loading for that specific step body.
- Step load failure is local to that step and exposes retry affordance without hiding summary or sibling steps.

### 2. Historical projection mapping

The current lazy runtime shape is too weak because it assumes the visible completed-step order matches executor block order. That assumption breaks when projection data contains plan/replan transitions.

Repair strategy:

- Keep projection hydration lightweight.
- Extend each historical step with a stable source index or equivalent block lookup metadata created during `projectionToLazyRuntime()` when that mapping can be derived.
- Make step content loading use this stable mapping instead of relying on the completed-step array position at click time.
- Preserve backward compatibility for older historical projections that do not provide enough information for a perfect mapping:
  - first try the stable mapping metadata
  - if absent, fall back to the safest deterministic lookup available in the projection adapter
  - if no safe lookup can be derived, still render the step title and keep the disclosure local-failure path instead of hiding the whole step list
- If a step title exists but no block can be resolved, still render the step title and fail only the local disclosure panel.

This preserves visibility of historical steps even when content recovery is partial.

### 3. Collapse state ownership

`AssistantReply` currently treats the completed-step collapse as fully controlled, but the change handler only reacts to newly opened keys. That allows the control state to drift and causes the "collapsed once, cannot expand again" failure mode.

Repair strategy:

- Keep the collapse component controlled.
- On every `onChange`, rebuild the expanded-key map from the latest key set, covering both open and close transitions.
- Trigger lazy loading only for keys newly entering the expanded set.
- Preserve loaded content and successful results in message-local cache so re-expansion is immediate.
- Scope this cache to `AssistantReply` instance state rather than a global store:
  - cache survives repeated expand/collapse while the reply stays mounted
  - cache is released when the reply unmounts or the session detail view is replaced
  - global history/projection caches remain responsible only for API-level projection/content fetch reuse

### 4. Follow-mode recovery after send

The current follow logic correctly detaches when the user scrolls upward, but it relies on resize-driven bottom alignment to continue following. Once detached, sending a new message does not explicitly restore follow mode, so the next run stays detached.

Repair strategy:

- Keep the existing `following` and `detached` semantics.
- Preserve manual scroll detachment and bottom-threshold recovery.
- Add an explicit send-time transition:
  - when the user submits a new message, force `followStateRef.current = 'following'`
  - schedule the send-time bottom alignment only after the new user turn has been committed to the DOM, so the scroll height reflects the inserted message
  - prefer a post-commit mechanism such as `requestAnimationFrame`, `useLayoutEffect`, or an equivalent render-synchronized hook rather than a same-tick synchronous scroll
  - let subsequent streamed updates continue in follow mode until the user manually detaches again
- Keep initial-open scroll behavior separate from send-time recovery so the two triggers remain debuggable.
- The implementation must treat "send reset" and "stream follow" as separate phases:
  - send reset guarantees one post-commit jump to the bottom
  - stream follow handles subsequent height changes for the same run without racing the initial insertion

This matches the intended UX: sending a message means "return me to the live conversation now."

## Components Affected

- `web/src/components/AI/historyProjection.ts`
  - Strengthen historical step metadata for stable lazy content lookup.
- `web/src/components/AI/AssistantReply.tsx`
  - Repair completed-step controlled expansion state and local step failure isolation.
- `web/src/components/AI/CopilotSurface.tsx`
  - Restore send-time bottom-follow transition without changing visible UI.
- `web/src/components/AI/__tests__/AssistantReply.test.tsx`
  - Add regression coverage for active/historical step behavior.
- `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
  - Add regression coverage for send-time follow recovery.

## Error Handling

1. Missing historical step content must produce a local step error state, not remove the step list.
2. Projection or content cache misses remain recoverable by retrying the affected step or rehydrating the message.
3. Scroll-follow restoration must ignore programmatic scroll events to avoid self-triggered detach transitions.

## Testing

Add or update tests to lock the repaired behavior:

1. Active completed steps can collapse and re-expand.
2. Historical projection-backed replies show step titles before content loads.
3. Historical step expansion resolves content through stable mapping rather than rendered-list position.
4. Historical step load failure stays local and supports retry.
5. Opening an existing conversation still performs one initial bottom alignment.
6. User upward scroll still detaches follow mode.
7. Sending a new message while detached forces bottom scroll and restores follow mode.
8. Streamed assistant updates keep following after send until the user manually detaches again.

Testing split:

- Component tests in `CopilotSurface.test.tsx` should continue to verify follow-state transitions, scheduling decisions, and `scrollTo` invocation contracts with mocked layout primitives.
- Browser-level E2E coverage should verify the real viewport behavior for detach, send-time recovery, and continued follow during streaming, because JSDOM cannot faithfully simulate actual layout and scroll physics.

## Acceptance Criteria

1. In an active session, completed steps can be expanded after being collapsed.
2. In a historical session, steps are visible whenever projection data includes them.
3. In a historical session, step content loads on demand and does not require eager hydration.
4. When the user sends a new message from a non-bottom scroll position, the surface jumps to the bottom and follows the new run.
5. No visual style changes are introduced.
