# Tool Inline Narrative + Approval Fold + Agent Flow Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make tool/approval/agent markers render inline with assistant narrative (global scope), auto-collapse approval details after decision, and keep live/history ordering consistent.

**Architecture:** Keep `activities` as mutable state source, and make `segments` the ordered narrative source for both plan and non-plan rendering. Update runtime reducers to append stable segment refs while tool/approval state mutates via activity upserts. Use one renderer contract in `AssistantReply` so live stream and hydrated history share the same rendering semantics.

**Tech Stack:** React, TypeScript, Ant Design/X, Vitest, existing AI chat runtime (`replyRuntime`, `historyProjection`, `PlatformChatProvider`).

---

## Chunk 1: Runtime Model and Stream Ingestion

### Task 1: Extend types for unified segment flow

**Files:**
- Modify: `web/src/components/AI/types.ts`
- Test: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: Write failing type-driven tests for `agent_ref` and top-level runtime segments**

```ts
// add cases in replyRuntime.test.ts that require:
// - AssistantReplySegment supports type='agent_ref' with agentId
// - AssistantReplyRuntime supports runtime.segments in non-plan mode
```

- [ ] **Step 2: Run targeted test to confirm failure**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts -t "agent_ref|segments"`  
Expected: FAIL with missing type fields or runtime contract mismatch.

- [ ] **Step 3: Implement minimal type updates**

```ts
// types.ts
export interface AssistantReplySegment {
  type: 'text' | 'tool_ref' | 'agent_ref';
  text?: string;
  callId?: string;
  agentId?: string;
}

export interface AssistantReplyRuntime {
  // existing fields...
  segments?: AssistantReplySegment[];
}
```

- [ ] **Step 4: Re-run test and ensure pass**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts -t "agent_ref|segments"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/replyRuntime.test.ts
git commit -m "feat(ai-ui): extend runtime segments for inline tool and agent refs"
```

### Task 2: Update reducers to append/merge non-plan segments safely

**Files:**
- Modify: `web/src/components/AI/replyRuntime.ts`
- Test: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: Write failing reducer tests (TDD via @test-driven-development)**

```ts
// add tests:
// 1) applyDelta merges consecutive text segments immutably in non-plan mode
// 2) applyToolCall appends tool_ref in non-plan mode
// 3) applyAgentHandoff appends agent_ref in non-plan mode
// 4) applyToolResult/applyToolApproval mutate activity without changing segment order
```

- [ ] **Step 2: Run reducer tests and confirm failure**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts -t "non-plan|segment|agent_ref|tool_ref"`  
Expected: FAIL for missing segment updates and ordering assertions.

- [ ] **Step 3: Implement minimal reducer changes**

```ts
// replyRuntime.ts
// - applyDelta: append/merge top-level text segment in non-plan mode (no in-place mutation)
// - applyToolCall: append top-level tool_ref when !runtime.plan
// - applyAgentHandoff: append top-level agent_ref when !runtime.plan
// - keep applyToolApproval/applyToolResult as activity mutations only
```

- [ ] **Step 4: Re-run tests**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts`  
Expected: PASS for new and existing reducer tests.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/replyRuntime.ts web/src/components/AI/replyRuntime.test.ts
git commit -m "feat(ai-ui): build unified non-plan narrative segments in reducers"
```

### Task 3: Align stream ingestion with live agent marker rules

**Files:**
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: Add failing tests for run-state agent marker injection and dedupe**

```ts
// tests:
// - when onRunState includes agent transitions, emit one agent_ref marker
// - duplicate run_state.agent does not append duplicate marker
// - existing reconnect and approval flows remain unchanged
```

- [ ] **Step 2: Run provider tests and verify failure**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts -t "run_state.agent|agent_ref|dedupe"`  
Expected: FAIL with missing runtime updates.

- [ ] **Step 3: Implement provider mapping**

```ts
// PlatformChatProvider.ts
// onRunState:
// - keep reconnectController handling
// - keep applyRunState
// - if payload.agent meaningful and new, append live-only agent_ref marker (dedupe key: runstate-agent:<agent>)
```

- [ ] **Step 4: Run provider suite**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts`  
Expected: PASS including reconnect/approval regressions.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat(ai-ui): inject and dedupe live run_state agent markers"
```

## Chunk 2: Unified Rendering and Approval Collapse

### Task 4: Introduce unified segment renderer contract

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Write failing render tests**

```tsx
// test cases:
// - non-plan message renders text + [tool] + text in order from runtime.segments
// - agent_ref renders inline [Agent: xxx] at exact segment position
// - missing activity renders placeholder token without crash
```

- [ ] **Step 2: Run tests and confirm fail**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx -t "inline|agent_ref|placeholder"`  
Expected: FAIL due to current split render paths.

- [ ] **Step 3: Implement unified flow renderer**

```tsx
// AssistantReply.tsx
// - extract renderSegmentFlow(segments, activities, ctx)
// - use for active/completed step and non-plan runtime.segments
// - retain legacy fallback: content + activities when segments absent
```

- [ ] **Step 4: Re-run tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx`  
Expected: PASS for new narrative-order assertions and existing markdown behaviors.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai-ui): unify segment rendering across plan and non-plan replies"
```

### Task 5: Implement approval auto-collapse and inline status tokens

**Files:**
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/components/AI/ToolResultCard.tsx` (only if display state helper needed)
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Add failing UI behavior tests**

```tsx
// cases:
// - waiting approval shows expanded details/actions
// - approved/rejected/expired auto-collapses to one-line token
// - collapsed token click reopens details
// - tool done/error token shows ✓/✕
```

- [ ] **Step 2: Run tests to verify failure**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx -t "approval|collapsed|✓|✕"`  
Expected: FAIL for missing collapse transitions.

- [ ] **Step 3: Implement minimal UI state transitions**

```tsx
// ToolReference.tsx
// - keep waiting/submitting/refresh-needed expanded by default
// - auto-close panel when state becomes approved/rejected/expired
// - preserve manual reopen on click
// - ensure inline token text format matches acceptance criteria
```

- [ ] **Step 4: Re-run tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx`  
Expected: PASS for approval/token behavior.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/ToolReference.tsx web/src/components/AI/ToolResultCard.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai-ui): auto-collapse approval details and normalize inline status tokens"
```

## Chunk 3: History Projection Parity and Final Verification

### Task 6: Rebuild history segments with persisted `agent_handoff` parity

**Files:**
- Modify: `web/src/components/AI/historyProjection.ts`
- Test: `web/src/components/AI/historyProjection.test.ts`

- [ ] **Step 1: Add failing history tests**

```ts
// cases:
// - non-plan projection builds runtime.segments from block order
// - executor content/tool_call become text/tool_ref
// - persisted agent_handoff block becomes agent_ref
// - run_state.agent is not required in hydration parity assertions
```

- [ ] **Step 2: Run tests to confirm fail**

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts -t "segments|agent_handoff|non-plan"`  
Expected: FAIL with missing segment reconstruction.

- [ ] **Step 3: Implement projection mapping**

```ts
// historyProjection.ts
// - in non-plan hydrate path, iterate projection.blocks in order
// - handle block.type === 'agent_handoff' via block.agent
// - handle executor block items for text/tool refs
// - keep current lazy-loading and fallback behavior
```

- [ ] **Step 4: Re-run history tests**

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/historyProjection.ts web/src/components/AI/historyProjection.test.ts
git commit -m "feat(ai-ui): align history hydration with inline segment narrative model"
```

### Task 7: Full regression and docs handoff

**Files:**
- Verify: `web/src/components/AI/replyRuntime.ts`
- Verify: `web/src/components/AI/AssistantReply.tsx`
- Verify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Verify: `web/src/components/AI/historyProjection.ts`

- [ ] **Step 1: Run focused frontend regression suite**

Run:
```bash
npm run test:run -- \
  web/src/components/AI/replyRuntime.test.ts \
  web/src/components/AI/historyProjection.test.ts \
  web/src/components/AI/__tests__/AssistantReply.test.tsx \
  web/src/components/AI/__tests__/PlatformChatProvider.test.ts
```
Expected: PASS all.

- [ ] **Step 2: Run typecheck/build guard**

Run: `cd web && npm run build`  
Expected: PASS (`tsc -b` and `vite build` complete with no type or bundling errors).

- [ ] **Step 3: Update plan progress notes (optional)**

```md
// append execution notes under this plan file during implementation
```

- [ ] **Step 4: Final commit**

```bash
git add web/src/components/AI web/src/api/modules/ai.ts
git commit -m "feat(ai-ui): ship inline tool narrative with approval fold and agent flow markers"
```
