# AI Big Bang Rewrite Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the entire AI module (DB schema + backend runtime + API contract + frontend runtime) in one cutover with no compatibility layer.

**Architecture:** Introduce a single event-sourced runtime engine (`engine`) as the only execution path for chat and approval resume, with strict transactional ordering (`event -> state -> projection cache`). Replace fragmented frontend runtime with one timeline reducer store and make UI consume only normalized `meta/timeline_delta/state/terminal` events.

**Tech Stack:** Go (Gin, GORM, ADK), MySQL/SQLite migrations, TypeScript + React + Vitest.

---

## Scope Check

This rewrite touches multiple subsystems (DB, backend runtime, HTTP contract, frontend state), but they are tightly coupled around the same run lifecycle contract. Splitting into separate plans would create temporary incompatible interfaces and guaranteed rework. Keep this as one coordinated plan with hard checkpoints.

---

## File Structure (Target)

### Backend Runtime Core

- Create: `internal/service/ai/engine/engine.go`
- Create: `internal/service/ai/engine/types.go`
- Create: `internal/service/ai/engine/state_machine.go`
- Create: `internal/service/ai/engine/event_sink.go`
- Create: `internal/service/ai/engine/stream_consumer.go`
- Create: `internal/service/ai/engine/timeline.go`
- Create: `internal/service/ai/engine/terminalizer.go`
- Test: `internal/service/ai/engine/*_test.go`

### Backend Integration Layer

- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/approval_expirer.go`
- Modify: `internal/service/ai/chat/service.go`
- Modify: `internal/service/ai/approval/service.go`
- Modify: `internal/service/ai/chat/handler.go`
- Modify: `internal/service/ai/approval/handler.go`
- Modify: `internal/service/ai/routes.go`
- Delete: deprecated resume-only paths in `internal/service/ai/logic/logic.go` (legacy `ResumeApproval` flow)

### Storage & Schema

- Modify: `internal/model/ai.go`
- Modify: `internal/dao/ai/run_event_dao.go`
- Modify: `internal/dao/ai/run_dao.go`
- Modify: `internal/dao/ai/run_projection_dao.go`
- Modify/Create: `storage/migration/*` for schema rewrite and backfill/drop old columns

### API Contract

- Modify: `api/ai/v1/*.go` (request/response + SSE event enums)
- Modify: `internal/ai/runtime/event_types.go` (new public event contract)
- Modify: `internal/ai/runtime/projection.go`
- Delete/Modify: old overlapping event names (`done/error/run_state` duplication logic)

### Frontend Runtime

- Create: `web/src/components/AI/runtime/store.ts`
- Create: `web/src/components/AI/runtime/reducer.ts`
- Create: `web/src/components/AI/runtime/events.ts`
- Create: `web/src/components/AI/runtime/selectors.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/api/modules/ai.ts`
- Delete: `web/src/components/AI/pendingRunStore.ts`
- Delete/replace: projection fallback code in `web/src/components/AI/historyProjection.ts`

### Tests

- Modify/Create: `internal/service/ai/logic/*_test.go`
- Modify/Create: `internal/service/ai/handler/*_test.go`
- Modify/Create: `web/src/components/AI/__tests__/*`
- Modify/Create: `web/src/api/modules/ai*.test.ts`

---

## Chunk 1: Workspace, Contract Freeze, and Schema Rewrite

### Task 1: Create isolated worktree and baseline validation

**Files:**
- Create: none
- Modify: none
- Test: none

- [ ] **Step 1: Create worktree for big-bang rewrite**

```bash
git worktree add .worktrees/ai-bigbang-rewrite -b feat/ai-bigbang-rewrite
```

- [ ] **Step 2: Switch to worktree and verify branch**

Run: `git -C .worktrees/ai-bigbang-rewrite status -sb`
Expected: `## feat/ai-bigbang-rewrite`

- [ ] **Step 3: Run baseline backend tests for AI modules**

Run: `go test ./internal/service/ai/... ./internal/ai/runtime/...`
Expected: PASS (or capture existing known failures in a note)

- [ ] **Step 4: Run baseline frontend AI tests**

Run: `npm run test:run -- web/src/components/AI web/src/api/modules/ai.test.ts web/src/api/modules/ai.contract.test.ts`
Expected: PASS (or capture existing known failures in a note)

- [ ] **Step 5: Commit baseline test snapshot note**

```bash
git add docs/superpowers/plans/2026-03-27-ai-bigbang-rewrite.md
git commit -m "docs: add big-bang AI rewrite implementation plan"
```

### Task 2: Define new canonical event contract

**Files:**
- Modify: `internal/ai/runtime/event_types.go`
- Modify: `api/ai/v1/*` (event payload contract files)
- Test: `internal/ai/runtime/event_types_test.go`

- [ ] **Step 1: Write failing tests for canonical event family**

```go
func TestCanonicalEventFamily(t *testing.T) {
    // assert only meta/timeline_delta/state/terminal are public
}
```

- [ ] **Step 2: Run test to confirm failure**

Run: `go test ./internal/ai/runtime -run CanonicalEventFamily -v`
Expected: FAIL with missing enum/payload mapping

- [ ] **Step 3: Implement minimal event type/payload definitions**

```go
const (
    EventTypeMeta          EventType = "meta"
    EventTypeTimelineDelta EventType = "timeline_delta"
    EventTypeState         EventType = "state"
    EventTypeTerminal      EventType = "terminal"
)
```

- [ ] **Step 4: Run event type tests**

Run: `go test ./internal/ai/runtime -run EventType -v`
Expected: PASS

- [ ] **Step 5: Commit canonical contract**

```bash
git add internal/ai/runtime/event_types.go internal/ai/runtime/event_types_test.go api/ai/v1
git commit -m "feat(ai): define canonical runtime event contract"
```

### Task 3: Rewrite run event storage schema for canonical timeline

**Files:**
- Modify: `internal/model/ai.go`
- Modify: `internal/dao/ai/run_event_dao.go`
- Modify/Create: `storage/migration/*`
- Test: `internal/dao/ai/run_event_dao_test.go`

- [ ] **Step 1: Write failing DAO test for strict run-seq ordering and event kind validation**

```go
func TestRunEventDAO_EnforcesCanonicalKindAndMonotonicSeq(t *testing.T) {}
```

- [ ] **Step 2: Run DAO test and confirm failure**

Run: `go test ./internal/dao/ai -run CanonicalKindAndMonotonicSeq -v`
Expected: FAIL due to missing constraints/validation

- [ ] **Step 3: Add schema migration (new columns/indexes and removal of legacy overlap fields)**

Run:
`go test ./storage/migration -run AI -v`
Expected: PASS with migration applying cleanly

- [ ] **Step 4: Update model tags and DAO query paths**

```go
Where("run_id = ? AND seq > ?", runID, cursorSeq).Order("seq ASC")
```

- [ ] **Step 5: Run DAO + migration tests**

Run: `go test ./internal/dao/ai ./storage/migration -run AI -v`
Expected: PASS

- [ ] **Step 6: Commit schema rewrite**

```bash
git add internal/model/ai.go internal/dao/ai/run_event_dao.go storage/migration
git commit -m "feat(ai): rewrite run event schema for canonical timeline"
```

---

## Chunk 2: Unified Backend Engine (Single Execution Path)

### Task 4: Introduce state machine and terminalizer

**Files:**
- Create: `internal/service/ai/engine/state_machine.go`
- Create: `internal/service/ai/engine/terminalizer.go`
- Create: `internal/service/ai/engine/types.go`
- Test: `internal/service/ai/engine/state_machine_test.go`

- [ ] **Step 1: Write failing state transition table tests**

```go
func TestStateMachine_TransitionMatrix(t *testing.T) {
    // running -> waiting_approval -> resuming -> completed
}
```

- [ ] **Step 2: Run state machine tests and confirm failure**

Run: `go test ./internal/service/ai/engine -run TransitionMatrix -v`
Expected: FAIL (state machine not implemented)

- [ ] **Step 3: Implement minimal transition validator + terminal resolver**

```go
func (m *Machine) Transit(from, byEvent string) (to string, err error)
```

- [ ] **Step 4: Run engine unit tests**

Run: `go test ./internal/service/ai/engine -v`
Expected: PASS

- [ ] **Step 5: Commit state machine foundation**

```bash
git add internal/service/ai/engine
git commit -m "feat(ai): add unified run state machine and terminalizer"
```

### Task 5: Build transactional event sink (`event -> state -> projection`)

**Files:**
- Create: `internal/service/ai/engine/event_sink.go`
- Modify: `internal/dao/ai/run_dao.go`
- Modify: `internal/dao/ai/run_projection_dao.go`
- Test: `internal/service/ai/engine/event_sink_test.go`

- [ ] **Step 1: Write failing test asserting atomic write order**

```go
func TestEventSink_WritesEventThenStateInSingleTxn(t *testing.T) {}
```

- [ ] **Step 2: Run test to confirm failure**

Run: `go test ./internal/service/ai/engine -run WritesEventThenStateInSingleTxn -v`
Expected: FAIL

- [ ] **Step 3: Implement `AppendAndApply` transaction API**

```go
func (s *EventSink) AppendAndApply(ctx context.Context, cmd ApplyCommand) error
```

- [ ] **Step 4: Run sink tests**

Run: `go test ./internal/service/ai/engine -run EventSink -v`
Expected: PASS

- [ ] **Step 5: Commit transactional sink**

```bash
git add internal/service/ai/engine/event_sink.go internal/service/ai/engine/event_sink_test.go internal/dao/ai/run_dao.go internal/dao/ai/run_projection_dao.go
git commit -m "feat(ai): add transactional event sink for run lifecycle"
```

### Task 6: Implement unified stream consumer and engine entrypoints

**Files:**
- Create: `internal/service/ai/engine/engine.go`
- Create: `internal/service/ai/engine/stream_consumer.go`
- Create: `internal/service/ai/engine/timeline.go`
- Test: `internal/service/ai/engine/engine_test.go`

- [ ] **Step 1: Write failing tests for shared chat/resume stream consumption path**

```go
func TestEngine_ChatAndResumeUseSameConsumer(t *testing.T) {}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/service/ai/engine -run ChatAndResumeUseSameConsumer -v`
Expected: FAIL

- [ ] **Step 3: Implement engine entrypoints and consume loop once**

```go
func (e *Engine) RunChat(...)
func (e *Engine) RunResume(...)
```

- [ ] **Step 4: Run engine package tests**

Run: `go test ./internal/service/ai/engine -v`
Expected: PASS

- [ ] **Step 5: Commit unified engine**

```bash
git add internal/service/ai/engine
git commit -m "feat(ai): add unified stream engine for chat and resume"
```

---

## Chunk 3: Replace Logic/Worker/Handlers with Engine Integration

### Task 7: Replace `logic.Chat` with thin orchestrator over engine

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing regression test for chat terminal convergence**

```go
func TestChat_EngineConvergesTerminalState(t *testing.T) {}
```

- [ ] **Step 2: Run logic test and confirm failure**

Run: `go test ./internal/service/ai/logic -run EngineConvergesTerminalState -v`
Expected: FAIL

- [ ] **Step 3: Remove duplicated consume loops and delegate to engine**

```go
err := l.engine.RunChat(ctx, input, emit)
```

- [ ] **Step 4: Run logic tests**

Run: `go test ./internal/service/ai/logic -v`
Expected: PASS

- [ ] **Step 5: Commit logic integration**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "refactor(ai): route chat logic through unified engine"
```

### Task 8: Replace `approval_worker` resume path with engine resume

**Files:**
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/approval_expirer.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Write failing worker test asserting no duplicated stream handling**

```go
func TestApprovalWorker_UsesEngineResumePath(t *testing.T) {}
```

- [ ] **Step 2: Run worker test and confirm failure**

Run: `go test ./internal/service/ai/logic -run UsesEngineResumePath -v`
Expected: FAIL

- [ ] **Step 3: Refactor worker to call `engine.RunResume` and delete duplicate branches**

- [ ] **Step 4: Run approval logic tests**

Run: `go test ./internal/service/ai/logic -run Approval -v`
Expected: PASS

- [ ] **Step 5: Commit worker rewrite**

```bash
git add internal/service/ai/logic/approval_worker.go internal/service/ai/logic/approval_expirer.go internal/service/ai/logic/approval_worker_test.go
git commit -m "refactor(ai): unify approval resume flow through engine"
```

### Task 9: Remove deprecated `ResumeApproval` legacy API path

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/approval/service.go`
- Modify: `internal/service/ai/approval/handler.go`
- Test: `internal/service/ai/handler/approval_test.go`

- [ ] **Step 1: Write failing contract test proving old resume endpoint is gone**

```go
func TestApprovalRoutes_DoNotExposeLegacyResumeEndpoint(t *testing.T) {}
```

- [ ] **Step 2: Run route test and verify failure**

Run: `go test ./internal/service/ai/handler -run LegacyResumeEndpoint -v`
Expected: FAIL

- [ ] **Step 3: Delete deprecated resume implementation and references**

- [ ] **Step 4: Run approval handler tests**

Run: `go test ./internal/service/ai/handler -run Approval -v`
Expected: PASS

- [ ] **Step 5: Commit legacy path removal**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/approval/service.go internal/service/ai/approval/handler.go internal/service/ai/handler/approval_test.go
git commit -m "chore(ai): remove legacy resume approval API path"
```

### Task 10: Introduce lifecycle-managed worker startup

**Files:**
- Modify: `internal/service/ai/routes.go`
- Modify: `internal/service/ai/approval/service.go`
- Test: `internal/service/ai/handler/routes_contract_test.go`

- [ ] **Step 1: Write failing test ensuring idempotent worker start semantics**

```go
func TestRegisterAIHandlers_WorkerStartIsLifecycleManaged(t *testing.T) {}
```

- [ ] **Step 2: Run test and verify failure**

Run: `go test ./internal/service/ai/handler -run WorkerStartIsLifecycleManaged -v`
Expected: FAIL

- [ ] **Step 3: Replace `context.Background()` startup with managed lifecycle hook**

- [ ] **Step 4: Run route/handler tests**

Run: `go test ./internal/service/ai/handler ./internal/service/ai/... -run Route -v`
Expected: PASS

- [ ] **Step 5: Commit lifecycle fix**

```bash
git add internal/service/ai/routes.go internal/service/ai/approval/service.go internal/service/ai/handler/routes_contract_test.go
git commit -m "refactor(ai): lifecycle-manage approval workers"
```

---

## Chunk 4: API Surface and Frontend Runtime Rewrite

### Task 11: Rewrite backend chat SSE payload emission contract

**Files:**
- Modify: `internal/service/ai/chat/handler.go`
- Modify: `internal/service/ai/chat/sse_writer.go`
- Test: `internal/service/ai/handler/sse_writer_test.go`

- [ ] **Step 1: Write failing test for canonical SSE event names and payload schema**

```go
func TestSSEWriter_EmitsCanonicalEventEnvelope(t *testing.T) {}
```

- [ ] **Step 2: Run SSE tests and verify failure**

Run: `go test ./internal/service/ai/handler -run SSEWriter_EmitsCanonicalEventEnvelope -v`
Expected: FAIL

- [ ] **Step 3: Implement canonical envelope writer and propagate write errors**

```go
type StreamEnvelope struct { EventID string; Type string; Data any }
```

- [ ] **Step 4: Run handler tests**

Run: `go test ./internal/service/ai/handler -run SSE -v`
Expected: PASS

- [ ] **Step 5: Commit SSE contract rewrite**

```bash
git add internal/service/ai/chat/handler.go internal/service/ai/chat/sse_writer.go internal/service/ai/handler/sse_writer_test.go
git commit -m "feat(ai): rewrite SSE output to canonical envelope"
```

### Task 12: Rewrite frontend API client to new timeline contract

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.contract.test.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
- Test: `web/src/api/modules/ai.test.ts`

- [ ] **Step 1: Write failing TypeScript contract tests for `timeline_delta/state/terminal` parsing**

```ts
it('parses canonical timeline events', () => { /* ... */ })
```

- [ ] **Step 2: Run frontend API tests and verify failure**

Run: `npm run test:run -- web/src/api/modules/ai.contract.test.ts web/src/api/modules/ai.streamChunk.test.ts`
Expected: FAIL

- [ ] **Step 3: Implement minimal parser and remove legacy event fallback**

- [ ] **Step 4: Re-run API module tests**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/api/modules/ai.contract.test.ts web/src/api/modules/ai.streamChunk.test.ts`
Expected: PASS

- [ ] **Step 5: Commit frontend API rewrite**

```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.test.ts web/src/api/modules/ai.contract.test.ts web/src/api/modules/ai.streamChunk.test.ts
git commit -m "feat(web-ai): switch API module to canonical timeline contract"
```

### Task 13: Build single frontend runtime store and reducer

**Files:**
- Create: `web/src/components/AI/runtime/events.ts`
- Create: `web/src/components/AI/runtime/reducer.ts`
- Create: `web/src/components/AI/runtime/store.ts`
- Create: `web/src/components/AI/runtime/selectors.ts`
- Test: `web/src/components/AI/__tests__/runtime.reducer.test.ts`

- [ ] **Step 1: Write failing reducer tests for lifecycle transitions**

```ts
it('reduces waiting_approval -> resuming -> terminal', () => { /* ... */ })
```

- [ ] **Step 2: Run reducer tests and verify failure**

Run: `npm run test:run -- web/src/components/AI/__tests__/runtime.reducer.test.ts`
Expected: FAIL

- [ ] **Step 3: Implement minimal runtime store + reducer + selectors**

- [ ] **Step 4: Run runtime tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/runtime.reducer.test.ts`
Expected: PASS

- [ ] **Step 5: Commit runtime store foundation**

```bash
git add web/src/components/AI/runtime web/src/components/AI/__tests__/runtime.reducer.test.ts
git commit -m "feat(web-ai): add single runtime store and reducer"
```

### Task 14: Rewire provider and surface components to runtime store

**Files:**
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/types.ts`
- Delete: `web/src/components/AI/pendingRunStore.ts`
- Modify/Delete: `web/src/components/AI/historyProjection.ts`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`

- [ ] **Step 1: Write failing integration tests for provider->store->UI pipeline**

```ts
it('renders approval and terminal states from runtime store', () => { /* ... */ })
```

- [ ] **Step 2: Run component tests and verify failure**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: FAIL

- [ ] **Step 3: Rewire provider and components; remove legacy pending/projection paths**

- [ ] **Step 4: Re-run AI component tests**

Run: `npm run test:run -- web/src/components/AI/__tests__`
Expected: PASS

- [ ] **Step 5: Commit frontend runtime integration**

```bash
git add web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/CopilotSurface.tsx web/src/components/AI/AssistantReply.tsx web/src/components/AI/types.ts web/src/components/AI/historyProjection.ts web/src/components/AI/__tests__
git rm -f web/src/components/AI/pendingRunStore.ts || true
git commit -m "refactor(web-ai): migrate AI surface to unified runtime store"
```

---

## Chunk 5: System Verification, Cleanup, and Final Cutover

### Task 15: Remove dead compatibility code and stale tests

**Files:**
- Modify/Delete: old runtime helpers no longer used in `internal/service/ai/logic/*`, `web/src/components/AI/*`
- Test: affected test files

- [ ] **Step 1: Identify dead symbols/references**

Run: `rg -n "pendingRunStore|historyProjection|ResumeApproval\(|legacy|deprecated" internal web/src/components/AI`
Expected: list of removable references

- [ ] **Step 2: Delete dead code with focused diffs**

- [ ] **Step 3: Run lint/build checks**

Run: `go test ./internal/service/ai/... ./internal/ai/runtime/...`
Expected: PASS

Run: `npm run test:run -- web/src/components/AI web/src/api/modules/ai.test.ts`
Expected: PASS

- [ ] **Step 4: Commit cleanup**

```bash
git add internal/service/ai internal/ai/runtime web/src/components/AI web/src/api/modules
git commit -m "chore(ai): remove dead compatibility paths after big-bang rewrite"
```

### Task 16: End-to-end chain verification (critical)

**Files:**
- Modify/Create: integration tests under `internal/service/ai/handler/*_test.go`
- Modify/Create: frontend integration tests under `web/src/components/AI/__tests__`

- [ ] **Step 1: Write failing backend integration test for full chain**

```go
func TestAIChain_ChatApprovalResumeTerminal(t *testing.T) {}
```

- [ ] **Step 2: Run integration test and confirm failure**

Run: `go test ./internal/service/ai/handler -run ChatApprovalResumeTerminal -v`
Expected: FAIL

- [ ] **Step 3: Implement missing glue until chain passes**

- [ ] **Step 4: Re-run backend + frontend critical suites**

Run: `go test ./internal/service/ai/... ./internal/ai/runtime/... ./internal/dao/ai/...`
Expected: PASS

Run: `npm run test:run -- web/src/api/modules/ai.contract.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS

- [ ] **Step 5: Commit E2E convergence**

```bash
git add internal/service/ai/handler internal/service/ai/logic internal/service/ai/engine web/src/components/AI web/src/api/modules
git commit -m "test(ai): validate full chat->approval->resume->terminal chain"
```

### Task 17: Final verification and release notes

**Files:**
- Modify/Create: `docs/superpowers/runbooks/ai-bigbang-cutover.md`

- [ ] **Step 1: Write cutover runbook (dev environment)**

```md
- apply migrations
- restart backend
- run smoke tests
```

- [ ] **Step 2: Run full verification before merge**

Run: `go test ./...`
Expected: PASS

Run: `npm run test:run`
Expected: PASS

- [ ] **Step 3: Commit runbook and verification state**

```bash
git add docs/superpowers/runbooks/ai-bigbang-cutover.md
git commit -m "docs(ai): add big-bang cutover runbook"
```

---

## Execution Rules for Implementers

- Follow `@test-driven-development` on every task before implementation.
- Use `@systematic-debugging` for any unexpected failure.
- Run `@verification-before-completion` before claiming done.
- Keep commits small and scoped to one task.
- Do not reintroduce compatibility adapters.
- Prefer deletion over migration shims (YAGNI).

---

## Plan Review Loop (Required During Execution)

For each chunk (`Chunk 1`..`Chunk 5`):

1. Dispatch `plan-document-reviewer` subagent with:
   - chunk markdown content
   - spec path: `docs/superpowers/specs/2026-03-27-ai-bigbang-rewrite-design.md` (or the approved design path)
2. If issues are reported:
   - fix chunk
   - re-dispatch reviewer
   - repeat until approved (max 5 iterations)
3. Execute only approved chunk tasks.

