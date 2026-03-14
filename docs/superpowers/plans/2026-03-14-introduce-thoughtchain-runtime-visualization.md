# ThoughtChain Runtime Visualization Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current block/card-first AI process UI with a native ThoughtChain runtime model where the backend emits chain-node events, the frontend renders a single narrative chain, and the final answer starts streaming only after the chain collapses.

**Architecture:** The backend becomes the source of truth for narrative chain semantics by emitting `chain_*` and `final_answer_*` SSE events from the orchestrator/runtime layer while still projecting compatibility events during rollout. The frontend stops inferring chain state from generic phase/tool events and instead consumes a dedicated reducer/state machine that drives `ThoughtChain`, collapse transitions, approval-in-chain rendering, and delayed final-answer typewriter streaming.

**Tech Stack:** Go, Gin, SSE, existing `internal/ai` runtime/orchestrator, React, TypeScript, `@ant-design/x`, Vitest, Go test

---

## File Map

### Backend

- Modify: `internal/ai/events/events.go`
  - Define native chain event constants and keep compatibility constants intact.
- Modify: `internal/ai/runtime/runtime.go`
  - Add chain node payload types, final answer payload types, and any shared status enums.
- Modify: `internal/ai/runtime/sse_converter.go`
  - Centralize native chain event creation and compatibility projection.
- Modify: `internal/ai/orchestrator.go`
  - Emit narrative chain lifecycle directly from runtime flow.
- Modify: `internal/ai/orchestrator_test.go`
  - Assert event sequence and semantic guarantees.
- Modify: `internal/service/ai/handler_aiv2_test.go`
  - Verify SSE transport emits the new native event order.
- Modify: `internal/ai/state/chat_store.go`
  - Persist chain node and final answer state in a replayable shape.
- Modify: `internal/ai/state/chat_store_test.go`
  - Prove storage ownership of node order, collapse state, and final answer replay fields.
- Modify: `internal/service/ai/session_recorder.go`
  - Record native chain/final-answer lifecycle without regressing legacy session behavior.

### Frontend

- Modify: `web/src/api/modules/ai.ts`
  - Add native chain/final-answer SSE types and handlers.
- Modify: `web/src/components/AI/types.ts`
  - Own shared UI-facing ThoughtChain node, final-answer, and replay types to avoid drift.
- Create: `web/src/components/AI/thoughtChainRuntime.ts`
  - Native chain reducer/state helpers; source of truth for the new process UI.
- Create: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
  - `ThoughtChain`-based process renderer with inline approval slot.
- Create: `web/src/components/AI/components/FinalAnswerStream.tsx`
  - Buffered typewriter renderer for delayed final answer display.
- Modify: `web/src/components/AI/Copilot.tsx`
  - Replace block/card-first process rendering with the native ThoughtChain runtime.
- Modify: `web/src/components/AI/hooks/useConversationRestore.ts`
  - Restore incomplete vs completed sessions into chain + final answer states.
- Modify: `web/src/components/AI/Copilot.test.tsx`
  - Cover chain rendering and delayed final answer behavior.
- Create: `web/src/components/AI/thoughtChainRuntime.test.ts`
  - Reducer tests for node lifecycle and collapse gating.
- Create: `web/src/components/AI/components/FinalAnswerStream.test.tsx`
  - Buffered typewriter tests.

### Docs

- Modify: `openspec/changes/introduce-thoughtchain-runtime-visualization/tasks.md`
  - Check off completed work as implementation proceeds.
- Modify: `docs/ai-api.md`
  - Document the native chain SSE protocol and compatibility notes.

## Chunk 1: Backend Native Chain Protocol

### Task 1: Define event constants and payload types

**Files:**
- Modify: `internal/ai/events/events.go`
- Modify: `internal/ai/runtime/runtime.go`
- Test: `internal/ai/orchestrator_test.go`

- [ ] **Step 1: Write the failing backend payload test**

Add a test case in `internal/ai/orchestrator_test.go` that expects native chain event names like `chain_node_open` and payload keys like `node_id`, `kind`, and `title`.

- [ ] **Step 2: Run the focused test to verify it fails**

Run: `go test ./internal/ai -run 'Test.*Chain.*'`
Expected: FAIL because the new event constants or payload fields do not exist yet.

- [ ] **Step 3: Add native chain event constants**

Update `internal/ai/events/events.go` to define:

```go
const (
    EventChainStarted     Name = "chain_started"
    EventChainNodeOpen    Name = "chain_node_open"
    EventChainNodePatch   Name = "chain_node_patch"
    EventChainNodeClose   Name = "chain_node_close"
    EventChainCollapsed   Name = "chain_collapsed"
    EventFinalAnswerStart Name = "final_answer_started"
    EventFinalAnswerDelta Name = "final_answer_delta"
    EventFinalAnswerDone  Name = "final_answer_done"
)
```

- [ ] **Step 4: Add payload structs**

Update `internal/ai/runtime/runtime.go` with focused types like:

```go
type ChainNodeKind string

const (
    ChainNodePlan     ChainNodeKind = "plan"
    ChainNodeExecute  ChainNodeKind = "execute"
    ChainNodeTool     ChainNodeKind = "tool"
    ChainNodeReplan   ChainNodeKind = "replan"
    ChainNodeApproval ChainNodeKind = "approval"
)

type ChainNodeInfo struct {
    TurnID     string         `json:"turn_id"`
    NodeID     string         `json:"node_id"`
    Kind       ChainNodeKind  `json:"kind"`
    Title      string         `json:"title"`
    Status     string         `json:"status"`
    Summary    string         `json:"summary,omitempty"`
    Details    map[string]any `json:"details,omitempty"`
    StartedAt  string         `json:"started_at,omitempty"`
    FinishedAt string         `json:"finished_at,omitempty"`
    Approval   map[string]any `json:"approval,omitempty"`
}

type FinalAnswerDelta struct {
    TurnID string `json:"turn_id"`
    Chunk  string `json:"chunk"`
}
```

- [ ] **Step 5: Run the focused test to verify it passes**

Run: `go test ./internal/ai -run 'Test.*Chain.*'`
Expected: PASS or move failure to converter/orchestrator logic instead of missing types.

- [ ] **Step 6: Commit**

```bash
git add internal/ai/events/events.go internal/ai/runtime/runtime.go internal/ai/orchestrator_test.go
git commit -m "feat: add native thoughtchain event types"
```

### Task 2: Refactor SSE converter to emit native chain events

**Files:**
- Modify: `internal/ai/runtime/sse_converter.go`
- Test: `internal/ai/runtime/sse_converter_test.go`

- [ ] **Step 1: Write failing converter tests**

Add tests that call converter helpers and expect:
- `chain_node_open` payloads for plan/tool/approval nodes
- `chain_node_patch` to update details without creating a new node
- `final_answer_delta` to carry append-only chunks

- [ ] **Step 2: Run converter tests to verify they fail**

Run: `go test ./internal/ai/runtime -run 'TestSSEConverter.*Chain|TestSSEConverter.*FinalAnswer'`
Expected: FAIL because helper methods do not exist.

- [ ] **Step 3: Implement converter helpers**

Add focused methods:

```go
func (c *SSEConverter) OnChainStarted(turnID string) StreamEvent
func (c *SSEConverter) OnChainNodeOpen(info *ChainNodeInfo) StreamEvent
func (c *SSEConverter) OnChainNodePatch(info *ChainNodeInfo) StreamEvent
func (c *SSEConverter) OnChainNodeClose(info *ChainNodeInfo) StreamEvent
func (c *SSEConverter) OnChainCollapsed(turnID string) StreamEvent
func (c *SSEConverter) OnFinalAnswerStarted(turnID string) StreamEvent
func (c *SSEConverter) OnFinalAnswerDelta(turnID, chunk string) StreamEvent
func (c *SSEConverter) OnFinalAnswerDone(turnID string) StreamEvent
```

- [ ] **Step 4: Preserve compatibility projection**

Keep current compatibility helpers, but mark native chain helpers as the primary path used by the orchestrator.

- [ ] **Step 5: Add compatibility projection assertions**

Extend `internal/ai/runtime/sse_converter_test.go` to prove native chain helpers still project required legacy SSE fields for compatibility clients during rollout.

- [ ] **Step 6: Run converter tests**

Run: `go test ./internal/ai/runtime -run 'TestSSEConverter.*Chain|TestSSEConverter.*FinalAnswer'`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/ai/runtime/sse_converter.go internal/ai/runtime/sse_converter_test.go
git commit -m "feat: add native thoughtchain sse converter"
```

### Task 3: Move orchestrator to native narrative chain emission

**Files:**
- Modify: `internal/ai/orchestrator.go`
- Test: `internal/ai/orchestrator_test.go`

- [ ] **Step 1: Write failing orchestrator sequence tests**

Add tests covering:
- normal plan -> execute -> tool -> final answer flow
- approval flow
- approval rejection -> terminate chain -> final answer flow
- replanning flow
- guarantee that `tool_result` updates the active tool node rather than opening a new node
- guarantee that `chain_collapsed` is emitted before `final_answer_started`
- guarantee that compatibility events remain available for legacy consumers during rollout

- [ ] **Step 2: Run the focused orchestrator tests**

Run: `go test ./internal/ai -run 'TestStreamExecution.*Chain|TestResumeApprovedStream.*Chain'`
Expected: FAIL with old event sequence.

- [ ] **Step 3: Refactor runtime state handling**

In `internal/ai/orchestrator.go`:
- create/open nodes only when the runtime actually enters a new narrative step
- close the active node before opening the next node
- patch active tool/approval nodes instead of creating extra nodes
- separate process-chain output from final answer output

- [ ] **Step 4: Add final-answer gating**

Ensure:
- planner JSON / tool args / replanning notes never enter final answer chunks
- `chain_collapsed` is emitted before final answer streaming starts
- final answer chunks are append-only and dedicated to user-facing result prose

- [ ] **Step 5: Run orchestrator tests**

Run: `go test ./internal/ai -run 'TestStreamExecution.*Chain|TestResumeApprovedStream.*Chain|TestEventTextContents|TestMergeTextProgress'`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/orchestrator.go internal/ai/orchestrator_test.go
git commit -m "feat: emit native thoughtchain runtime flow"
```

### Task 4: Update persistence and session recording for chain/final-answer separation

**Files:**
- Modify: `internal/ai/state/chat_store.go`
- Modify: `internal/ai/state/chat_store_test.go`
- Modify: `internal/service/ai/session_recorder.go`
- Test: `internal/service/ai/handler_aiv2_test.go`

- [ ] **Step 1: Write failing persistence tests**

Add tests that restore:
- an incomplete chain-only turn
- a completed turn with collapsed chain and final answer
- a turn where approval was pending and later resumed
- a turn where approval was rejected and the chain terminated into final answer

- [ ] **Step 2: Add direct store tests first**

In `internal/ai/state/chat_store_test.go`, verify store ownership of:
- node ordering
- node detail/status updates
- collapse state
- final-answer content and completion flags

- [ ] **Step 3: Run the focused tests**

Run: `go test ./internal/ai/state ./internal/service/ai -run 'Test.*ThoughtChain'`
Expected: FAIL because recorder/store do not yet persist the new shape.

- [ ] **Step 4: Persist native chain state**

Update recorder/store so native events write replayable chain state and final answer state separately. Keep compatibility message writes during rollout, but do not let them become the only source of truth.

- [ ] **Step 5: Add compatibility replay assertions**

Extend `internal/service/ai/handler_aiv2_test.go` to verify legacy message-compatible fields still exist after native chain persistence is introduced.

- [ ] **Step 6: Run tests**

Run: `go test ./internal/ai/state ./internal/service/ai -run 'Test.*ThoughtChain|Test.*Compatibility'`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/ai/state/chat_store.go internal/ai/state/chat_store_test.go internal/service/ai/session_recorder.go internal/service/ai/handler_aiv2_test.go
git commit -m "feat: persist thoughtchain replay state"
```

## Chunk 2: Frontend ThoughtChain Runtime

### Task 5: Add native SSE types and reducer tests

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/types.ts`
- Create: `web/src/components/AI/thoughtChainRuntime.ts`
- Create: `web/src/components/AI/thoughtChainRuntime.test.ts`

- [ ] **Step 1: Write failing reducer tests**

Add tests for:
- node appears only when `chain_node_open` arrives
- previous node becomes done when closed and next node opens
- approval node holds embedded approval data
- `chain_collapsed` hides final answer until collapse completes

- [ ] **Step 2: Run the reducer tests**

Run: `npm run test:run -- src/components/AI/thoughtChainRuntime.test.ts`
Expected: FAIL because the runtime reducer does not exist.

- [ ] **Step 3: Implement native SSE typings and shared UI types**

Extend `web/src/api/modules/ai.ts` with transport-level SSE payloads and put shared UI-facing runtime types in `web/src/components/AI/types.ts`.

Example transport type in `web/src/api/modules/ai.ts`:

```ts
export interface SSEChainNodeEvent {
  turn_id: string;
  node_id: string;
  kind: 'plan' | 'execute' | 'tool' | 'replan' | 'approval';
  title: string;
  status: 'loading' | 'done' | 'error' | 'waiting';
  summary?: string;
  details?: Record<string, unknown>;
  approval?: Record<string, unknown>;
}
```

Example shared UI type in `web/src/components/AI/types.ts`:

```ts
export interface RuntimeThoughtChainNode {
  id: string;
  kind: 'plan' | 'execute' | 'tool' | 'replan' | 'approval';
  title: string;
  status: 'loading' | 'done' | 'error' | 'waiting';
  summary?: string;
  details?: Record<string, unknown>;
  approval?: ConfirmationRequest;
}
```

- [ ] **Step 4: Implement reducer helpers**

Create `thoughtChainRuntime.ts` with pure functions:
- `createThoughtChainState`
- `applyChainStarted`
- `applyChainNodeOpen`
- `applyChainNodePatch`
- `applyChainNodeClose`
- `applyChainCollapsed`
- `applyFinalAnswerStarted`
- `applyFinalAnswerDelta`
- `applyFinalAnswerDone`

- [ ] **Step 5: Run reducer tests**

Run: `npm run test:run -- src/components/AI/thoughtChainRuntime.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/api/modules/ai.ts web/src/components/AI/thoughtChainRuntime.ts web/src/components/AI/thoughtChainRuntime.test.ts
git commit -m "feat: add frontend thoughtchain runtime reducer"
```

### Task 6: Build `RuntimeThoughtChain` with inline approval slot

**Files:**
- Create: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
- Test: `web/src/components/AI/Copilot.test.tsx`

- [ ] **Step 1: Write a failing component test**

Add a test that renders:
- one loading plan node
- one completed plan node plus one loading tool node
- one approval node with inline `ConfirmationPanel`

- [ ] **Step 2: Run the component test**

Run: `npm run test:run -- src/components/AI/Copilot.test.tsx`
Expected: FAIL because the component does not exist or render path is still block-first.

- [ ] **Step 3: Implement `RuntimeThoughtChain.tsx`**

Build a focused component that:
- takes normalized chain nodes
- renders `ThoughtChain`
- uses user-narrative titles only
- renders details in expandable content
- mounts `ConfirmationPanel` inside approval node content

- [ ] **Step 4: Run the component test**

Run: `npm run test:run -- src/components/AI/Copilot.test.tsx`
Expected: PASS or shift failures to `Copilot` integration.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/components/RuntimeThoughtChain.tsx web/src/components/AI/Copilot.test.tsx
git commit -m "feat: render runtime thoughtchain"
```

### Task 7: Add delayed final-answer typewriter renderer

**Files:**
- Create: `web/src/components/AI/components/FinalAnswerStream.tsx`
- Create: `web/src/components/AI/components/FinalAnswerStream.test.tsx`

- [ ] **Step 1: Write failing typewriter tests**

Add tests for:
- nothing renders before `visible=true`
- chunks append in order
- reduced-motion mode degrades to immediate append

- [ ] **Step 2: Run the tests**

Run: `npm run test:run -- src/components/AI/components/FinalAnswerStream.test.tsx`
Expected: FAIL because the component does not exist.

- [ ] **Step 3: Implement `FinalAnswerStream.tsx`**

Use a small buffered renderer:
- buffer incoming chunks
- reveal them with a short interval for typewriter feel
- flush immediately when reduced motion is enabled or stream completes

- [ ] **Step 4: Run the tests**

Run: `npm run test:run -- src/components/AI/components/FinalAnswerStream.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/components/FinalAnswerStream.tsx web/src/components/AI/components/FinalAnswerStream.test.tsx
git commit -m "feat: add delayed final answer stream"
```

### Task 8: Integrate `Copilot` with native ThoughtChain runtime

**Files:**
- Modify: `web/src/components/AI/Copilot.tsx`
- Modify: `web/src/components/AI/hooks/useConversationRestore.ts`
- Test: `web/src/components/AI/Copilot.test.tsx`

- [ ] **Step 1: Write failing integration tests**

Cover:
- active stream shows only ThoughtChain
- final answer starts only after collapse
- completed restored session shows collapsed chain + final answer
- incomplete restored session shows active chain without final answer
- rejected approval restored session shows terminal chain + final answer without resumed execution
- compatibility fallback still renders usable history when native chain data is absent

- [ ] **Step 2: Run the integration tests**

Run: `npm run test:run -- src/components/AI/Copilot.test.tsx src/components/AI/hooks/useConversationRestore.test.tsx`
Expected: FAIL because `Copilot` still relies on block/card-first process rendering.

- [ ] **Step 3: Refactor `Copilot.tsx`**

Replace the current main process render path with:
- native chain state consumption
- `RuntimeThoughtChain`
- `FinalAnswerStream`
- approval action wiring through the approval node

Keep compatibility reads only where needed for rollout or historical fallback.

- [ ] **Step 4: Refactor restore path**

Update `useConversationRestore.ts` so replay uses native chain/final-answer state first and only falls back to legacy message fields when the new shape is absent.

- [ ] **Step 5: Run the integration tests**

Run: `npm run test:run -- src/components/AI/Copilot.test.tsx src/components/AI/hooks/useConversationRestore.test.tsx`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/Copilot.tsx web/src/components/AI/hooks/useConversationRestore.ts web/src/components/AI/Copilot.test.tsx
git commit -m "feat: switch copilot to native thoughtchain runtime"
```

## Chunk 3: Motion, API Verification, and Docs

### Task 9: Add collapse and transition behavior

**Files:**
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
- Modify: `web/src/components/AI/Copilot.tsx`
- Test: `web/src/components/AI/Copilot.test.tsx`

- [ ] **Step 1: Write failing motion-state tests**

Add tests for:
- node loading -> done transitions
- chain collapse flag flips before final answer visibility
- reduced motion bypasses delayed collapse animation

- [ ] **Step 2: Run tests**

Run: `npm run test:run -- src/components/AI/Copilot.test.tsx`
Expected: FAIL because collapse/transition state is incomplete.

- [ ] **Step 3: Implement transition state**

Add small, explicit state phases:
- `idle`
- `streaming`
- `collapsing`
- `collapsed`

Keep animation ownership in one place to avoid split timing bugs.

- [ ] **Step 4: Run tests**

Run: `npm run test:run -- src/components/AI/Copilot.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/components/RuntimeThoughtChain.tsx web/src/components/AI/Copilot.tsx web/src/components/AI/Copilot.test.tsx
git commit -m "feat: add thoughtchain collapse transitions"
```

### Task 10: Verify API and replay behavior end to end

**Files:**
- Modify: `internal/service/ai/handler_aiv2_test.go`
- Modify: `web/src/components/AI/Copilot.tsx`
- Modify: `web/src/components/AI/hooks/useConversationRestore.test.tsx`
- Modify: `docs/ai-api.md`

- [ ] **Step 1: Write failing API/replay tests**

Add tests asserting:
- SSE emits `chain_collapsed` before `final_answer_started`
- session detail restores completed and incomplete chain states correctly
- rejected approval restores terminal chain state correctly
- compatibility clients still receive legacy SSE/message fields during rollout

- [ ] **Step 2: Run the tests**

Run: `go test ./internal/service/ai -run 'TestAIChatStream.*ThoughtChain'`
Run: `npm run test:run -- src/components/AI/hooks/useConversationRestore.test.tsx`
Expected: FAIL until API docs and replay code match the new contract.

- [ ] **Step 3: Fix replay and compatibility handling**

Update `web/src/components/AI/Copilot.tsx` and any needed wiring so completed, incomplete, and rejected sessions follow the new runtime contract while compatibility fallback remains intact.

- [ ] **Step 4: Update docs and replay expectations**

Document:
- native event list
- event ordering guarantees
- compatibility note for old clients

- [ ] **Step 5: Run the tests again**

Run: `go test ./internal/service/ai -run 'TestAIChatStream.*ThoughtChain'`
Run: `npm run test:run -- src/components/AI/hooks/useConversationRestore.test.tsx`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/handler_aiv2_test.go web/src/components/AI/Copilot.tsx web/src/components/AI/hooks/useConversationRestore.test.tsx docs/ai-api.md
git commit -m "docs: record thoughtchain runtime api contract"
```

### Task 11: Full verification pass and task sync

**Files:**
- Modify: `openspec/changes/introduce-thoughtchain-runtime-visualization/tasks.md`

- [ ] **Step 1: Run backend verification**

Run: `go test ./internal/ai ./internal/ai/runtime ./internal/service/ai`
Expected: PASS

- [ ] **Step 2: Run frontend verification**

Run: `npm run test:run -- src/components/AI/thoughtChainRuntime.test.ts src/components/AI/components/FinalAnswerStream.test.tsx src/components/AI/Copilot.test.tsx src/components/AI/hooks/useConversationRestore.test.tsx`
Expected: PASS

- [ ] **Step 3: Run build verification**

Run: `npm run build`
Expected: PASS

- [ ] **Step 4: Sync OpenSpec task state**

Mark completed items in `openspec/changes/introduce-thoughtchain-runtime-visualization/tasks.md`.

- [ ] **Step 5: Commit**

```bash
git add openspec/changes/introduce-thoughtchain-runtime-visualization/tasks.md
git commit -m "chore: sync thoughtchain runtime implementation tasks"
```
