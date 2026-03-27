# AI Stream Projector Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current ad hoc AI stream projection logic with a two-stage backend projector that normalizes `adk.AgentEvent` values and emits a stable public SSE protocol including approval-aware `run_state` events.

**Architecture:** Keep the external SSE event family compatible, but split the backend path into `AgentEvent -> NormalizedEvent -> PublicStreamEvent`. Move runtime interpretation out of `internal/service/ai/logic/logic.go` and into `internal/ai/runtime`, with focused files for normalization, projection, event definitions, and planner envelope parsing.

**Tech Stack:** Go 1.26, CloudWeGo Eino ADK, standard library JSON/SSE framing, `testing`, existing `go test` package tests.

**Execution Status:** Completed on 2026-03-17. The implementation landed on `main`, the feature branch was merged and removed, and repository-wide verification passed with `go test ./...`.

---

## File Structure

### Runtime projection files

- Create: `internal/ai/runtime/events.go`
  - public event structs
  - event envelope types
  - helper constructors for `meta`, `done`, `error`, `run_state`
- Create: `internal/ai/runtime/normalize.go`
  - normalized runtime event model
  - `NormalizeAgentEvent` entrypoint
- Create: `internal/ai/runtime/project.go`
  - `NormalizedEvent -> PublicStreamEvent`
  - planner/replanner projection logic
  - approval projection logic
- Create: `internal/ai/runtime/projector.go`
  - `StreamProjector`
  - `ProjectionState`
  - `Consume`, `Finish`, `Fail`
- Create: `internal/ai/runtime/plan_decode.go`
  - `decodeStepsEnvelope`
  - `decodeResponseEnvelope`
  - tool-argument decode helpers if still needed
- Modify: `internal/ai/runtime/streamer.go`
  - reduce to compatibility helpers or remove duplicated logic after migration
- Create: `internal/ai/runtime/normalize_test.go`
  - unit tests for ADK event normalization
- Create: `internal/ai/runtime/project_test.go`
  - unit tests for normalized-to-public projection
- Create: `internal/ai/runtime/projector_test.go`
  - stateful projector tests including `run_state`

### Chat logic files

- Modify: `internal/service/ai/logic/logic.go`
  - instantiate projector
  - delegate event handling to projector
  - keep DB persistence and assistant-content accumulation
- Modify or Create: `internal/service/ai/logic/logic_test.go`
  - add chat-facing tests around projected `delta`, `error`, `done`, and approval forwarding

### SSE writer files

- Modify: `internal/service/ai/handler/sse_writer_test.go`
  - add coverage for the new public `run_state` event framing

### Optional cleanup

- Modify: `docs/superpowers/specs/2026-03-17-ai-stream-projector-design.md`
  - only if implementation changes a contract or file layout

## Chunk 1: Runtime Event Model And Normalization

### Task 1: Define public and normalized event types

**Files:**
- Create: `internal/ai/runtime/events.go`
- Create: `internal/ai/runtime/normalize.go`
- Test: `internal/ai/runtime/normalize_test.go`

- [ ] **Step 1: Write the failing normalized-event type tests**

```go
func TestNormalizeAgentEvent_TransferAction(t *testing.T) {
	event := &adk.AgentEvent{
		AgentName: "OpsPilotAgent",
		Action: adk.NewTransferToAgentAction("DiagnosisAgent"),
	}

	got := NormalizeAgentEvent(event)

	if len(got) != 1 || got[0].Kind != NormalizedKindHandoff {
		t.Fatalf("expected one handoff event, got %#v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/runtime -run TestNormalizeAgentEvent_TransferAction -v`
Expected: FAIL with undefined `NormalizeAgentEvent` or missing normalized types

- [ ] **Step 3: Write minimal normalized event types**

```go
type NormalizedKind string

const (
	NormalizedKindMessage   NormalizedKind = "message"
	NormalizedKindToolCall  NormalizedKind = "tool_call"
	NormalizedKindToolResult NormalizedKind = "tool_result"
	NormalizedKindHandoff   NormalizedKind = "handoff"
	NormalizedKindInterrupt NormalizedKind = "interrupt"
	NormalizedKindError     NormalizedKind = "error"
)
```

- [ ] **Step 4: Implement minimal `NormalizeAgentEvent` handoff support**

```go
func NormalizeAgentEvent(event *adk.AgentEvent) []NormalizedEvent {
	if event == nil {
		return nil
	}
	if event.Action != nil && event.Action.TransferToAgent != nil {
		return []NormalizedEvent{{
			Kind:      NormalizedKindHandoff,
			AgentName: event.AgentName,
			Handoff: &NormalizedHandoff{
				From: event.AgentName,
				To:   event.Action.TransferToAgent.DestAgentName,
			},
			Raw: event,
		}}
	}
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/ai/runtime -run TestNormalizeAgentEvent_TransferAction -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/runtime/events.go internal/ai/runtime/normalize.go internal/ai/runtime/normalize_test.go
git commit -m "feat: add runtime normalized event model"
```

### Task 2: Normalize message, tool, and interrupt events

**Files:**
- Modify: `internal/ai/runtime/normalize.go`
- Modify: `internal/ai/runtime/normalize_test.go`

- [ ] **Step 1: Write failing tests for assistant, tool, and interrupt normalization**

```go
func TestNormalizeAgentEvent_InterruptAction(t *testing.T) {
	event := &adk.AgentEvent{
		AgentName: "executor",
		Action: &adk.AgentAction{
			Interrupted: &adk.InterruptInfo{
				Data: map[string]any{
					"approval_id":     "ap-1",
					"call_id":         "call-1",
					"tool_name":       "restart_workload",
					"timeout_seconds": 300,
				},
			},
		},
	}

	got := NormalizeAgentEvent(event)
	if len(got) != 1 || got[0].Kind != NormalizedKindInterrupt {
		t.Fatalf("expected interrupt event, got %#v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ai/runtime -run 'TestNormalizeAgentEvent_(InterruptAction|AssistantMessage|ToolResult)' -v`
Expected: FAIL with missing interrupt or message normalization behavior

- [ ] **Step 3: Implement message normalization for assistant and tool outputs**

```go
func normalizeMessageOutput(event *adk.AgentEvent) []NormalizedEvent {
	// Handle assistant content, tool calls, and tool result messages.
}
```

- [ ] **Step 4: Implement interrupt normalization from `Action.Interrupted.Data`**

```go
func normalizeInterrupt(event *adk.AgentEvent) *NormalizedEvent {
	// Extract approval_id, call_id, tool_name, preview, timeout_seconds.
}
```

- [ ] **Step 5: Run package tests**

Run: `go test ./internal/ai/runtime -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/runtime/normalize.go internal/ai/runtime/normalize_test.go
git commit -m "feat: normalize runtime message and interrupt events"
```

## Chunk 2: Public Projection And Stateful Projector

### Task 3: Add public event types and planner envelope parsers

**Files:**
- Modify: `internal/ai/runtime/events.go`
- Create: `internal/ai/runtime/plan_decode.go`
- Create or Modify: `internal/ai/runtime/project_test.go`

- [ ] **Step 1: Write failing tests for planner and replanner payload decoding**

```go
func TestDecodeStepsEnvelope(t *testing.T) {
	steps, ok := decodeStepsEnvelope(`{"steps":["inspect pods","check events"]}`)
	if !ok || len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %#v ok=%v", steps, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/runtime -run 'TestDecode(StepsEnvelope|ResponseEnvelope)' -v`
Expected: FAIL with undefined decode helpers

- [ ] **Step 3: Implement event structs and helper constructors**

```go
type PublicStreamEvent struct {
	Event    string `json:"event"`
	Data     any    `json:"data"`
	Envelope *EventEnvelope `json:"-"`
}

func NewRunStateEvent(status string, payload map[string]any) PublicStreamEvent {
	return PublicStreamEvent{Event: "run_state", Data: map[string]any{"status": status}}
}
```

- [ ] **Step 4: Implement planner/replanner envelope parsers**

```go
func decodeStepsEnvelope(raw string) ([]string, bool) { /* json.Unmarshal */ }
func decodeResponseEnvelope(raw string) (string, bool) { /* json.Unmarshal */ }
```

- [ ] **Step 5: Run package tests**

Run: `go test ./internal/ai/runtime -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/runtime/events.go internal/ai/runtime/plan_decode.go internal/ai/runtime/project_test.go
git commit -m "feat: add public stream event helpers"
```

### Task 4: Project normalized events into public SSE events

**Files:**
- Create: `internal/ai/runtime/project.go`
- Modify: `internal/ai/runtime/project_test.go`

- [ ] **Step 1: Write failing projection tests for handoff, plan, replan, and approval**

```go
func TestProjectNormalizedEvent_ApprovalEmitsToolApprovalAndRunState(t *testing.T) {
	state := &ProjectionState{}
	event := NormalizedEvent{
		Kind:      NormalizedKindInterrupt,
		AgentName: "executor",
		Interrupt: &NormalizedInterrupt{
			ApprovalID: "ap-1",
			CallID:     "call-1",
			ToolName:   "restart_workload",
		},
	}

	got := projectNormalizedEvent(event, state)
	if len(got) != 2 {
		t.Fatalf("expected 2 projected events, got %#v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ai/runtime -run 'TestProjectNormalizedEvent_' -v`
Expected: FAIL with undefined projector helpers

- [ ] **Step 3: Implement minimal normalized-to-public projection**

```go
func projectNormalizedEvent(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
	switch event.Kind {
	case NormalizedKindHandoff:
		return []PublicStreamEvent{{Event: "agent_handoff", Data: ...}}
	case NormalizedKindInterrupt:
		return []PublicStreamEvent{
			{Event: "tool_approval", Data: ...},
			NewRunStateEvent("waiting_approval", map[string]any{"agent": event.AgentName}),
		}
	}
	return nil
}
```

- [ ] **Step 4: Add planner/replanner projection rules**

```go
// planner -> plan
// replanner with steps -> replan
// replanner with response -> replan + delta
```

- [ ] **Step 5: Run package tests**

Run: `go test ./internal/ai/runtime -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/runtime/project.go internal/ai/runtime/project_test.go
git commit -m "feat: project normalized events to public stream protocol"
```

### Task 5: Add the stateful `StreamProjector`

**Files:**
- Create: `internal/ai/runtime/projector.go`
- Create: `internal/ai/runtime/projector_test.go`

- [ ] **Step 1: Write the failing projector state tests**

```go
func TestStreamProjector_ConsumeTracksPlanAndReplanIterations(t *testing.T) {
	projector := NewStreamProjector()

	first := projector.Consume(&adk.AgentEvent{ /* planner message */ })
	second := projector.Consume(&adk.AgentEvent{ /* replanner response */ })

	if len(first) == 0 || len(second) == 0 {
		t.Fatalf("expected projected output from both events")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/ai/runtime -run 'TestStreamProjector_' -v`
Expected: FAIL with undefined `NewStreamProjector` or incorrect state handling

- [ ] **Step 3: Implement `ProjectionState` and `StreamProjector`**

```go
type ProjectionState struct {
	TotalPlanSteps  int
	ReplanIteration int
	RunPhase        string
}

type StreamProjector struct {
	state ProjectionState
}
```

- [ ] **Step 4: Implement `Consume`, `Finish`, and `Fail`**

```go
func (p *StreamProjector) Consume(event *adk.AgentEvent) []PublicStreamEvent {
	normalized := NormalizeAgentEvent(event)
	return projectNormalizedEvents(normalized, &p.state)
}
```

- [ ] **Step 5: Run package tests**

Run: `go test ./internal/ai/runtime -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/ai/runtime/projector.go internal/ai/runtime/projector_test.go
git commit -m "feat: add stateful ai stream projector"
```

## Chunk 3: Chat Logic Integration And Compatibility Cleanup

### Task 6: Refactor `Logic.Chat()` to use the projector

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write the failing logic test for projected delta accumulation**

```go
func TestChat_AccumulatesAssistantContentFromProjectedDelta(t *testing.T) {
	// Stub runtime events so projector emits delta events.
	// Assert final assistant message content equals concatenated projected delta content.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service/ai/logic -run 'TestChat_' -v`
Expected: FAIL because `Logic.Chat()` still uses direct runtime helper functions

- [ ] **Step 3: Replace direct helper calls with `StreamProjector` usage**

```go
projector := runtime.NewStreamProjector()
for {
	event, ok := iterator.Next()
	if !ok {
		break
	}
	projected := projector.Consume(event)
	for _, item := range projected {
		emit(item.Event, item.Data)
	}
}
```

- [ ] **Step 4: Preserve existing DB behavior while reading projected `delta` events**

```go
if item.Event == "delta" {
	assistantContent.WriteString(...)
}
```

- [ ] **Step 5: Use projector `Fail` and `Finish` helpers for terminal events**

```go
emit(projector.Fail(run.ID, event.Err).Event, projector.Fail(run.ID, event.Err).Data)
emit(projector.Finish(run.ID).Event, projector.Finish(run.ID).Data)
```

- [ ] **Step 6: Run logic and runtime tests**

Run: `go test ./internal/service/ai/logic ./internal/ai/runtime -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "refactor: route ai chat events through stream projector"
```

### Task 7: Keep SSE framing compatible and add `run_state` coverage

**Files:**
- Modify: `internal/service/ai/handler/sse_writer_test.go`
- Modify: `internal/ai/runtime/streamer.go`

- [ ] **Step 1: Write the failing SSE writer test for `run_state`**

```go
func TestSSEWriter_WriteRunStateEvent(t *testing.T) {
	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	err := writer.WriteEvent("run_state", map[string]any{"status": "planning"})
	if err != nil {
		t.Fatalf("write event: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify current framing still passes but new contract is covered**

Run: `go test ./internal/service/ai/handler -run 'TestSSEWriter_' -v`
Expected: FAIL only if helper expectations or public whitelist code still reject `run_state`

- [ ] **Step 3: Update compatibility helpers and whitelist**

```go
var publicEventNames = map[string]struct{}{
	"meta": {}, "agent_handoff": {}, "plan": {}, "replan": {},
	"delta": {}, "tool_call": {}, "tool_result": {}, "tool_approval": {},
	"run_state": {}, "done": {}, "error": {},
}
```

- [ ] **Step 4: Remove or slim duplicated projection helpers from `streamer.go`**

```go
// Keep only compatibility-level helpers that remain necessary after projector extraction.
```

- [ ] **Step 5: Run handler, runtime, and logic tests**

Run: `go test ./internal/service/ai/handler ./internal/ai/runtime ./internal/service/ai/logic -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/handler/sse_writer_test.go internal/ai/runtime/streamer.go
git commit -m "test: cover run state stream events"
```

## Chunk 4: Final Verification And Doc Sync

### Task 8: Run focused verification and sync docs if contracts changed

**Files:**
- Modify: `docs/superpowers/specs/2026-03-17-ai-stream-projector-design.md` (only if needed)

- [ ] **Step 1: Run the focused backend verification suite**

Run: `go test ./internal/ai/runtime ./internal/service/ai/logic ./internal/service/ai/handler -v`
Expected: PASS

- [ ] **Step 2: Run a repo-wide targeted search for removed helper references**

Run: `rg -n 'projectAgentEvent|projectAssistantMessage|projectAgentHandoff|projectApprovalEvent|doneEvent|errorEvent' internal`
Expected: only intended compatibility references remain

- [ ] **Step 3: Update the design spec if file layout or protocol details changed during implementation**

```md
- add note if `streamer.go` remains as a thin compatibility wrapper
- update `run_state` payload shape if implementation differs from the original spec
```

- [ ] **Step 4: Re-run the focused verification suite after doc sync**

Run: `go test ./internal/ai/runtime ./internal/service/ai/logic ./internal/service/ai/handler -v`
Expected: PASS

- [ ] **Step 5: Commit final cleanup**

```bash
git add docs/superpowers/specs/2026-03-17-ai-stream-projector-design.md internal/ai/runtime internal/service/ai/logic internal/service/ai/handler
git commit -m "refactor: finalize ai stream projector integration"
```

## Notes For The Implementer

- Follow `@test-driven-development` discipline even if the current package has limited test coverage.
- Keep `Logic.Chat()` focused on orchestration and persistence. Do not move DB writes into the projector.
- Do not expose raw `adk.AgentEvent` structure to the frontend contract.
- Keep planner/replanner name-based parsing isolated to one projection file. Do not spread it back into `logic.go`.
- Add `run_state` conservatively. The first implementation only needs the states described in the spec.
- If `streamer.go` cannot be removed cleanly in one pass, leave it as a thin compatibility facade and document that in the spec update instead of forcing a large refactor.
