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

### 5.1 Types

Update `web/src/components/AI/types.ts`:

#### 5.1.1 Extend `AssistantReplySegment`

```typescript
export interface AssistantReplySegment {
  type: 'text' | 'tool_ref' | 'agent_ref';  // NEW: add 'agent_ref'
  text?: string;
  callId?: string;
  agentId?: string;  // NEW: for agent_ref
}
```

#### 5.1.2 Add Top-Level `segments` to `AssistantReplyRuntime`

**Current state:** Segments only exist within `AssistantReplyPlanStep.segments`. Non-plan runtime has no segment infrastructure.

**Required change:**

```typescript
export interface AssistantReplyRuntime {
  phase?: AssistantReplyPhase;
  phaseLabel?: string;
  activities: AssistantReplyActivity[];
  plan?: AssistantReplyPlan;
  segments?: AssistantReplySegment[];  // NEW: top-level for non-plan narrative
  summary?: AssistantReplySummary;
  status?: AssistantReplyRuntimeStatus;
  pendingRun?: PendingRunMetadata;
  todos?: AssistantReplyTodo[];
  _executorBlocks?: SlimExecutorBlock[];
}
```

**Migration rule:**
- If `runtime.plan` exists → use `plan.steps[activeStepIndex].segments` (existing path)
- If `runtime.plan` is undefined → use `runtime.segments` (new path)
- Legacy runtimes without `segments` → fallback to `content + activities` rendering

### 5.2 Runtime Builders

Update `web/src/components/AI/replyRuntime.ts`:

#### 5.2.1 `applyDelta` Refactoring

**Current behavior:** Only updates `content` string on `XChatMessage`.

**Required change:** For non-plan mode, also append/merge text segment.

```typescript
export function applyDelta(
  message: Pick<XChatMessage, 'content' | 'runtime'>,
  payload: { content: string },
): Pick<XChatMessage, 'content' | 'runtime'> {
  const chunk = payload.content || '';
  const current = message.content || '';
  const nextContent = !current || current === PLACEHOLDER_CONTENT ? chunk : `${current}${chunk}`;

  // NEW: Append text segment for non-plan runtime
  let nextRuntime = message.runtime;
  if (nextRuntime && !nextRuntime.plan) {
    const segments = [...(nextRuntime.segments || [])];
    const lastSegment = segments[segments.length - 1];
    if (lastSegment?.type === 'text') {
      // Merge with previous text segment (immutably)
      segments[segments.length - 1] = {
        ...lastSegment,
        text: `${lastSegment.text || ''}${chunk}`,
      };
    } else {
      segments.push({ type: 'text', text: chunk });
    }
    nextRuntime = { ...nextRuntime, segments };
  }

  return { content: nextContent, runtime: nextRuntime };
}
```

#### 5.2.2 `applyToolCall` Update

**Current behavior:** Appends `tool_ref` to active step only.

**Required change:** Also append to top-level `segments` for non-plan.

```typescript
// After existing activity upsert logic:
if (!nextRuntime.plan && payload.call_id) {
  const segments = [...(nextRuntime.segments || [])];
  segments.push({ type: 'tool_ref', callId: payload.call_id });
  nextRuntime = { ...nextRuntime, segments };
}
```

#### 5.2.3 `applyAgentHandoff` Update

**Current behavior:** Creates activity only, no segment.

**Required change:** Append `agent_ref` segment.

```typescript
export function applyAgentHandoff(
  runtime: AssistantReplyRuntime,
  payload: { from: string; to: string; intent: string },
): AssistantReplyRuntime {
  let nextRuntime = upsertActivity(
    runtime,
    {
      id: `handoff:${payload.to}`,
      kind: 'agent_handoff',
      label: payload.to,
      detail: payload.intent,
      status: 'done',
    },
    (item) => item.id === `handoff:${payload.to}`,
  );

  // NEW: Append agent_ref segment for non-plan
  if (!nextRuntime.plan) {
    const segments = [...(nextRuntime.segments || [])];
    segments.push({ type: 'agent_ref', agentId: payload.to });
    nextRuntime = { ...nextRuntime, segments };
  }

  return {
    ...nextRuntime,
    phase: 'executing',
    phaseLabel: `${payload.to} 开始处理`,
  };
}
```

#### 5.2.4 `applyToolApproval` / `applyToolResult`

No segment changes needed — these only mutate activity state, segment position is immutable after insertion.

### 5.3 History Projection

Update `web/src/components/AI/historyProjection.ts`:

#### 5.3.1 Add `agent_handoff` Handling at Block Level (Not Executor Item Level)

```typescript
// source: AIRunProjectionBlock (type='agent_handoff')
if (block.type === 'agent_handoff' && block.agent) {
  activities.push({
    id: `handoff:${block.id}`,
    kind: 'agent_handoff',
    label: block.agent,
    detail: String(block.data?.intent || ''),
    status: 'done',
    stepIndex,
  });
  segments.push({ type: 'agent_ref', agentId: block.agent });
}
```

#### 5.3.2 Non-Plan History Hydration

For messages without plan, build `runtime.segments` from projection blocks:

```typescript
// In hydrateAssistantHistoryFromProjection or projectionToLazyRuntime
if (!steps.length) {
  // Non-plan: build segments from projection blocks in order
  const segments: AssistantReplySegment[] = [];
  for (const block of projection.blocks) {
    if (block.type === 'agent_handoff' && block.agent) {
      segments.push({ type: 'agent_ref', agentId: block.agent });
      continue;
    }
    if (block.type === 'executor') {
      for (const item of block.items || []) {
        if (item.type === 'content' && item.content_id) {
          // Load and append text segment
        }
        if (item.type === 'tool_call' && item.tool_call_id) {
          segments.push({ type: 'tool_ref', callId: item.tool_call_id });
        }
      }
    }
  }
  runtime.segments = segments;
}
```

## 6. Rendering Design

### 6.1 Unified Segment Renderer

**Current state:**
- `StepContentRenderer` handles segment rendering for plan steps only
- Non-plan uses `standaloneActivities` array outside narrative flow
- Two separate rendering paths exist

**Target state:** Single `renderSegmentFlow(segments, activities, context)` utility.

```typescript
// web/src/components/AI/renderSegmentFlow.ts (new file or in AssistantReply.tsx)

interface RenderContext {
  isStreaming: boolean;
  styles: Record<string, string>;
  onToolClick?: (activity: AssistantReplyActivity) => void;
}

function renderSegmentFlow(
  segments: AssistantReplySegment[],
  activities: AssistantReplyActivity[],
  context: RenderContext,
): React.ReactNode {
  const activityMap = new Map(activities.map(a => [a.id, a]));
  const elements: React.ReactNode[] = [];

  for (const segment of segments) {
    if (segment.type === 'text') {
      elements.push(renderTextSegment(segment, context));
    } else if (segment.type === 'tool_ref') {
      const activity = activityMap.get(segment.callId || '');
      elements.push(renderToolToken(segment, activity, context));
    } else if (segment.type === 'agent_ref') {
      const activity = activityMap.get(`handoff:${segment.agentId}`);
      elements.push(renderAgentToken(segment, activity, context));
    }
  }

  return wrapWithFlowLayout(elements, context);
}
```

**Usage in AssistantReply.tsx:**
- Plan active step: `renderSegmentFlow(step.segments, stepActivities, context)`
- Plan completed step: `renderSegmentFlow(cachedContent.segments, cachedContent.activities, context)`
- Non-plan: `renderSegmentFlow(runtime.segments, runtime.activities, context)`

### 6.2 Inline Token Style Rules

Inline tokens:

- Tool active: `[tool_name …]`
- Tool success: `[tool_name ✓]`
- Tool error: `[tool_name ✕]`
- Agent: `[Agent: <name>]` (from real event payload)

Layout behavior:

- Default inline with text flow (`inline-block`).
- Auto fallback to block-safe layout around markdown block structures to avoid broken typography.

### 6.3 Approval Card Behavior

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

### 7.1 Stream Ingestion Seam (Authoritative)

Authoritative ingestion layer: `web/src/components/AI/providers/PlatformChatProvider.ts` handler map.

Required handler-to-reducer mapping (must remain explicit in implementation plan):

- `onDelta` -> `applyDelta` (non-plan) or `applyStepDelta` (plan active step)
- `onToolCall` -> `applyToolCall`
- `onToolApproval` -> `applyToolApproval`
- `onToolResult` -> `applyToolResult`
- `onAgentHandoff` -> `applyAgentHandoff`
- `onRunState` -> `applyRunState` (and optional agent marker injection, see 7.2)

Ordering and replay rules at ingestion seam:

1. Preserve incoming SSE order for segment append operations.
2. Mutating events (`tool_result`, approval state transitions) must update existing activity records without changing segment position.
3. Reconnect/replay events must be deduped by reference IDs (`call_id`, `handoff-id`, or deterministic key).
4. Out-of-order `tool_result` (before visible `tool_call`) must still upsert activity; segment insertion follows first observed call/handoff marker and must not duplicate later.

### 7.2 Agent Marker Construction Rule

For live stream:

- Primary source: `onAgentHandoff` payload (`from`, `to`, `intent`) -> append `agent_ref`.
- Secondary source: `onRunState.agent` when present and meaningful transition detected -> append `agent_ref` only if not duplicate of current tail marker.
  - Dedupe key for live-only marker: `runstate-agent:<agentName>`
  - This requires lightweight provider-side injection logic in `PlatformChatProvider` before/after `applyRunState`.

For history hydration:

- Authoritative persisted source: `AIRunProjection.blocks` with `type='agent_handoff'`.
- Reconstruct `agent_ref` from `agent_handoff` blocks in stored order.
- `run_state.agent` is treated as **live-only enrichment** unless/until backend projection schema explicitly persists equivalent state transitions.
- Never synthesize fixed roles not present in stored events.

## 8. Error Handling and Fallbacks

- Missing activity for a segment ref: render lightweight placeholder token, do not break message rendering.
- Legacy runtime without segments: keep backward-compatible fallback path (`content + activities`).
- Reconnect duplicate events: dedupe via reference IDs and append guards.

### 8.1 Render Contract (Testable Boundary)

`renderSegmentFlow(segments, activities, context)` contract:

| Input Pattern | Expected Rendering |
|---|---|
| `text("...") + tool_ref(call-1) + text("...")` with tool done | continuous narrative with inline `[tool_name ✓]` between text fragments |
| same with tool error | inline `[tool_name ✕]` |
| tool approval waiting state | inline approval token + expanded detail card with actions |
| tool approval terminal state (`approved/rejected/expired`) | auto-collapsed inline token; detail hidden by default but expandable |
| `agent_ref(agent-x)` between texts | inline `[Agent: agent-x]` at exact segment position |
| markdown block text segment (list/code heading) adjacent to tool token | block-safe rendering for markdown; inline token shown before/after block without breaking layout |
| missing activity for `tool_ref`/`agent_ref` | render placeholder token (`[tool missing]` / `[agent missing]`), no crash |

Renderer boundary rules:

1. Segment order is authoritative for visible token placement.
2. Activity table is authoritative for mutable state/status/detail.
3. Renderer is side-effect free (no mutation of runtime state).
4. Collapsed/expanded approval visibility is UI state layered on top of segment+activity inputs.

## 9. Testing Plan

### 9.1 Unit Tests

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

### 9.2 Regression Coverage

- Existing approval submit/conflict/retry behavior remains unchanged.
- Existing reconnect behavior for waiting approval and resume statuses remains unchanged.

## 10. Acceptance Criteria

1. Tool/approval markers are no longer globally stacked above正文; they appear in narrative order.
2. Approval detail is visible while pending and auto-collapses after terminal approval state.
3. Tool status is visible inline with success/failure markers (`✓` / `✕`).
4. Agent flow is shown only from real runtime events.
5. Live stream and history projection present the same ordering semantics.
   - For agent markers, parity guarantee is scoped to persisted `agent_handoff` events.
   - Live-only `run_state.agent` markers are best-effort runtime hints and are not required to appear in hydrated history.

## 11. Implementation Scope (Next Step)

### 11.1 Touched Files

| File | Change Type | Key Changes |
|------|-------------|-------------|
| `web/src/components/AI/types.ts` | Extend | Add `agent_ref` to segment union, add `segments` to runtime |
| `web/src/components/AI/replyRuntime.ts` | Modify | Refactor `applyDelta`, `applyToolCall`, `applyAgentHandoff` |
| `web/src/components/AI/historyProjection.ts` | Modify | Add `agent_handoff` handler, non-plan segment building |
| `web/src/components/AI/AssistantReply.tsx` | Modify | Create unified `renderSegmentFlow`, handle non-plan segments |
| `web/src/components/AI/ToolReference.tsx` | Modify | Add placeholder fallback, approval collapse logic |
| `web/src/components/AI/ToolResultCard.tsx` | Minor | Collapse behavior if needed |
| `web/src/components/AI/providers/PlatformChatProvider.ts` | Modify | Optional live `run_state.agent` marker injection + dedupe |

### 11.2 Implementation Phases

**Phase 1: Data Model (Low Risk)**
- Add `agent_ref` to `AssistantReplySegment`
- Add `segments?: AssistantReplySegment[]` to `AssistantReplyRuntime`
- Write unit tests for new types

**Phase 2: Runtime Builders (Medium Risk)**
- Refactor `applyDelta` with segment appending
- Update `applyToolCall` for non-plan segment
- Update `applyAgentHandoff` for segment
- Unit tests for segment order preservation

**Phase 3: Rendering (Medium Risk)**
- Extract `renderSegmentFlow` utility
- Update `AssistantReplyContent` for non-plan segments
- Add placeholder tokens for missing activities
- Visual regression tests

**Phase 4: History Projection (Low Risk)**
- Add `agent_handoff` handler in `loadStepContent`
- Build non-plan segments from projection
- Unit tests for history parity

**Phase 5: Approval Collapse (Low Risk)**
- Update `ToolReference` for auto-collapse
- Ensure expanded/collapsed state management
- Unit tests for approval state transitions

### 11.3 Risk Mitigation

1. **Legacy compatibility:** Always check for `segments` existence before using
2. **Reconnect safety:** Dedupe segments by ID during replay
3. **Plan vs non-plan:** Clear conditional paths with explicit `if (runtime.plan)` checks
