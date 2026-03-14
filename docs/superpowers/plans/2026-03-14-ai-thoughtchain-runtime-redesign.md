# AI ThoughtChain Runtime Redesign Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a runtime-first `thoughtChain` chat flow that preserves markdown fidelity, renders structured plan/tool/replan nodes, uses one approval decision path, and persists restorable assistant answers.

**Architecture:** Delete remaining legacy `turn/block`, `phase/step`, and detached approval runtime wiring from the primary chat path first. Then tighten the canonical chain contract across backend SSE emission, frontend reducers/rendering, and session persistence so live streaming and restored sessions use the same `thoughtChain + final answer` model.

**Tech Stack:** Go 1.25, Gin, internal AI runtime/orchestrator packages, React 19, TypeScript, Vite, Ant Design X `ThoughtChain`, Vitest, Go test, OpenSpec.

---

## Scope Check

This remains one implementation plan for one subsystem: the AI chat runtime. Backend streaming, frontend rendering, replay persistence, approval, and observability all serve the same runtime contract, so keep them in one execution thread and commit in small vertical slices.

## File Structure Map

### Backend runtime contract and streaming

- Modify: `internal/ai/contracts.go`
  Responsibility: canonical chain/node payload types, including `headline`, `body`, `structured`, and `raw`.
- Modify: `internal/ai/events/events.go`
  Responsibility: canonical event names only for the primary path.
- Modify: `internal/ai/runtime/sse_converter.go`
  Responsibility: emit `chain_*`, `node_*`, `final_answer_*`, and `heartbeat`; remove legacy phase/block primary-path dependence.
- Modify: `internal/ai/runtime/runtime.go`
  Responsibility: runtime lifecycle entry points and state handoff between planner, executor, approval pause, replan, and final answer.
- Modify: `internal/ai/orchestrator.go`
  Responsibility: produce thoughtChain-native node payloads and final-answer flow.

### Backend approval and persistence

- Modify: `internal/service/ai/routes.go`
  Responsibility: unified chain approval decision route, no detached primary-path resume routes.
- Modify: `internal/service/ai/tooling_handlers.go`
  Responsibility: chain approval decision request/response handling.
- Modify: `internal/service/ai/session_recorder.go`
  Responsibility: persist runtime-first replay state and final answer flushes.
- Modify: `internal/ai/state/chat_store.go`
  Responsibility: store and retrieve canonical replay payloads.
- Modify: `internal/service/ai/handler.go`
  Responsibility: session detail/stream wiring for runtime-first replay and streaming.

### Backend observability

- Modify: `internal/service/ai/execution_observability.go`
  Responsibility: callback hook wiring from runtime lifecycle to metrics/traces.
- Modify: `internal/ai/observability/metrics.go`
  Responsibility: chain/node/approval/replan/final-answer metrics.
- Modify: `internal/ai/observability/metrics_test.go`
  Responsibility: observability regression coverage.

### Frontend streaming and state

- Modify: `web/src/api/modules/ai.ts`
  Responsibility: SSE parser, visible chunk normalization, replay DTOs, unified approval decision API.
- Modify: `web/src/api/modules/ai.test.ts`
  Responsibility: parser contract, DTO mapping, approval API expectations.
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
  Responsibility: markdown whitespace preservation and envelope normalization regression tests.
- Modify: `web/src/components/AI/types.ts`
  Responsibility: canonical runtime node types and replay-first message model.
- Modify: `web/src/components/AI/thoughtChainRuntime.ts`
  Responsibility: reducer logic for structured node payloads and final answer state.
- Modify: `web/src/components/AI/thoughtChainRuntime.test.ts`
  Responsibility: reducer regression coverage for structured payloads and event ordering.

### Frontend restore and rendering

- Modify: `web/src/components/AI/hooks/useConversationRestore.ts`
  Responsibility: restore assistant state from runtime-first replay.
- Modify: `web/src/components/AI/hooks/useConversationRestore.test.tsx`
  Responsibility: restore priority regression coverage.
- Modify: `web/src/components/AI/Copilot.tsx`
  Responsibility: one runtime-first assistant rendering path and new-session placeholder behavior.
- Modify: `web/src/components/AI/Copilot.test.tsx`
  Responsibility: live UI rendering, race handling, approval and final answer interactions.
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
  Responsibility: top-level node cards plus `ThoughtChain.Item`-style structured children.
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.test.tsx`
  Responsibility: plan/replan/tool rendering and approval card tests.
- Modify: `web/src/components/AI/components/RuntimeChain.css`
  Responsibility: node card, structured tool panel, and compact collapsed styling.
- Modify: `web/src/components/AI/components/FinalAnswerStream.tsx`
  Responsibility: final markdown-first answer rendering.
- Modify: `web/src/components/AI/components/ToolCard.tsx`
  Responsibility: beautified raw-result rendering fallback helpers if reused by thoughtChain tool nodes.

### Compatibility and cleanup

- Modify: `web/src/components/AI/turnLifecycle.ts`
  Responsibility: replay-only compatibility adapter; remove dead legacy helpers.
- Modify: `web/src/components/AI/components/AssistantMessageBlocks.tsx`
  Responsibility: keep only non-runtime message blocks if still needed.
- Modify: `internal/service/ai/routes_test.go`
  Responsibility: unified route and removed-route regression tests.
- Modify: `internal/ai/runtime/sse_converter_test.go`
  Responsibility: canonical event emission tests.

### Specs and docs to keep open while implementing

- Read: `docs/superpowers/specs/2026-03-14-ai-thoughtchain-runtime-redesign-design.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/proposal.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/design.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/tasks.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-thoughtchain-runtime/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-streaming-events/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-chat-session-contract/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-pre-execution-approval-gate/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-thoughtchain-observability/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/prometheus-integration/spec.md`

## Chunk 1: Streaming Contract and Parser Fidelity

### Task 1: Lock markdown-safe SSE parsing with failing tests

**Files:**
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
- Modify: `web/src/api/modules/ai.test.ts`
- Test: `web/src/api/modules/ai.streamChunk.test.ts`

- [ ] **Step 1: Write the failing SSE whitespace preservation test**

```ts
it('preserves markdown whitespace and blank lines from SSE data lines', async () => {
  const events: Array<{ type: string; data: unknown }> = [];
  await consumeAIStream(
    streamFromChunks([
      'event: final_answer_delta\n',
      'data: {"chunk":"## Title\\n\\n| A | B |\\n| - | - |"}\n\n',
    ]),
    { onEvent: (type, data) => events.push({ type, data }) },
  );
  expect(events).toEqual([
    {
      type: 'final_answer_delta',
      data: { chunk: '## Title\n\n| A | B |\n| - | - |' },
    },
  ]);
});
```

- [ ] **Step 2: Run the focused parser test to verify it fails**

Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts`
Expected: FAIL because `data:` lines are trimmed and markdown spacing is altered.

- [ ] **Step 3: Write the failing envelope normalization test**

```ts
it('unwraps only complete response envelopes', () => {
  expect(normalizeVisibleStreamChunk('{"response":"ok"}')).toBe('ok');
  expect(normalizeVisibleStreamChunk('{"response":')).toBe('{"response":');
  expect(normalizeVisibleStreamChunk('\n\n| a | b |\n')).toBe('\n\n| a | b |\n');
});
```

- [ ] **Step 4: Run the normalization test to verify it fails**

Run: `npm run test:run -- src/api/modules/ai.test.ts -t "unwraps only complete response envelopes"`
Expected: FAIL because the helper trims raw user-visible content.

- [ ] **Step 5: Commit the red tests**

```bash
git add web/src/api/modules/ai.streamChunk.test.ts web/src/api/modules/ai.test.ts
git commit -m "test: pin markdown-safe AI stream parsing"
```

### Task 2: Implement markdown-safe parsing and explicit envelope handling

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Test: `web/src/api/modules/ai.streamChunk.test.ts`
- Test: `web/src/api/modules/ai.test.ts`

- [ ] **Step 1: Remove line-level trimming from SSE parsing**

```ts
if (line.startsWith('event:')) {
  eventType = line.slice(6).trim();
}
if (line.startsWith('data:')) {
  dataLines.push(line.slice(5).replace(/^ /, ''));
}
```

- [ ] **Step 2: Make visible chunk normalization preserve raw markdown**

```ts
export function normalizeVisibleStreamChunk(rawChunk: string): string {
  if (typeof rawChunk !== 'string' || rawChunk === '') return '';
  const candidate = rawChunk;
  const trimmedForJSON = candidate.trim();
  if (!trimmedForJSON.startsWith('{') || !trimmedForJSON.endsWith('}')) {
    return candidate;
  }
  // parse only complete envelopes, otherwise return candidate unchanged
}
```

- [ ] **Step 3: Re-run focused parser tests**

Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts src/api/modules/ai.test.ts`
Expected: PASS.

- [ ] **Step 4: Run thoughtChain reducer tests to catch stream contract fallout**

Run: `npm run test:run -- src/components/AI/thoughtChainRuntime.test.ts src/components/AI/Copilot.test.tsx`
Expected: PASS or targeted failures showing downstream contract updates still needed.

- [ ] **Step 5: Commit parser fidelity fixes**

```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.streamChunk.test.ts web/src/api/modules/ai.test.ts
git commit -m "fix: preserve markdown in AI stream parser"
```

### Task 3: Lock backend canonical event emission with failing tests

**Files:**
- Modify: `internal/ai/runtime/sse_converter_test.go`
- Modify: `internal/service/ai/routes_test.go`
- Test: `internal/ai/runtime/sse_converter_test.go`

- [ ] **Step 1: Add a failing test that primary-path converter emits only canonical events**

```go
func TestSSEConverter_PrimaryPathDoesNotEmitLegacyRuntimeEvents(t *testing.T) {
	events := NewSSEConverter().ChainStarted("turn-1")
	for _, event := range events {
		if event.Type == "phase_started" || event.Type == "turn_started" || event.Type == "block_open" {
			t.Fatalf("unexpected legacy event: %s", event.Type)
		}
	}
}
```

- [ ] **Step 2: Run the backend test to verify it fails if legacy emission remains**

Run: `go test ./internal/ai/runtime -run TestSSEConverter_PrimaryPathDoesNotEmitLegacyRuntimeEvents -v`
Expected: FAIL if any legacy event names are still present on the primary path.

- [ ] **Step 3: Add a failing route test for the chain approval decision endpoint**

```go
func TestRegisterAIHandlers_RegistersChainApprovalDecisionRoute(t *testing.T) {
	routes := collectAIRoutesForTest()
	require.Contains(t, routes, "POST /api/v1/ai/chains/:chain_id/approvals/:node_id/decision")
}
```

- [ ] **Step 4: Run the focused route test**

Run: `go test ./internal/service/ai -run TestRegisterAIHandlers_RegistersChainApprovalDecisionRoute -v`
Expected: FAIL if the unified route is not the only primary approval path.

- [ ] **Step 5: Commit the red backend contract tests**

```bash
git add internal/ai/runtime/sse_converter_test.go internal/service/ai/routes_test.go
git commit -m "test: pin canonical thoughtchain backend contract"
```

## Chunk 2: Backend Runtime, Approval, and Persistence

### Task 4: Canonicalize backend event and node payload contracts

**Files:**
- Modify: `internal/ai/contracts.go`
- Modify: `internal/ai/events/events.go`
- Modify: `internal/ai/runtime/sse_converter.go`
- Modify: `internal/ai/orchestrator.go`
- Test: `internal/ai/runtime/sse_converter_test.go`

- [ ] **Step 1: Add canonical node payload fields in shared contracts**

```go
type ChainNodePayload struct {
	NodeID    string         `json:"node_id"`
	Kind      string         `json:"kind"`
	Title     string         `json:"title"`
	Headline  string         `json:"headline,omitempty"`
	Body      string         `json:"body,omitempty"`
	Structured map[string]any `json:"structured,omitempty"`
	Raw       any            `json:"raw,omitempty"`
	Status    string         `json:"status,omitempty"`
}
```

- [ ] **Step 2: Keep only canonical primary-path event names**

```go
const (
	EventChainStarted   = "chain_started"
	EventChainMeta      = "chain_meta"
	EventNodeOpen       = "node_open"
	EventNodeDelta      = "node_delta"
	EventNodeReplace    = "node_replace"
	EventNodeClose      = "node_close"
	EventChainPaused    = "chain_paused"
	EventChainResumed   = "chain_resumed"
	EventChainCompleted = "chain_completed"
	EventChainError     = "chain_error"
)
```

- [ ] **Step 3: Emit node payloads with separated `headline/body/structured/raw`**

```go
func toolNodeData(node ToolNode) map[string]any {
	return compactMap(map[string]any{
		"node_id":    node.NodeID,
		"kind":       "tool",
		"title":      node.Title,
		"headline":   node.Headline,
		"structured": node.Structured,
		"raw":        node.Raw,
		"status":     node.Status,
	})
}
```

- [ ] **Step 4: Run backend runtime tests**

Run: `go test ./internal/ai/runtime ./internal/ai/...`
Expected: PASS for updated converter tests; any failures should point to remaining payload-shape updates.

- [ ] **Step 5: Commit the backend contract slice**

```bash
git add internal/ai/contracts.go internal/ai/events/events.go internal/ai/runtime/sse_converter.go internal/ai/orchestrator.go internal/ai/runtime/sse_converter_test.go
git commit -m "refactor: canonicalize thoughtchain node payloads"
```

### Task 5: Unify approval flow on the same chain

**Files:**
- Modify: `internal/service/ai/routes.go`
- Modify: `internal/service/ai/tooling_handlers.go`
- Modify: `internal/ai/runtime/runtime.go`
- Modify: `internal/ai/runtime/approval.go`
- Test: `internal/service/ai/routes_test.go`

- [ ] **Step 1: Add the unified approval decision request type**

```go
type chainApprovalDecisionRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}
```

- [ ] **Step 2: Register only the chain approval decision route on the primary path**

```go
g.POST("/chains/:chain_id/approvals/:node_id/decision", h.DecideChainApproval)
```

- [ ] **Step 3: Resume or terminate the same chain from approval handling**

```go
func (h *Handler) DecideChainApproval(c *gin.Context) {
	// load chain_id + node_id context
	// resolve approval
	// resume same chain or close it as rejected
}
```

- [ ] **Step 4: Run approval route and runtime tests**

Run: `go test ./internal/service/ai -run TestRegisterAIHandlers_RegistersChainApprovalDecisionRoute -v && go test ./internal/service/ai/... ./internal/ai/...`
Expected: PASS.

- [ ] **Step 5: Commit the approval unification**

```bash
git add internal/service/ai/routes.go internal/service/ai/tooling_handlers.go internal/ai/runtime/runtime.go internal/ai/runtime/approval.go internal/service/ai/routes_test.go
git commit -m "refactor: unify chain approval decisions"
```

### Task 6: Persist runtime-first replay and terminal final answer flushes

**Files:**
- Modify: `internal/service/ai/session_recorder.go`
- Modify: `internal/ai/state/chat_store.go`
- Modify: `internal/service/ai/handler.go`
- Create or modify: `internal/service/ai/session_recorder_test.go`
- Test: `internal/service/ai/session_recorder_test.go`

- [ ] **Step 1: Write the failing persistence regression test**

```go
func TestSessionRecorder_CompletedAssistantTurnPersistsRuntimeAndFinalAnswer(t *testing.T) {
	recorder := newRecorderForTest(t)
	recorder.RecordFinalAnswerDelta("turn-1", "## Hosts\n\n| id | status |")
	recorder.RecordFinalAnswerDone("turn-1")

	session := recorder.SessionDetail("session-1")
	turn := findAssistantTurn(session, "turn-1")
	require.NotNil(t, turn.Runtime)
	require.NotEmpty(t, turn.Runtime.FinalAnswer.Content)
}
```

- [ ] **Step 2: Run the persistence test to verify it fails**

Run: `go test ./internal/service/ai -run TestSessionRecorder_CompletedAssistantTurnPersistsRuntimeAndFinalAnswer -v`
Expected: FAIL because completed turns can still persist empty replay payloads.

- [ ] **Step 3: Persist runtime-first replay and final-answer flush points**

```go
func (r *SessionRecorder) flushTurn(turn *RecordedTurn) error {
	turn.Runtime = buildRuntimeReplay(turn)
	turn.Blocks = deriveCompatibilityBlocks(turn.Runtime)
	return r.store.SaveTurn(turn)
}
```

- [ ] **Step 4: Run focused persistence and handler tests**

Run: `go test ./internal/service/ai -run TestSessionRecorder_ -v && go test ./internal/service/ai/...`
Expected: PASS.

- [ ] **Step 5: Commit the persistence slice**

```bash
git add internal/service/ai/session_recorder.go internal/ai/state/chat_store.go internal/service/ai/handler.go internal/service/ai/session_recorder_test.go
git commit -m "fix: persist runtime-first AI replay state"
```

### Task 7: Add callback-driven observability and Prometheus coverage

**Files:**
- Modify: `internal/service/ai/execution_observability.go`
- Modify: `internal/ai/observability/metrics.go`
- Modify: `internal/ai/observability/metrics_test.go`
- Test: `internal/ai/observability/metrics_test.go`

- [ ] **Step 1: Write the failing callback metric test**

```go
func TestObserveChainCompleted_RecordsDurationAndStatus(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)
	metrics.ObserveChainCompleted("deployment:hosts", "completed", time.Second)
	requireMetricExists(t, registry, "ai_thoughtchain_chain_duration_seconds")
}
```

- [ ] **Step 2: Run the metrics test to verify it fails**

Run: `go test ./internal/ai/observability -run TestObserveChainCompleted_RecordsDurationAndStatus -v`
Expected: FAIL if metrics/callback hooks are incomplete.

- [ ] **Step 3: Wire runtime callbacks to metrics observers**

```go
type RuntimeCallbacks struct {
	OnChainStarted   func(meta ChainMeta)
	OnNodeUpdated    func(meta NodeMeta)
	OnApprovalResolved func(meta ApprovalMeta)
	OnChainCompleted func(meta ChainMeta)
}
```

- [ ] **Step 4: Run observability tests**

Run: `go test ./internal/ai/observability ./internal/service/ai/...`
Expected: PASS.

- [ ] **Step 5: Commit observability updates**

```bash
git add internal/service/ai/execution_observability.go internal/ai/observability/metrics.go internal/ai/observability/metrics_test.go
git commit -m "feat: add thoughtchain observability callbacks"
```

## Chunk 3: Frontend Runtime, Rendering, and Replay

### Task 8: Rebuild runtime types and reducer around structured nodes

**Files:**
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/thoughtChainRuntime.ts`
- Modify: `web/src/components/AI/thoughtChainRuntime.test.ts`
- Test: `web/src/components/AI/thoughtChainRuntime.test.ts`

- [ ] **Step 1: Write the failing reducer test for structured node fields**

```ts
it('stores headline body structured and raw separately on tool nodes', () => {
  const state = reduceThoughtChainRuntimeEvent(undefined, {
    type: 'chain_node_open',
    data: {
      turn_id: 'turn-1',
      node_id: 'tool-1',
      kind: 'tool',
      title: 'host_list_inventory',
      headline: '已获取 5 台主机',
      body: '当前主机全部在线',
      structured: { columns: ['id', 'status'], rows: [[1, 'online']] },
      raw: { total: 5 },
      status: 'loading',
    },
  });
  expect(state.nodes[0]).toMatchObject({
    headline: '已获取 5 台主机',
    body: '当前主机全部在线',
  });
});
```

- [ ] **Step 2: Run the reducer test to verify it fails**

Run: `npm run test:run -- src/components/AI/thoughtChainRuntime.test.ts`
Expected: FAIL because runtime node types still collapse content into `summary/details`.

- [ ] **Step 3: Update runtime types and reducer minimally**

```ts
export interface RuntimeThoughtChainNode {
  nodeId: string;
  kind: RuntimeThoughtChainNodeKind;
  title: string;
  status: RuntimeThoughtChainNodeStatus;
  headline?: string;
  body?: string;
  structured?: Record<string, unknown>;
  raw?: unknown;
  approval?: ...
}
```

- [ ] **Step 4: Re-run reducer tests**

Run: `npm run test:run -- src/components/AI/thoughtChainRuntime.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit the runtime state model**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/thoughtChainRuntime.ts web/src/components/AI/thoughtChainRuntime.test.ts
git commit -m "refactor: split thoughtchain node content layers"
```

### Task 9: Restore runtime-first assistant history

**Files:**
- Modify: `web/src/components/AI/hooks/useConversationRestore.ts`
- Modify: `web/src/components/AI/hooks/useConversationRestore.test.tsx`
- Modify: `web/src/components/AI/turnLifecycle.ts`
- Test: `web/src/components/AI/hooks/useConversationRestore.test.tsx`

- [ ] **Step 1: Write the failing restore priority test**

```tsx
it('prefers persisted runtime and finalAnswer content over compatibility blocks', () => {
  const turn = buildAssistantTurn({
    runtime: { finalAnswer: { content: 'runtime answer', visible: true, streaming: false, revealState: 'complete' }, nodes: [] },
    blocks: [{ id: 'b1', type: 'text', position: 0, content: 'legacy answer' }],
  });
  expect(runtimeStateFromReplayTurn(turn)?.finalAnswer.content).toBe('runtime answer');
});
```

- [ ] **Step 2: Run the restore test to verify it fails**

Run: `npm run test:run -- src/components/AI/hooks/useConversationRestore.test.tsx`
Expected: FAIL if restore still privileges legacy block text.

- [ ] **Step 3: Make restore order runtime-first**

```ts
const restoredRuntime = turn.runtime ?? runtimeStateFromReplayTurn(turn);
const restoredContent =
  restoredRuntime?.finalAnswer.content ||
  deriveContentFromBlocks(turn.blocks) ||
  '';
```

- [ ] **Step 4: Re-run restore and Copilot tests**

Run: `npm run test:run -- src/components/AI/hooks/useConversationRestore.test.tsx src/components/AI/Copilot.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit the replay restore slice**

```bash
git add web/src/components/AI/hooks/useConversationRestore.ts web/src/components/AI/hooks/useConversationRestore.test.tsx web/src/components/AI/turnLifecycle.ts
git commit -m "fix: restore AI sessions from runtime-first replay"
```

### Task 10: Render structured plan and replan nodes

**Files:**
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.test.tsx`
- Modify: `web/src/components/AI/components/RuntimeChain.css`
- Test: `web/src/components/AI/components/RuntimeThoughtChain.test.tsx`

- [ ] **Step 1: Write the failing rendering test for plan steps**

```tsx
it('renders plan nodes as structured ThoughtChain items', () => {
  render(<RuntimeThoughtChain nodes={[{
    nodeId: 'plan-1',
    kind: 'plan',
    title: '动态调整计划',
    status: 'done',
    structured: {
      steps: [
        { id: 's1', title: '获取主机列表', status: 'done' },
        { id: 's2', title: '整理状态汇总', status: 'active' },
      ],
    },
  }]} />);
  expect(screen.getByText('获取主机列表')).toBeInTheDocument();
  expect(screen.getByText('整理状态汇总')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the rendering test to verify it fails**

Run: `npm run test:run -- src/components/AI/components/RuntimeThoughtChain.test.tsx`
Expected: FAIL because the component still renders only generic text/detail blocks.

- [ ] **Step 3: Implement structured plan/replan item rendering**

```tsx
function renderStructuredSteps(node: RuntimeThoughtChainNode) {
  const steps = asStepList(node.structured);
  return (
    <Flex gap="small" vertical>
      {steps.map((step) => (
        <ThoughtChain.Item key={step.id} variant="solid" status={toThoughtItemStatus(step.status)} title={step.title} description={step.description} />
      ))}
    </Flex>
  );
}
```

- [ ] **Step 4: Re-run RuntimeThoughtChain tests**

Run: `npm run test:run -- src/components/AI/components/RuntimeThoughtChain.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit structured plan rendering**

```bash
git add web/src/components/AI/components/RuntimeThoughtChain.tsx web/src/components/AI/components/RuntimeThoughtChain.test.tsx web/src/components/AI/components/RuntimeChain.css
git commit -m "feat: render structured thoughtchain plan nodes"
```

### Task 11: Render beautified raw tool results by default

**Files:**
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
- Modify: `web/src/components/AI/components/ToolCard.tsx`
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.test.tsx`
- Test: `web/src/components/AI/components/RuntimeThoughtChain.test.tsx`

- [ ] **Step 1: Write the failing tool result rendering test**

```tsx
it('renders host tool results as readable rows instead of one JSON blob', () => {
  render(<RuntimeThoughtChain nodes={[{
    nodeId: 'tool-1',
    kind: 'tool',
    title: 'host_list_inventory',
    status: 'done',
    structured: {
      resource: 'hosts',
      rows: [{ id: 1, name: 'test', status: 'online', ip: 'localhost' }],
    },
    raw: { total: 1, list: [{ id: 1, name: 'test', status: 'online', ip: 'localhost' }] },
  }]} />);
  expect(screen.getByText('test')).toBeInTheDocument();
  expect(screen.getByText('online')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the tool rendering test to verify it fails**

Run: `npm run test:run -- src/components/AI/components/RuntimeThoughtChain.test.tsx -t "renders host tool results as readable rows"`
Expected: FAIL because tool nodes still surface raw text or generic detail cards.

- [ ] **Step 3: Implement structured tool rendering with raw fallback**

```tsx
function renderToolStructured(node: RuntimeThoughtChainNode) {
  if (looksLikeHostRows(node.structured)) {
    return <HostResultTable rows={extractRows(node.structured)} />;
  }
  return <pre>{JSON.stringify(node.raw ?? node.structured, null, 2)}</pre>;
}
```

- [ ] **Step 4: Re-run tool and Copilot tests**

Run: `npm run test:run -- src/components/AI/components/RuntimeThoughtChain.test.tsx src/components/AI/Copilot.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit tool result beautification**

```bash
git add web/src/components/AI/components/RuntimeThoughtChain.tsx web/src/components/AI/components/ToolCard.tsx web/src/components/AI/components/RuntimeThoughtChain.test.tsx
git commit -m "feat: beautify thoughtchain tool results"
```

### Task 12: Keep final answer markdown separate and fix new-session rendering race

**Files:**
- Modify: `web/src/components/AI/Copilot.tsx`
- Modify: `web/src/components/AI/components/FinalAnswerStream.tsx`
- Modify: `web/src/components/AI/Copilot.test.tsx`
- Modify: `web/src/components/AI/AIAssistantDrawer.test.tsx`
- Test: `web/src/components/AI/Copilot.test.tsx`

- [ ] **Step 1: Write the failing recommended prompt race test**

```tsx
it('creates a placeholder assistant runtime immediately for recommended prompts', async () => {
  render(<Copilot />);
  await user.click(screen.getByText('查询所有服务器的状态'));
  expect(screen.queryByText('AI assistant unavailable')).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Write the failing final-answer separation test**

```tsx
it('renders final answer markdown outside process node bodies', () => {
  // mount runtime with replan body and finalAnswer content
  // assert markdown table is in final answer area, not duplicated under plan/replan card
});
```

- [ ] **Step 3: Run the focused UI tests**

Run: `npm run test:run -- src/components/AI/Copilot.test.tsx src/components/AI/AIAssistantDrawer.test.tsx`
Expected: FAIL where live UI still mixes runtime and legacy content or delays assistant container creation.

- [ ] **Step 4: Implement placeholder assistant creation and strict final-answer rendering boundary**

```tsx
const assistantMessage = {
  id: pendingId,
  role: 'assistant',
  content: '',
  runtime: createThoughtChainRuntimeState(),
  createdAt: new Date().toISOString(),
};
```

- [ ] **Step 5: Re-run key frontend runtime tests**

Run: `npm run test:run -- src/components/AI/Copilot.test.tsx src/components/AI/AIAssistantDrawer.test.tsx src/components/AI/hooks/useConversationRestore.test.tsx src/components/AI/components/RuntimeThoughtChain.test.tsx`
Expected: PASS.

- [ ] **Step 6: Commit the frontend runtime experience slice**

```bash
git add web/src/components/AI/Copilot.tsx web/src/components/AI/components/FinalAnswerStream.tsx web/src/components/AI/Copilot.test.tsx web/src/components/AI/AIAssistantDrawer.test.tsx
git commit -m "fix: stabilize runtime-first AI assistant rendering"
```

## Chunk 4: Validation, Cleanup, and Handoff

### Task 13: Remove final legacy primary-path branches and compatibility leaks

**Files:**
- Modify: `web/src/components/AI/turnLifecycle.ts`
- Modify: `web/src/components/AI/components/AssistantMessageBlocks.tsx`
- Modify: `web/src/components/AI/Copilot.tsx`
- Modify: `internal/service/ai/routes.go`
- Test: existing frontend/backend runtime tests

- [ ] **Step 1: Search for remaining primary-path legacy hooks**

Run: `cd /root/project/k8s-manage/.worktrees/ai-thoughtchain-runtime-redesign && rg -n "phase_started|phase_complete|plan_generated|step_started|step_complete|replan_triggered|approval_required|turn_started|block_open|block_delta|block_replace|block_close" internal/service/ai internal/ai web/src/components/AI web/src/api/modules/ai.ts`
Expected: only narrow replay compatibility or tests remain.

- [ ] **Step 2: Remove dead compatibility branches**

```ts
// delete unused phase/replan helpers once runtime-first restore/render paths are green
```

- [ ] **Step 3: Run targeted regression suites**

Run: `npm run test:run -- src/api/modules/ai.test.ts src/api/modules/ai.streamChunk.test.ts src/components/AI/thoughtChainRuntime.test.ts src/components/AI/hooks/useConversationRestore.test.tsx src/components/AI/Copilot.test.tsx src/components/AI/components/RuntimeThoughtChain.test.tsx && go test ./internal/ai/... ./internal/service/ai/...`
Expected: PASS.

- [ ] **Step 4: Commit legacy cleanup**

```bash
git add web/src/components/AI/turnLifecycle.ts web/src/components/AI/components/AssistantMessageBlocks.tsx web/src/components/AI/Copilot.tsx internal/service/ai/routes.go
git commit -m "refactor: remove legacy AI runtime branches"
```

### Task 14: Final verification and OpenSpec validation

**Files:**
- Modify: `openspec/changes/ai-thoughtchain-runtime-redesign/progress.md`
- Verify: OpenSpec artifacts and test output

- [ ] **Step 1: Run OpenSpec validation**

Run: `openspec validate ai-thoughtchain-runtime-redesign --json`
Expected: valid `true` with zero issues.

- [ ] **Step 2: Run final backend verification**

Run: `go test ./internal/ai/... ./internal/service/ai/...`
Expected: PASS.

- [ ] **Step 3: Run final frontend verification**

Run: `cd web && npm run test:run -- src/api/modules/ai.test.ts src/api/modules/ai.streamChunk.test.ts src/components/AI/thoughtChainRuntime.test.ts src/components/AI/hooks/useConversationRestore.test.tsx src/components/AI/Copilot.test.tsx src/components/AI/AIAssistantDrawer.test.tsx src/components/AI/components/RuntimeThoughtChain.test.tsx && npm run build`
Expected: PASS.

- [ ] **Step 4: Update progress tracking**

```md
- Completed runtime-first thoughtChain redesign slices for streaming, persistence, UI, and approval.
- Verified OpenSpec change and project test suites.
```

- [ ] **Step 5: Commit verification and progress updates**

```bash
git add openspec/changes/ai-thoughtchain-runtime-redesign/progress.md
git commit -m "docs: update thoughtchain redesign progress"
```

## Notes For The Implementer

- Prefer vertical slices that produce one meaningful behavior change per commit.
- Keep live rendering and restore rendering on the same runtime model. If a fallback is needed, treat it as replay-only compatibility.
- Do not reintroduce generic trimming in the parser or generic summary buckets in runtime node payloads.
- If a tool result shape is not recognized, ship a formatted raw fallback instead of inventing prose.
- If any completed assistant turn can still serialize with empty runtime/final answer state, stop and fix persistence before touching more UI.
