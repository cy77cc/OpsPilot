# AI ThoughtChain Runtime Redesign Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the mixed AI chat runtime with a deletion-first, `thoughtChain`-only execution model that fixes chain rendering, approval pause/resume, new-session prompt races, and Prometheus observability.

**Architecture:** Start by deleting legacy `turn/block`, `phase/step`, and detached approval semantics from the AI chat primary path so they stop shaping new code. Then introduce one canonical `thoughtChain` contract across backend streaming, replay persistence, frontend state, approval decisions, and metrics callbacks, keeping live rendering and replay on the same model.

**Tech Stack:** Go 1.25, Gin, GORM, React 19, TypeScript, Vite, Ant Design X, Jest/Vitest-style frontend tests, Go test, Prometheus client.

---

## Scope Check

This spec is still one coherent subsystem: AI chat runtime. It touches backend streaming, approval, frontend rendering, replay, and metrics, but those pieces all serve one runtime contract and one user flow. Do not split implementation unless a concrete blocker forces it.

## File Structure Map

### Backend runtime and transport

- Modify: `internal/ai/events/events.go`
  Responsibility: canonical event names; delete old primary-path constants.
- Modify: `internal/ai/runtime/sse_converter.go`
  Responsibility: convert runtime lifecycle into SSE events; stop emitting legacy phase/step/approval events on the primary path.
- Modify: `internal/ai/runtime/runtime.go`
  Responsibility: runtime orchestration entry points; thread chain identity and approval pause/resume.
- Modify: `internal/ai/orchestrator.go`
  Responsibility: planner/executor/replan orchestration; emit thoughtChain-native lifecycle only.
- Modify: `internal/ai/contracts.go`
  Responsibility: shared runtime contract types.

### Backend AI service and persistence

- Modify: `internal/service/ai/routes.go`
  Responsibility: remove detached approval/resume primary-path routes, add unified chain approval decision route.
- Modify: `internal/service/ai/handler.go`
  Responsibility: chat streaming/session response wiring.
- Modify: `internal/service/ai/tooling_handlers.go`
  Responsibility: approval create/approve/reject handlers; refactor toward chain decision semantics.
- Modify: `internal/service/ai/session_recorder.go`
  Responsibility: persist canonical thoughtChain nodes and final answer, delete legacy stage synthesis.
- Modify: `internal/ai/state/chat_store.go`
  Responsibility: store/retrieve assistant replay state.
- Modify: `internal/service/ai/execution_observability.go`
  Responsibility: hook runtime lifecycle to observability.

### Backend observability

- Modify: `internal/ai/observability/metrics.go`
  Responsibility: add chain/node/approval/replan metrics.
- Create: `internal/ai/observability/thoughtchain.go`
  Responsibility: focused helpers for chain/node metric observation if `metrics.go` becomes crowded.

### Frontend API and runtime state

- Modify: `web/src/api/modules/ai.ts`
  Responsibility: canonical SSE event parsing, session/replay DTOs, unified approval decision API.
- Modify: `web/src/components/AI/types.ts`
  Responsibility: canonical thoughtChain node/store types; remove legacy compatibility typing.
- Modify: `web/src/components/AI/thoughtChainRuntime.ts`
  Responsibility: single reducer for chain/node/final-answer runtime.
- Modify: `web/src/components/AI/turnLifecycle.ts`
  Responsibility: replay reconstruction; likely shrink or become adapter-only during migration, then delete dead legacy helpers.
- Modify: `web/src/components/AI/hooks/useConversationRestore.ts`
  Responsibility: restore assistant state from canonical thoughtChain replay.

### Frontend UI

- Modify: `web/src/components/AI/Copilot.tsx`
  Responsibility: stop merging old state models; drive message UI from one chain store.
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
  Responsibility: render native chain nodes, approval card, final answer separation.
- Modify: `web/src/components/AI/components/AssistantMessageBlocks.tsx`
  Responsibility: either narrow to non-runtime content or remove legacy fallback use in chat runtime.
- Modify: `web/src/components/AI/AISurfaceBoundary.tsx`
  Responsibility: unavailable-state behavior.
- Modify: `web/src/components/AI/components/RuntimeChain.css`
  Responsibility: upgraded node card styling for plan/tool/approval/replan/answer.

### Tests

- Modify: `web/src/api/modules/ai.test.ts`
- Modify: `web/src/components/AI/thoughtChainRuntime.test.ts`
- Modify: `web/src/components/AI/hooks/useConversationRestore.test.ts`
- Modify: `web/src/components/AI/Copilot.test.tsx`
- Modify: `web/src/components/AI/AIAssistantDrawer.test.tsx`
- Create or modify: backend tests near `internal/service/ai` and `internal/ai/runtime`

### Specs and docs to keep open during implementation

- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/proposal.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/design.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/tasks.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-thoughtchain-runtime/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-streaming-events/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-pre-execution-approval-gate/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-chat-session-contract/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/prometheus-integration/spec.md`
- Read: `openspec/changes/ai-thoughtchain-runtime-redesign/specs/ai-thoughtchain-observability/spec.md`

## Chunk 1: Delete Legacy Runtime Surfaces

### Task 1: Freeze the legacy baseline with failing tests

**Files:**
- Modify: `web/src/api/modules/ai.test.ts`
- Modify: `web/src/components/AI/Copilot.test.tsx`
- Modify: `web/src/components/AI/thoughtChainRuntime.test.ts`
- Create: `internal/ai/runtime/sse_converter_test.go`
- Create: `internal/service/ai/routes_test.go`

- [ ] **Step 1: Write a failing backend test for legacy event non-emission**

```go
func TestSSEConverter_PrimaryPathDoesNotEmitLegacyPhaseEvents(t *testing.T) {
	converter := NewSSEConverter()
	events := converter.OnPlannerStart("sess-1", "plan-1", "turn-1")
	for _, event := range events {
		if event.Type == EventPhaseStarted || event.Type == EventTurnStarted {
			t.Fatalf("unexpected legacy event on primary path: %s", event.Type)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/runtime -run TestSSEConverter_PrimaryPathDoesNotEmitLegacyPhaseEvents -v`
Expected: FAIL because `OnPlannerStart` still emits `turn_started` and `phase_started`.

- [ ] **Step 3: Write a failing frontend test for phase-event dependence**

```tsx
it('ignores legacy phase events for the primary chain reducer', () => {
  const state = reduceThoughtChainRuntimeEvent(undefined, {
    // @ts-expect-error verifying legacy event rejection during migration
    type: 'phase_started',
    data: { phase: 'planning', status: 'loading' },
  });
  expect(state.nodes).toHaveLength(0);
});
```

- [ ] **Step 4: Run the focused frontend tests**

Run: `npm run test:run -- web/src/components/AI/thoughtChainRuntime.test.ts web/src/api/modules/ai.test.ts`
Expected: FAIL because parsing and reducer paths still accept legacy phase/approval events.

- [ ] **Step 5: Commit the red tests**

```bash
git add web/src/api/modules/ai.test.ts web/src/components/AI/Copilot.test.tsx web/src/components/AI/thoughtChainRuntime.test.ts internal/ai/runtime/sse_converter_test.go internal/service/ai/routes_test.go
git commit -m "test: pin legacy AI runtime removal baseline"
```

### Task 2: Remove legacy event names and converters from the backend primary path

**Files:**
- Modify: `internal/ai/events/events.go`
- Modify: `internal/ai/runtime/sse_converter.go`
- Modify: `internal/ai/runtime/runtime.go`
- Modify: `internal/ai/orchestrator.go`
- Test: `internal/ai/runtime/sse_converter_test.go`

- [ ] **Step 1: Remove legacy primary-path event constants and keep only canonical thoughtChain events**

```go
const (
	ChainStarted   Name = "chain_started"
	ChainMeta      Name = "chain_meta"
	NodeOpen       Name = "node_open"
	NodeDelta      Name = "node_delta"
	NodeReplace    Name = "node_replace"
	NodeClose      Name = "node_close"
	ChainPaused    Name = "chain_paused"
	ChainResumed   Name = "chain_resumed"
	ChainCompleted Name = "chain_completed"
	ChainError     Name = "chain_error"
	Heartbeat      Name = "heartbeat"
)
```

- [ ] **Step 2: Update the SSE converter to emit only canonical chain events**

```go
func (c *SSEConverter) OnApprovalPaused(chainID string, node ChainNodeInfo) []StreamEvent {
	return []StreamEvent{
		{Type: EventNodeOpen, Data: chainNodeData(node)},
		{Type: EventChainPaused, Data: compactMap(map[string]any{"chain_id": chainID, "node_id": node.NodeID})},
	}
}
```

- [ ] **Step 3: Remove phase/step/replan compatibility emission from orchestrator and runtime**

Run search: `rg -n "PhaseStarted|PhaseComplete|PlanGenerated|StepStarted|StepComplete|ReplanTriggered|ApprovalRequired|TurnStarted" internal/ai internal/service/ai`
Expected: only non-chat or test references remain after the edit.

- [ ] **Step 4: Re-run backend tests**

Run: `go test ./internal/ai/runtime ./internal/ai/... ./internal/service/ai/...`
Expected: PASS for updated runtime tests and failures only where downstream code still expects removed events.

- [ ] **Step 5: Commit the backend protocol cleanup**

```bash
git add internal/ai/events/events.go internal/ai/runtime/sse_converter.go internal/ai/runtime/runtime.go internal/ai/orchestrator.go internal/ai/runtime/sse_converter_test.go
git commit -m "refactor: remove legacy AI chat streaming events"
```

### Task 3: Delete detached approval and legacy route semantics

**Files:**
- Modify: `internal/service/ai/routes.go`
- Modify: `internal/service/ai/tooling_handlers.go`
- Modify: `internal/service/ai/handler.go`
- Test: `internal/service/ai/routes_test.go`

- [ ] **Step 1: Write a failing route test for the new approval endpoint**

```go
func TestRegisterAIHandlers_RegistersChainApprovalDecisionRoute(t *testing.T) {
	routes := collectAIRoutesForTest()
	require.Contains(t, routes, "POST /api/v1/ai/chains/:chain_id/approvals/:node_id/decision")
}
```

- [ ] **Step 2: Run the route test to verify it fails**

Run: `go test ./internal/service/ai -run TestRegisterAIHandlers_RegistersChainApprovalDecisionRoute -v`
Expected: FAIL because only `/ai/approvals/:id/approve` and `/ai/approval/respond` style routes exist.

- [ ] **Step 3: Add the unified route and remove old primary-path approval handlers from the chat path**

```go
g.POST("/chains/:chain_id/approvals/:node_id/decision", h.DecideChainApproval)
```

- [ ] **Step 4: Refactor handler request types around chain identity**

```go
type chainApprovalDecisionRequest struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}
```

- [ ] **Step 5: Commit the route and handler cleanup**

```bash
git add internal/service/ai/routes.go internal/service/ai/tooling_handlers.go internal/service/ai/handler.go internal/service/ai/routes_test.go
git commit -m "refactor: unify AI chain approval decision flow"
```

## Chunk 2: Introduce the Canonical ThoughtChain Contract

### Task 4: Canonicalize backend and frontend contract types

**Files:**
- Modify: `internal/ai/contracts.go`
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/types.ts`
- Test: `web/src/api/modules/ai.test.ts`

- [ ] **Step 1: Write a failing parser test for canonical chain events**

```ts
it('parses canonical node events and rejects detached approval_required events', async () => {
  const seen: string[] = [];
  await aiApi.streamFromResponse(mockSSE([
    'event: node_open\ndata: {"chain_id":"chain-1","node_id":"node-1","kind":"plan"}\n\n',
  ]), {
    onEvent: (event) => seen.push(event.type),
  });
  expect(seen).toContain('node_open');
  expect(seen).not.toContain('approval_required');
});
```

- [ ] **Step 2: Run the parser test**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts`
Expected: FAIL because the parser still routes legacy event families and old DTO shapes.

- [ ] **Step 3: Replace DTOs with canonical chain and replay shapes**

```ts
export interface AIChainNode {
  chain_id: string;
  node_id: string;
  parent_node_id?: string;
  kind: 'plan' | 'step' | 'tool' | 'approval' | 'replan' | 'answer' | 'status';
  status: 'pending' | 'running' | 'waiting' | 'success' | 'error' | 'aborted';
  title?: string;
  summary?: string;
  payload?: Record<string, unknown>;
}
```

- [ ] **Step 4: Update event parsing and approval API helpers**

Run search: `rg -n "respondApproval|approveApproval|confirmApproval|phase_started|approval_required" web/src/api/modules/ai.ts`
Expected: replaced by canonical `decideChainApproval` and `node_*` parsing.

- [ ] **Step 5: Commit the contract cleanup**

```bash
git add internal/ai/contracts.go web/src/api/modules/ai.ts web/src/components/AI/types.ts web/src/api/modules/ai.test.ts
git commit -m "refactor: define canonical AI thoughtchain contracts"
```

### Task 5: Persist and restore canonical thoughtChain replay

**Files:**
- Modify: `internal/service/ai/session_recorder.go`
- Modify: `internal/ai/state/chat_store.go`
- Modify: `web/src/components/AI/hooks/useConversationRestore.ts`
- Modify: `web/src/components/AI/hooks/useConversationRestore.test.ts`

- [ ] **Step 1: Write a failing restore test using only chain replay data**

```tsx
it('restores assistant runtime from canonical thoughtChain replay only', async () => {
  const restored = restoreConversation({
    turns: [{
      id: 'turn-1',
      role: 'assistant',
      chain: { chain_id: 'chain-1', nodes: [{ node_id: 'plan-1', kind: 'plan', status: 'success' }] },
    }],
  });
  expect(restored.messages[0].runtime?.nodes[0].kind).toBe('plan');
});
```

- [ ] **Step 2: Run the restore test**

Run: `npm run test:run -- web/src/components/AI/hooks/useConversationRestore.test.ts`
Expected: FAIL because restore still reconstructs from legacy turn/block and message compatibility fields.

- [ ] **Step 3: Rewrite session recorder to persist chain nodes and answer separately**

```go
type StoredThoughtChain struct {
	ChainID string           `json:"chain_id"`
	Nodes   []StoredChainNode `json:"nodes"`
}
```

- [ ] **Step 4: Rewrite restore logic to consume canonical replay first and delete legacy merge rules**

Run search: `rg -n "legacy_thought_chain|turns\\?|summaryStage|normalizeTurnBlocks" internal/ai/state/chat_store.go web/src/components/AI/hooks/useConversationRestore.ts`
Expected: dead compatibility branches removed or isolated behind short-lived comments slated for deletion.

- [ ] **Step 5: Commit replay persistence changes**

```bash
git add internal/service/ai/session_recorder.go internal/ai/state/chat_store.go web/src/components/AI/hooks/useConversationRestore.ts web/src/components/AI/hooks/useConversationRestore.test.ts
git commit -m "refactor: persist canonical AI thoughtchain replay"
```

## Chunk 3: Rebuild the Frontend Runtime and Approval UX

### Task 6: Replace mixed frontend runtime state with one chain reducer

**Files:**
- Modify: `web/src/components/AI/thoughtChainRuntime.ts`
- Modify: `web/src/components/AI/Copilot.tsx`
- Modify: `web/src/components/AI/turnLifecycle.ts`
- Modify: `web/src/components/AI/thoughtChainRuntime.test.ts`
- Modify: `web/src/components/AI/Copilot.test.tsx`

- [ ] **Step 1: Write a failing reducer test for pause/resume and final answer separation**

```ts
it('keeps approval waiting in-chain and appends final answer only after chain completion', () => {
  let state = createThoughtChainRuntimeState();
  state = reduceThoughtChainRuntimeEvent(state, { type: 'node_open', data: { chain_id: 'c1', node_id: 'a1', kind: 'approval', status: 'waiting' } });
  state = reduceThoughtChainRuntimeEvent(state, { type: 'chain_completed', data: { chain_id: 'c1' } });
  expect(state.nodes[0].status).toBe('waiting');
  expect(state.finalAnswer.visible).toBe(false);
});
```

- [ ] **Step 2: Run reducer and Copilot tests**

Run: `npm run test:run -- web/src/components/AI/thoughtChainRuntime.test.ts web/src/components/AI/Copilot.test.tsx`
Expected: FAIL because `Copilot.tsx` still merges `thoughtChain`, `turnLifecycle`, confirmation, and raw content paths.

- [ ] **Step 3: Simplify `Copilot.tsx` to one runtime source**

```tsx
const [runtime, dispatchRuntime] = useReducer(reduceThoughtChainRuntimeEvent, undefined, createThoughtChainRuntimeState);
const assistantView = projectAssistantView(runtime, restoredReplay);
```

- [ ] **Step 4: Delete or isolate old lifecycle helpers**

Run search: `rg -n "applyPhaseStarted|applyPlanGenerated|applyStepStarted|applyStepComplete|applyReplanTriggered|upsertThoughtStage|finalizeThoughtStage" web/src/components/AI`
Expected: removed from the primary chat render path.

- [ ] **Step 5: Commit the frontend state rewrite**

```bash
git add web/src/components/AI/thoughtChainRuntime.ts web/src/components/AI/Copilot.tsx web/src/components/AI/turnLifecycle.ts web/src/components/AI/thoughtChainRuntime.test.ts web/src/components/AI/Copilot.test.tsx
git commit -m "refactor: drive AI chat from one thoughtchain runtime"
```

### Task 7: Upgrade the thoughtChain UI and fix the recommended-prompt race

**Files:**
- Modify: `web/src/components/AI/components/RuntimeThoughtChain.tsx`
- Modify: `web/src/components/AI/components/RuntimeChain.css`
- Modify: `web/src/components/AI/AISurfaceBoundary.tsx`
- Modify: `web/src/components/AI/AIAssistantDrawer.test.tsx`
- Modify: `web/src/components/AI/AICopilotButton.test.tsx`
- Modify: `web/src/components/AI/Copilot.test.tsx`

- [ ] **Step 1: Write a failing UI test for fresh-session recommended prompt submission**

```tsx
it('creates a placeholder assistant chain immediately when a recommended prompt is clicked', async () => {
  render(<Copilot />);
  await userEvent.click(screen.getByRole('button', { name: /推荐/i }));
  expect(screen.queryByText('AI 助手暂时不可用')).not.toBeInTheDocument();
  expect(screen.getByText(/正在连接|正在思考/)).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the UI test**

Run: `npm run test:run -- web/src/components/AI/Copilot.test.tsx web/src/components/AI/AIAssistantDrawer.test.tsx`
Expected: FAIL because the assistant container is not created until streaming state settles.

- [ ] **Step 3: Implement immediate placeholder chain creation and upgraded node rendering**

```tsx
setMessages((current) => current.concat([
  createUserMessage(prompt),
  createAssistantPlaceholder({ chainId: `pending:${Date.now()}`, status: 'connecting' }),
]));
```

- [ ] **Step 4: Style dedicated node cards for plan/tool/approval/replan/answer**

Run visual check: `npm run preview`
Expected: approval is visually emphasized, replan reason is readable, and raw JSON is never shown as assistant prose.

- [ ] **Step 5: Commit the UI and race-condition fix**

```bash
git add web/src/components/AI/components/RuntimeThoughtChain.tsx web/src/components/AI/components/RuntimeChain.css web/src/components/AI/AISurfaceBoundary.tsx web/src/components/AI/AIAssistantDrawer.test.tsx web/src/components/AI/AICopilotButton.test.tsx web/src/components/AI/Copilot.test.tsx
git commit -m "feat: upgrade AI thoughtchain UI and prompt handling"
```

## Chunk 4: Observability, Verification, and Cleanup

### Task 8: Add callback-based thoughtChain metrics and traces

**Files:**
- Modify: `internal/ai/observability/metrics.go`
- Modify: `internal/service/ai/execution_observability.go`
- Create: `internal/ai/observability/thoughtchain.go`
- Create or modify: `internal/ai/observability/metrics_test.go`

- [ ] **Step 1: Write a failing metric test for approval wait and chain completion**

```go
func TestObserveThoughtChainApprovalWait(t *testing.T) {
	metrics := newMetrics()
	metrics.ObserveThoughtChainNode(NodeRecord{
		ChainID: "chain-1",
		NodeID: "approval-1",
		NodeKind: "approval",
		Status: "waiting",
		Duration: 3 * time.Second,
	})
	// assert metric family contains approval wait sample
}
```

- [ ] **Step 2: Run the metric test**

Run: `go test ./internal/ai/observability -run TestObserveThoughtChainApprovalWait -v`
Expected: FAIL because no thoughtChain-specific counters/histograms exist yet.

- [ ] **Step 3: Implement focused thoughtChain metric helpers**

```go
type NodeRecord struct {
	ChainID string
	NodeID string
	NodeKind string
	Scene string
	Tool string
	Status string
	Duration time.Duration
}
```

- [ ] **Step 4: Wire callbacks from the AI service/runtime**

Run search: `rg -n "ObserveAgentExecution|ObserveToolExecution|execution_observability" internal/ai internal/service/ai`
Expected: updated call sites now include chain/node callbacks and trace metadata propagation.

- [ ] **Step 5: Commit observability support**

```bash
git add internal/ai/observability/metrics.go internal/ai/observability/thoughtchain.go internal/service/ai/execution_observability.go internal/ai/observability/metrics_test.go
git commit -m "feat: add AI thoughtchain observability metrics"
```

### Task 9: Full verification, spec validation, and dead-code sweep

**Files:**
- Modify: any touched files that still contain temporary migration shims
- Validate: `openspec/changes/ai-thoughtchain-runtime-redesign/*`

- [ ] **Step 1: Run backend tests for AI runtime and service paths**

Run: `go test ./internal/ai/... ./internal/service/ai/...`
Expected: PASS.

- [ ] **Step 2: Run frontend tests for AI chat runtime**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/components/AI/thoughtChainRuntime.test.ts web/src/components/AI/hooks/useConversationRestore.test.ts web/src/components/AI/Copilot.test.tsx web/src/components/AI/AIAssistantDrawer.test.tsx web/src/components/AI/AICopilotButton.test.tsx`
Expected: PASS.

- [ ] **Step 3: Validate OpenSpec artifacts**

Run: `openspec validate --json`
Expected: valid change with no schema errors.

- [ ] **Step 4: Sweep for banned legacy primary-path concepts**

Run: `rg -n "phase_started|phase_complete|plan_generated|step_started|step_complete|replan_triggered|approval_required|respondApproval|confirmApproval|approveApproval|turn_started|block_open|block_delta|block_replace|block_close" internal/ai internal/service/ai web/src/components/AI web/src/api/modules/ai.ts`
Expected: no remaining primary-path references except intentionally isolated compatibility comments or deleted tests updated in this change.

- [ ] **Step 5: Commit final cleanup**

```bash
git add -A
git commit -m "refactor: complete AI thoughtchain runtime redesign"
```

## Implementation Notes

- Follow `@test-driven-development`: write the failing test first for each task, then make the minimal change to pass it.
- Follow `@systematic-debugging` if any migration slice breaks streaming, replay, or approval recovery in unexpected ways.
- Follow `@verification-before-completion`: do not claim a slice is done until the exact command in that slice passes.
- Keep commits small and aligned with the tasks above. If one task balloons, split it before coding.
