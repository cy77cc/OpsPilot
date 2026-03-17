# A2UI Stream Protocol Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current mixed AI streaming contract with the A2UI SSE protocol end-to-end, and delete old stream compatibility paths instead of preserving them.

**Architecture:** Backend chat execution should emit only A2UI events and write them as standard SSE frames. Frontend stream parsing should consume only the A2UI event family, with UI state driven by structured `meta / agent_handoff / plan / replan / delta / tool_* / done / error` payloads instead of legacy `init / intent / status / progress / report_ready` branches or JSON-in-delta cleanup.

**Tech Stack:** Go, Gin, ADK/eino event iterator, Vitest, React, Ant Design X SDK, TypeScript

---

## Scope Decisions

- No protocol versioning.
- No backward compatibility shim.
- No rollout flag or dual-read/dual-write path.
- Old SSE event names, old frontend handlers, and delta JSON normalization should be removed in the same branch.

## File Map

### Backend

- Modify: `internal/ai/events/events.go`
  Defines canonical public/internal event names. This file should reflect only the A2UI contract plus any explicitly internal-only events.
- Modify: `internal/ai/runtime/stream.go`
  Replace JSON wrapper helper behavior with focused SSE encoding helpers and a public-event allowlist that matches A2UI.
- Modify: `internal/ai/runtime/stream_test.go`
  Locks the event allowlist and SSE framing behavior.
- Create: `internal/service/ai/logic/a2ui_mapper.go`
  Holds ADK-event-to-A2UI projection helpers so `logic.go` stops mixing persistence, orchestration, and protocol translation.
- Create: `internal/service/ai/logic/a2ui_mapper_test.go`
  Covers planner/replanner parsing, agent handoff, tool activity, approvals, final completion, and error mapping.
- Modify: `internal/service/ai/logic/logic.go`
  Emit only A2UI events via mapper helpers; remove legacy stream emissions.
- Create: `internal/service/ai/handler/sse_writer.go`
  Small SSE writer for `event: ...`, `data: ...`, and heartbeat comments.
- Create: `internal/service/ai/handler/sse_writer_test.go`
  Tests standard SSE framing and heartbeat output.
- Modify: `internal/service/ai/handler/chat.go`
  Use the dedicated SSE writer instead of hand-writing frames inline.
- Modify: `internal/service/ai/handler/chat_test.go`
  Expect `meta` as first event and A2UI-only stream behavior.
- Modify: `internal/service/ai/handler/run.go`
  Trim run payload to fields still used after A2UI migration.
- Modify: `internal/service/ai/handler/run_test.go`
  Locks the post-cleanup run payload.

### Frontend

- Modify: `web/src/api/modules/ai.ts`
  Replace legacy stream event typings and dispatch branches with A2UI-native types and handlers.
- Modify: `web/src/api/modules/ai.test.ts`
  Validates the fetch stream consumer against A2UI events only.
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
  Deletes `normalizeVisibleStreamChunk` expectations and replaces them with strict A2UI delta behavior.
- Create: `web/src/components/AI/a2uiState.ts`
  Focused reducer/helpers for plan list, tool activity, approval state, and streamed assistant text.
- Create: `web/src/components/AI/__tests__/a2uiState.test.ts`
  Covers state transitions from `meta` through `done/error`.
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
  Build placeholder/status behavior from `meta`, `agent_handoff`, and `plan`, and stream visible text directly from A2UI `delta`.
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
  Update provider expectations to the new event family.
- Modify: `web/src/components/AI/CopilotSurface.tsx`
  Render plan progress, tool activity, and approval state from A2UI reducer output instead of relying on old intent/status placeholders.
- Modify: `web/src/components/AI/types.ts`
  Add any typed view-models needed by the reducer/provider pair and remove stale legacy-only fields if unused.

## Chunk 1: Backend A2UI Event Surface

### Task 1: Lock the canonical A2UI event list and real SSE framing

**Files:**
- Modify: `internal/ai/events/events.go`
- Modify: `internal/ai/runtime/stream.go`
- Modify: `internal/ai/runtime/stream_test.go`
- Create: `internal/service/ai/handler/sse_writer.go`
- Create: `internal/service/ai/handler/sse_writer_test.go`
- Modify: `internal/service/ai/handler/chat.go`
- Modify: `internal/service/ai/handler/chat_test.go`

- [ ] **Step 1: Write failing backend tests for A2UI-only framing**

Add/replace tests so they assert:

```go
func TestEncodePublicEvent_RejectsLegacyEvents(t *testing.T) {
	for _, name := range []string{"init", "intent", "status", "progress", "report_ready", "heartbeat"} {
		if _, err := EncodePublicEvent(name, map[string]any{"ok": true}); err == nil {
			t.Fatalf("expected legacy event %q to be rejected", name)
		}
	}
}

func TestSSEWriter_WriteEvent(t *testing.T) {
	var buf bytes.Buffer
	w := NewSSEWriter(&buf)
	require.NoError(t, w.WriteEvent("meta", map[string]any{"session_id": "sess-1", "run_id": "run-1", "turn": 1}))
	require.Equal(t, "event: meta\ndata: {\"session_id\":\"sess-1\",\"run_id\":\"run-1\",\"turn\":1}\n\n", buf.String())
}
```

- [ ] **Step 2: Run backend tests and confirm they fail for the current implementation**

Run: `go test ./internal/ai/runtime ./internal/service/ai/handler -run 'TestEncodePublicEvent|TestSSEWriter|TestChatHandler'`

Expected: FAIL because `chat_test.go` still expects `init`, and there is no dedicated SSE writer/heartbeat helper yet.

- [ ] **Step 3: Implement canonical event constants and SSE writer**

Make these changes:

```go
var publicEventNames = map[string]struct{}{
	"meta": {},
	"agent_handoff": {},
	"plan": {},
	"replan": {},
	"delta": {},
	"tool_call": {},
	"tool_result": {},
	"tool_approval": {},
	"done": {},
	"error": {},
}
```

```go
type SSEWriter struct {
	w io.Writer
}

func (w *SSEWriter) WriteEvent(event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w.w, "event: %s\ndata: %s\n\n", event, data)
	return err
}

func (w *SSEWriter) WritePing() error {
	_, err := io.WriteString(w.w, ": ping\n\n")
	return err
}
```

Update `chat.go` to call the writer instead of manually writing bytes.

- [ ] **Step 4: Re-run backend tests until the A2UI framing contract passes**

Run: `go test ./internal/ai/runtime ./internal/service/ai/handler -run 'TestEncodePublicEvent|TestSSEWriter|TestChatHandler'`

Expected: PASS with first stream event now `meta`, no legacy public events accepted, and SSE frames encoded in standard format.

- [ ] **Step 5: Commit the event-surface cleanup**

```bash
git add internal/ai/events/events.go internal/ai/runtime/stream.go internal/ai/runtime/stream_test.go internal/service/ai/handler/sse_writer.go internal/service/ai/handler/sse_writer_test.go internal/service/ai/handler/chat.go internal/service/ai/handler/chat_test.go
git commit -m "refactor: replace legacy ai stream framing with a2ui sse surface"
```

### Task 2: Project ADK runtime output into A2UI events only

**Files:**
- Create: `internal/service/ai/logic/a2ui_mapper.go`
- Create: `internal/service/ai/logic/a2ui_mapper_test.go`
- Modify: `internal/service/ai/logic/logic.go`

- [ ] **Step 1: Write failing mapper tests before touching orchestration code**

Add focused tests for:

```go
func TestMapPlannerMessageToPlan(t *testing.T) {
	got, ok := mapAssistantEnvelope("planner", `{"steps":["inspect pods","check quota"]}`, 0)
	require.True(t, ok)
	require.Equal(t, "plan", got.Event)
}

func TestMapReplannerResponseToFinalReplanAndDelta(t *testing.T) {
	events := mapAssistantEnvelopeSequence("replanner", `{"response":"quota exhausted"}`, 2)
	require.Equal(t, []string{"replan", "delta"}, collectEventNames(events))
}

func TestMapTransferToAgentToAgentHandoff(t *testing.T) {
	got := mapTransfer("OpsPilotAgent", "DiagnosisAgent")
	require.Equal(t, "agent_handoff", got.Event)
	require.Equal(t, "diagnosis", got.Data["intent"])
}
```

- [ ] **Step 2: Run the mapper/logic tests and verify failure**

Run: `go test ./internal/service/ai/logic -run 'TestMap|TestLogic'`

Expected: FAIL because no A2UI mapper exists and `logic.Chat()` still emits `init`, `intent`, and `status`.

- [ ] **Step 3: Implement mapper helpers and switch `logic.Chat()` to them**

Target behavior:

```go
emit("meta", map[string]any{
	"session_id": sessionID,
	"run_id": run.ID,
	"turn": turnNumber,
})
```

```go
if handoff := mapTransferEvent(event); handoff != nil {
	emit(handoff.Event, handoff.Data)
}

for _, projected := range ProjectA2UIEvents(event, state) {
	emit(projected.Event, projected.Data)
}
```

Delete these legacy emissions from `logic.go`:

```go
emit("init", ...)
emit("intent", ...)
emit("status", ...)
emit("progress", ...)
emit("report_ready", ...)
```

Also remove delta-side JSON envelope cleanup assumptions from backend emission. Planner/replanner JSON must become `plan`/`replan` events, not textual delta.

- [ ] **Step 4: Re-run backend logic tests**

Run: `go test ./internal/service/ai/logic ./internal/service/ai/handler -run 'TestMap|TestLogic|TestChatHandler'`

Expected: PASS with chat streams emitting only A2UI events in the right order.

- [ ] **Step 5: Commit the mapper rewrite**

```bash
git add internal/service/ai/logic/a2ui_mapper.go internal/service/ai/logic/a2ui_mapper_test.go internal/service/ai/logic/logic.go
git commit -m "refactor: project ai runtime events into a2ui protocol"
```

## Chunk 2: Frontend A2UI Stream Consumer

### Task 3: Replace the API stream parser with A2UI-native types and dispatch

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.test.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`

- [ ] **Step 1: Write failing frontend parser tests for the A2UI event family**

Update tests to assert only the new contract:

```ts
it('consumes a2ui stream events', async () => {
  const body = buildStream([
    'event: meta\\ndata: {"session_id":"sess-1","run_id":"run-1","turn":1}\\n\\n',
    'event: agent_handoff\\ndata: {"from":"OpsPilotAgent","to":"DiagnosisAgent","intent":"diagnosis"}\\n\\n',
    'event: plan\\ndata: {"steps":["inspect pods"],"iteration":0}\\n\\n',
    'event: delta\\ndata: {"content":"checking cluster"}\\n\\n',
    'event: done\\ndata: {"run_id":"run-1","status":"completed","iterations":1}\\n\\n',
  ]);
})
```

Delete tests for:

- `init`
- `intent`
- `status`
- `progress`
- `report_ready`
- `normalizeVisibleStreamChunk`

- [ ] **Step 2: Run frontend API tests and confirm the legacy parser breaks**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts`

Expected: FAIL because `AIChatStreamHandlers` and `dispatchAIStreamEvent()` still expect legacy branches and delta normalization.

- [ ] **Step 3: Rewrite `web/src/api/modules/ai.ts` to be A2UI-native**

Implement a smaller event model:

```ts
export interface A2UIStreamHandlers {
  onMeta?: (payload: A2UIMeta) => void;
  onAgentHandoff?: (payload: A2UIAgentHandoff) => void;
  onPlan?: (payload: A2UIPlan) => void;
  onReplan?: (payload: A2UIReplan) => void;
  onDelta?: (payload: A2UIDelta) => void;
  onToolCall?: (payload: A2UIToolCall) => void;
  onToolApproval?: (payload: A2UIToolApproval) => void;
  onToolResult?: (payload: A2UIToolResult) => void;
  onDone?: (payload: A2UIDone) => void;
  onError?: (payload: A2UIError) => void;
}
```

And dispatch with direct field names:

```ts
if (eventType === 'delta') {
  handlers.onDelta?.(payload as A2UIDelta);
}
```

Delete:

- `normalizeVisibleStreamChunk`
- `toContentChunk`
- all `onInit/onIntent/onStatus/onProgress/onReportReady/onThinkingDelta/onHeartbeat` branches
- fallback logic that decodes planner/replanner JSON from delta text

- [ ] **Step 4: Re-run frontend API tests**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts`

Expected: PASS with only A2UI events accepted.

- [ ] **Step 5: Commit the parser cleanup**

```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts
git commit -m "refactor: switch frontend ai stream parser to a2ui"
```

### Task 4: Drive provider/UI state from structured A2UI events

**Files:**
- Create: `web/src/components/AI/a2uiState.ts`
- Create: `web/src/components/AI/__tests__/a2uiState.test.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/types.ts`

- [ ] **Step 1: Write failing reducer/provider tests**

Add reducer coverage such as:

```ts
it('marks completed steps when replan arrives', () => {
  let state = reduceA2UI(initialState, { type: 'plan', payload: { steps: ['a', 'b'], iteration: 0 } });
  state = reduceA2UI(state, { type: 'replan', payload: { steps: ['b'], completed: 1, iteration: 1, is_final: false } });
  expect(state.plan.items[0].status).toBe('done');
  expect(state.plan.items[1].status).toBe('active');
});
```

Update provider tests to assert:

- placeholder starts at `[准备中]` after `meta`
- agent handoff changes status copy
- `plan` can show planning status
- visible text streams from `delta.content`
- no waiting on `intent`

- [ ] **Step 2: Run the UI tests and confirm failure**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/a2uiState.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx`

Expected: FAIL because the provider still keys off legacy `onInit/onIntent/onStatus`, and there is no A2UI reducer yet.

- [ ] **Step 3: Implement reducer-driven A2UI UI state**

Suggested reducer shape:

```ts
export interface A2UIState {
  sessionId?: string;
  runId?: string;
  turn?: number;
  handoff?: { from: string; to: string; intent: string };
  plan: { items: Array<{ content: string; status: 'pending' | 'active' | 'done' }> };
  tools: Array<{ callId: string; toolName: string; status: 'running' | 'waiting_approval' | 'done'; content?: string }>;
  approval?: A2UIToolApproval;
  content: string;
  done: boolean;
  error?: string;
}
```

Provider migration rules:

```ts
onMeta -> placeholder "[准备中]"
onAgentHandoff -> placeholder "[诊断助手开始处理]" / "[变更助手开始处理]"
onPlan -> placeholder "[正在规划处理方式]"
onDelta -> append payload.content directly
onError -> terminal error
```

`CopilotSurface.tsx` should read reducer output and render:

- current streamed assistant markdown
- plan checklist
- tool activity cards
- approval waiting card/modal state

Do not reintroduce compatibility parsing in UI code.

- [ ] **Step 4: Re-run the UI tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/a2uiState.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx`

Expected: PASS with provider and surface driven by A2UI state only.

- [ ] **Step 5: Commit the UI migration**

```bash
git add web/src/components/AI/a2uiState.ts web/src/components/AI/__tests__/a2uiState.test.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/CopilotSurface.tsx web/src/components/AI/types.ts
git commit -m "feat: render ai copilot state from a2ui stream events"
```

## Chunk 3: End-to-End Cleanup

### Task 5: Remove residual legacy payloads and verify the integrated flow

**Files:**
- Modify: `internal/service/ai/handler/run.go`
- Modify: `internal/service/ai/handler/run_test.go`
- Modify: `internal/service/ai/handler/chat_test.go`
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/CopilotSurface.tsx`

- [ ] **Step 1: Write failing cleanup assertions**

Add tests asserting:

- chat stream never emits `init`, `intent`, `status`, `progress`, or `report_ready`
- frontend ignores unknown events but does not rely on legacy ones
- `GET /ai/runs/:id` no longer carries fields that only existed for legacy stream placeholders if nothing reads them anymore

Example:

```go
for _, event := range events {
	if slices.Contains([]string{"init", "intent", "status", "progress", "report_ready"}, event.Event) {
		t.Fatalf("unexpected legacy event %q", event.Event)
	}
}
```

- [ ] **Step 2: Run the full focused verification suite and capture any remaining legacy assumptions**

Run: `go test ./internal/ai/runtime ./internal/service/ai/logic ./internal/service/ai/handler`

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/a2uiState.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx`

Expected: FAIL if any old event names or old placeholder fields still leak through.

- [ ] **Step 3: Remove the remaining dead paths**

Cleanup checklist:

- delete legacy event interfaces/types
- delete dead helper functions for delta envelope normalization
- delete old provider logic that waits for `intent`
- delete dead run payload fields not read anywhere after migration
- update comments/docstrings that still describe Phase 1 events

- [ ] **Step 4: Re-run the full focused verification suite**

Run: `go test ./internal/ai/runtime ./internal/service/ai/logic ./internal/service/ai/handler`

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/a2uiState.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx`

Expected: PASS with zero legacy event expectations left in backend or frontend.

- [ ] **Step 5: Commit the final cleanup**

```bash
git add internal/service/ai/handler/run.go internal/service/ai/handler/run_test.go internal/service/ai/handler/chat_test.go web/src/api/modules/ai.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/CopilotSurface.tsx
git commit -m "chore: remove legacy ai stream compatibility paths"
```

## Verification Notes

- Prefer focused tests while iterating on each task; run the full focused suite at the end of Chunk 3.
- If mapper logic touches real ADK event shapes that are hard to build in unit tests, capture the minimum event subset in table-driven fixtures inside `a2ui_mapper_test.go`.
- Keep commits small and task-scoped. Do not bundle backend and frontend parser rewrites into one giant commit.
- If a test currently asserts legacy event names, update the assertion instead of layering compatibility code on top.

## Risks And Guardrails

- Planner/replanner payloads are currently leaking through text paths. Fix the source mapping first; do not add more frontend parsing heuristics.
- The current provider UX is tied to `intent` and `status`. Moving to A2UI will break placeholders unless reducer/provider tests land first.
- `stream.go` currently does not encode literal SSE frames. Correct this at the transport boundary instead of spreading `fmt.Fprintf` calls across handlers.
- Keep `thinking_delta` internal unless product requirements explicitly change. This migration should not accidentally expose chain-of-thought.

Plan complete and saved to `docs/superpowers/plans/2026-03-17-a2ui-stream-protocol.md`. Ready to execute?
