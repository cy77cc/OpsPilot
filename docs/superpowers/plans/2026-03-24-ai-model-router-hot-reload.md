# AI Model Router Hot Reload Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make AI model/default-provider changes in DB take effect for new chat requests without service restart, while ongoing runs continue on their existing router snapshot.

**Architecture:** Add a request-time router freshness gate in `Logic.Chat` that checks a cached DB config version and rebuilds router lazily only when version changes. Publish router+version as an immutable atomic snapshot to avoid races, and guard rebuild/query fan-out with `sync.Mutex` + `singleflight`. On rebuild failure, keep serving with old router and emit observability signals.

**Tech Stack:** Go, Gin, GORM, `sync/atomic`, `golang.org/x/sync/singleflight`, existing AI logic/DAO/test stack.

**Critical Safety Rules (must enforce in implementation):**
- Version query must use index-friendly predicates/ordering; verify index on `(deleted_at, updated_at)` (or equivalent) before merge.
- Router hot-swap must include old-router lifecycle handling (explicit close if supported, otherwise documented no-op policy).
- `singleflight` return value must use safe type assertion with fallback path (no panic).
- Rebuild success must reset backoff state (`nextRetryAt`, current backoff), preventing stale throttling.

---

## Chunk 1: Router State + Version Probe Foundations

### Task 1: Add immutable router snapshot and TTL query cache fields in Logic

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing tests for new router state behavior scaffolding**

```go
func TestLogic_LoadRouterState_EmptyWhenUninitialized(t *testing.T) { /* ... */ }
func TestLogic_StoreRouterState_ReplacesAtomically(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run target test to verify failure**

Run: `go test ./internal/service/ai/logic -run 'TestLogic_(LoadRouterState_EmptyWhenUninitialized|StoreRouterState_ReplacesAtomically)' -v`
Expected: FAIL (missing types/methods)

- [ ] **Step 3: Implement minimal router snapshot primitives**

Add to `Logic` and helpers in `logic.go`:

```go
type routerState struct {
    router    adk.ResumableAgent
    version   string
    versionAt time.Time
}

// in Logic:
routerStatePtr atomic.Pointer[routerState]
routerBuildMu  sync.Mutex
routerVersionSF singleflight.Group
versionCacheTTL time.Duration

func (l *Logic) loadRouterState() *routerState { /* ... */ }
func (l *Logic) storeRouterState(s *routerState) { /* ... */ }
```

Initialize initial state in `NewAILogic(...)` with existing `aiRouter` and empty version.

- [ ] **Step 4: Run test to verify pass**

Run: `go test ./internal/service/ai/logic -run 'TestLogic_(LoadRouterState_EmptyWhenUninitialized|StoreRouterState_ReplacesAtomically)' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "refactor(ai): add atomic router state primitives"
```

### Task 2: Add DB model config version query API (time/count + hash fallback)

**Files:**
- Modify: `internal/dao/ai/llm_provider_dao.go`
- Modify: `internal/dao/ai/llm_provider_dao_test.go`

- [ ] **Step 1: Write failing DAO tests for version computation**

```go
func TestLLMProviderDAO_GetConfigVersion_TimeCount(t *testing.T) { /* ... */ }
func TestLLMProviderDAO_GetConfigVersion_UsesHashFallbackWhenNoHighPrecision(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run DAO tests to verify failure**

Run: `go test ./internal/dao/ai -run 'TestLLMProviderDAO_GetConfigVersion_' -v`
Expected: FAIL (method not found)

- [ ] **Step 3: Implement version query method**

Add in `llm_provider_dao.go`:

```go
func (d *LLMProviderDAO) GetConfigVersion(ctx context.Context) (string, error) {
    // Query count + max(updated_at)
    // Build <unixNano>:<count>
    // Optional fallback: stable hash of selected fields when needed
}
```

Use only non-deleted rows (`deleted_at IS NULL`).

- [ ] **Step 3.1: Add/verify supporting index to prevent query degradation**

Ensure query plan can use index for version probe:
- verify existing index coverage for `deleted_at`, `updated_at`
- if missing, add migration/index update and corresponding migration test

Run (example): `EXPLAIN`-equivalent check in test/dev DB or migration test assertion.

- [ ] **Step 4: Run DAO tests**

Run: `go test ./internal/dao/ai -run 'TestLLMProviderDAO_GetConfigVersion_' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/dao/ai/llm_provider_dao.go internal/dao/ai/llm_provider_dao_test.go storage/migration/* internal/model/*
git commit -m "feat(ai): add llm provider config version query with index safeguards"
```

## Chunk 2: Router Freshness Gate and Chat Integration

### Task 3: Implement ensureRouterFresh with DCL + singleflight + fallback

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing tests for freshness gate**

```go
func TestEnsureRouterFresh_NoVersionChange_ReusesRouter(t *testing.T) { /* ... */ }
func TestEnsureRouterFresh_VersionChanged_RebuildsOnce(t *testing.T) { /* ... */ }
func TestEnsureRouterFresh_RebuildFail_FallsBackOldRouter(t *testing.T) { /* ... */ }
func TestEnsureRouterFresh_ConcurrentCalls_OneVersionQueryAndOneRebuild(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run target tests to verify failure**

Run: `go test ./internal/service/ai/logic -run 'TestEnsureRouterFresh_' -v`
Expected: FAIL

- [ ] **Step 3: Implement freshness gate**

In `logic.go` add:

```go
func (l *Logic) ensureRouterFresh(ctx context.Context) (adk.ResumableAgent, error) {
    // load atomic snapshot
    // check cached TTL
    // singleflight version query
    // fast return when unchanged
    // buildMu lock + second check + rebuild
    // on rebuild error: return old router + log/metric
}
```

Add override hooks for tests if needed (e.g., function fields):

```go
queryRouterVersionFn func(context.Context) (string, error)
buildRouterFn func(context.Context) (adk.ResumableAgent, error)
```

Mandatory details in implementation:
- Handle `singleflight.Do` return with safe assertion:

```go
v, err, _ := l.routerVersionSF.Do("router-version", fn)
if err != nil { /* fallback */ }
version, ok := v.(string)
if !ok { /* fallback with error, no panic */ }
```

- On successful swap, perform old-router lifecycle hook:
  - if old router implements `interface{ Close(context.Context) error }`, invoke async graceful close;
  - if not closable, document and no-op explicitly.

- [ ] **Step 4: Run target tests**

Run: `go test ./internal/service/ai/logic -run 'TestEnsureRouterFresh_' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "feat(ai): add router freshness gate with atomic+dcl+singleflight"
```

### Task 4: Wire Chat to use request-scoped router snapshot

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing test ensuring Chat picks refreshed router for new requests**

```go
func TestChat_UsesFreshRouterSnapshotPerRequest(t *testing.T) { /* request1 old, request2 new */ }
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/service/ai/logic -run 'TestChat_UsesFreshRouterSnapshotPerRequest' -v`
Expected: FAIL

- [ ] **Step 3: Implement Chat integration**

In `Chat(...)`:
- Call `router, err := l.ensureRouterFresh(ctx)` at start.
- Replace `RunnerConfig.Agent: l.AIRouter` with `Agent: router`.
- Preserve current behavior for uninitialized service (`AI service not initialized`).

- [ ] **Step 4: Run test**

Run: `go test ./internal/service/ai/logic -run 'TestChat_UsesFreshRouterSnapshotPerRequest' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "feat(ai): bind chat runner to request-scoped fresh router"
```

## Chunk 3: Observability, Backoff, and Regression Coverage

### Task 5: Add observability for version change/reload/fallback

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/approval_event_metrics.go` (or create dedicated metrics file)
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing tests for fallback logging/metrics hooks**

```go
func TestEnsureRouterFresh_RebuildFail_EmitsFallbackSignal(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/service/ai/logic -run 'TestEnsureRouterFresh_RebuildFail_EmitsFallbackSignal' -v`
Expected: FAIL

- [ ] **Step 3: Implement logs/metrics**

Emit at least:
- reload success/fail counter
- reload duration
- fallback flag when serving old router after failed rebuild

- [ ] **Step 4: Run test**

Run: `go test ./internal/service/ai/logic -run 'TestEnsureRouterFresh_RebuildFail_EmitsFallbackSignal' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/approval_event_metrics.go internal/service/ai/logic/logic_test.go
git commit -m "chore(ai): add router hot-reload observability and fallback signal"
```

### Task 6: Add startup/rebuild failure backoff

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing tests for backoff behavior**

```go
func TestEnsureRouterFresh_RebuildFailure_BackoffSkipsImmediateRetry(t *testing.T) { /* ... */ }
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/service/ai/logic -run 'TestEnsureRouterFresh_RebuildFailure_BackoffSkipsImmediateRetry' -v`
Expected: FAIL

- [ ] **Step 3: Implement minimal backoff state**

Add fields:

```go
nextRetryAt atomic.Int64 // unix nano
backoff time.Duration
maxBackoff time.Duration
```

Behavior:
- On rebuild failure, set `nextRetryAt` with bounded exponential backoff.
- Before rebuild, if now < nextRetryAt, skip rebuild and return current router.
- On rebuild success, reset `nextRetryAt` and current backoff to zero/base.

- [ ] **Step 4: Run test**

Run: `go test ./internal/service/ai/logic -run 'TestEnsureRouterFresh_RebuildFailure_BackoffSkipsImmediateRetry' -v`
Expected: PASS

- [ ] **Step 4.1: Add and run success-reset regression test**

```go
func TestEnsureRouterFresh_RebuildSuccess_ResetsBackoffState(t *testing.T) { /* ... */ }
```

Run: `go test ./internal/service/ai/logic -run 'TestEnsureRouterFresh_Rebuild(Success_ResetsBackoffState|Failure_BackoffSkipsImmediateRetry)' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "feat(ai): add router rebuild backoff on repeated failures"
```

## Chunk 4: End-to-End Validation and Documentation

### Task 7: Add integration-style regression test for default model switch without restart

**Files:**
- Modify: `internal/service/ai/logic/logic_test.go` (or create `logic_hot_reload_test.go`)
- Optional modify: `internal/ai/chatmodel/registry_test.go`

- [ ] **Step 1: Write failing end-to-end regression test**

```go
func TestChat_DefaultModelSwitch_TakesEffectForNewRequestsWithoutRestart(t *testing.T) {
    // seed default A
    // request #1 uses A
    // update default to B in DB
    // request #2 uses B
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/service/ai/logic -run 'TestChat_DefaultModelSwitch_TakesEffectForNewRequestsWithoutRestart' -v`
Expected: FAIL

- [ ] **Step 3: Adjust implementation/tests as needed (minimal)**

No new features; only fix gaps found by regression test.

- [ ] **Step 4: Run focused + package tests**

Run:
- `go test ./internal/service/ai/logic -run 'TestChat_DefaultModelSwitch_TakesEffectForNewRequestsWithoutRestart' -v`
- `go test ./internal/service/ai/logic ./internal/dao/ai ./internal/ai/chatmodel`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic_test.go internal/ai/chatmodel/registry_test.go internal/service/ai/logic/logic.go internal/dao/ai/llm_provider_dao.go
git commit -m "test(ai): cover model switch hot-reload behavior for new requests"
```

### Task 8: Update design/plan linkage docs

**Files:**
- Modify: `docs/superpowers/specs/2026-03-24-ai-model-router-hot-reload-design.md`
- Modify: `docs/superpowers/plans/2026-03-24-ai-model-router-hot-reload.md`

- [ ] **Step 1: Add implementation notes/checklist references**

Document any deliberate deviations from spec and final chosen version strategy.

- [ ] **Step 2: Run quick doc sanity check**

Run: `rg -n "TODO|TBD|FIXME" docs/superpowers/specs/2026-03-24-ai-model-router-hot-reload-design.md docs/superpowers/plans/2026-03-24-ai-model-router-hot-reload.md`
Expected: no unresolved placeholders for this scope.

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/specs/2026-03-24-ai-model-router-hot-reload-design.md docs/superpowers/plans/2026-03-24-ai-model-router-hot-reload.md
git commit -m "docs(ai): align hot-reload spec and execution plan"
```

---

## Full Verification Before Merge

- [ ] Run core package tests:

```bash
go test ./internal/service/ai/logic ./internal/dao/ai ./internal/ai/chatmodel ./internal/ai/agents/...
```

- [ ] Run broader AI regression sweep (if time permits):

```bash
go test ./internal/service/ai/... ./internal/ai/...
```

- [ ] Confirm no accidental frontend/API contract break:

```bash
go test ./internal/service/ai/handler -run 'TestAdminLLMProvider|TestChat' -v
```

- [ ] Capture final verification summary in PR description / execution log.
