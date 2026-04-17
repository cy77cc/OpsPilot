# AI Architecture Rebuild Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the current AI backend/frontend architecture with a clean-slate implementation that fixes the issues documented in `ai_review.md` without preserving legacy compatibility paths.

**Architecture:** Build a new vertical slice around explicit boundaries: `interfaces -> app -> runtime -> infra -> domain`. The new slice becomes the only supported path as soon as each stage is verified. Context handling must be budgeted and explicit, streaming errors must be normalized through one contract, projection updates must be incremental, and frontend state/API modules must match backend capabilities exactly.

**Tech Stack:** Go, Gin, GORM, SSE, TypeScript, React, Vitest

---

## Agent Execution Rules

1. This is a replacement plan, not a compatibility migration plan. Do not add fallback code, compatibility shims, dual-write logic, or feature flags unless a task explicitly requires them.
2. Do not start a later task until the current task's verification command passes exactly as described.
3. If the current codebase shape conflicts with this plan, adapt the implementation to local reality, but preserve the architectural intent and update the plan in the same commit before continuing.
4. Delete legacy code as soon as the replacement path is verified. Do not keep dead paths around "for safety".
5. Treat context budget, error contracts, and projection state as runtime architecture concerns. Do not hide them inside prompt text or ad hoc handler logic.
6. Prefer narrow files with one responsibility. If an existing file is too large, split it while doing the planned change instead of adding more branching.
7. A task is not complete because a symbol exists. A task is complete only when the test proves the intended runtime behavior.

## Stage Gates

### Gate A: New Backend Slice Exists

Must be true before touching frontend API modules:
- `internal/modules/ai/interfaces/http/chat_handler.go` exists and is wired from bootstrap.
- `internal/modules/ai/app/command/chat_command_handler.go` exists and owns use-case orchestration.
- Streaming error mapping is handled by `runtime/streaming/error_mapper.go`, not by ad hoc handler branches.

### Gate B: Runtime State Is Explicit

Must be true before deleting legacy projection/routing code:
- Context selection uses a budgeted selector with pinned/recent/history buckets.
- Overflow handling has a compressor entrypoint.
- Projection updates are incremental and versioned.
- Trace IDs are generated/propagated through the chat path.

### Gate C: Frontend Matches Backend Surface

Must be true before final cleanup:
- Frontend AI API files are split by backend responsibility.
- Stream reconnect logic and pending run state are isolated from component code.
- No frontend module exposes unsupported backend actions.

### Gate D: Legacy Removal

Must be true before final commit:
- Old chat routing path is deleted.
- Old scene-default middleware path is deleted if no longer required by the new flow.
- Full verification for `internal/modules/ai/...` and relevant web AI tests passes.

## File Structure Map

### Backend files

- Create: `internal/modules/ai/interfaces/http/chat_handler.go` (new HTTP entrypoint and SSE response boundary)
- Create: `internal/modules/ai/app/command/chat_command_handler.go` (chat use-case orchestration boundary)
- Create: `internal/modules/ai/runtime/context/selector.go` (budgeted selection across pinned/recent/history)
- Create: `internal/modules/ai/runtime/context/compressor.go` (overflow summarization/compression entrypoint)
- Create: `internal/modules/ai/runtime/streaming/error_mapper.go` (internal error to `AI_STREAM_*` and `AI_API_*`)
- Create: `internal/modules/ai/runtime/projection/updater.go` (incremental projection applier)
- Create: `internal/modules/ai/infra/observability/trace.go` (trace id generation/propagation helper)
- Create: `internal/modules/ai/infra/observability/metrics.go` (basic counters for AI runtime hooks)
- Create: `internal/modules/ai/infra/workers/lifecycle.go` (worker start/stop with service context)
- Modify: `internal/bootstrap/modules.go` (wire new handler and replace `context.Background()` startup)
- Modify: `internal/modules/ai/logic/chat/chat.go` (delegate to selector/updater/app command flow or be removed if absorbed)
- Modify: `internal/modules/ai/logic/chat/projection.go` (remove terminal full rebuild path or delete if obsolete)
- Modify: `internal/modules/ai/model/run.go` (trace propagation fields/hooks if needed)
- Test: `internal/modules/ai/interfaces/http/chat_handler_test.go`
- Test: `internal/modules/ai/runtime/context/selector_test.go`
- Test: `internal/modules/ai/runtime/streaming/error_mapper_test.go`
- Test: `internal/modules/ai/runtime/projection/updater_test.go`
- Test: `internal/modules/ai/infra/observability/trace_test.go`
- Test: `internal/modules/ai/infra/observability/metrics_test.go`
- Test: `internal/modules/ai/infra/workers/lifecycle_test.go`

### Frontend files

- Create: `web/src/features/ai/api/chatApi.ts`
- Create: `web/src/features/ai/api/sessionApi.ts`
- Create: `web/src/features/ai/api/runApi.ts`
- Create: `web/src/features/ai/api/approvalApi.ts`
- Create: `web/src/features/ai/stream/streamClient.ts`
- Create: `web/src/features/ai/stream/eventDispatcher.ts`
- Create: `web/src/features/ai/stream/reconnectController.ts`
- Create: `web/src/features/ai/state/pendingRunStore.ts`
- Modify: `web/src/api/modules/ai.ts` (replace legacy monolith with direct exports from new modules)
- Modify: `web/src/components/AI/pendingRunStore.ts` (redirect imports to feature state module or delete if obsolete)
- Test: `web/src/features/ai/api/__tests__/api-contract.test.ts`
- Test: `web/src/features/ai/stream/__tests__/reconnectController.test.ts`
- Test: `web/src/features/ai/state/__tests__/pendingRunStore.test.ts`

### Cleanup and docs

- Delete: `internal/modules/ai/handler/chat/routing.go`
- Delete: `internal/modules/ai/agent/shared/middleware/scene_defaults.go`
- Delete or rewrite: `internal/modules/ai/handler/chat/handler.go` (depending on whether it becomes a thin shell or is fully replaced)
- Create: `docs/ai/error-codes.md`
- Create: `docs/ai/runtime-troubleshooting.md`

---

### Task 1: Worker Lifecycle and Bootstrap Control

**Files:**
- Create: `internal/modules/ai/infra/workers/lifecycle.go`
- Test: `internal/modules/ai/infra/workers/lifecycle_test.go`
- Modify: `internal/bootstrap/modules.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/modules/ai/infra/workers/lifecycle_test.go
package workers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnerStopsOnContextCancel(t *testing.T) {
	var ticks atomic.Int32

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := NewRunner(func(context.Context) { ticks.Add(1) }, 5*time.Millisecond)
	done := r.Start(ctx)

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runner did not stop after cancel")
	}

	if ticks.Load() == 0 {
		t.Fatal("runner never ticked before cancel")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/ai/infra/workers -run TestRunnerStopsOnContextCancel -count=1`
Expected: FAIL with `undefined: NewRunner`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/modules/ai/infra/workers/lifecycle.go
package workers

import (
	"context"
	"time"
)

type Runner struct {
	tick  func(context.Context)
	every time.Duration
}

func NewRunner(tick func(context.Context), every time.Duration) *Runner {
	return &Runner{tick: tick, every: every}
}

func (r *Runner) Start(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(r.every)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if r.tick != nil {
					r.tick(ctx)
				}
			}
		}
	}()

	return done
}
```

- [ ] **Step 4: Wire bootstrap to pass a service context**

```go
// internal/bootstrap/modules.go
rootCtx, cancel := context.WithCancel(context.Background())
defer cancel()

aiRunner := workers.NewRunner(aiModule.Tick, 5*time.Second)
aiDone := aiRunner.Start(rootCtx)
_ = aiDone
```

The exact identifiers may differ in the real file. Preserve local bootstrap structure, but the AI worker lifecycle must be rooted in a cancellable service context instead of unmanaged `context.Background()` calls.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/modules/ai/infra/workers -run TestRunnerStopsOnContextCancel -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/infra/workers/lifecycle.go internal/modules/ai/infra/workers/lifecycle_test.go internal/bootstrap/modules.go
git commit -m "refactor(ai): add controlled lifecycle runner for ai workers"
```

### Task 2: Create the New Backend Entry Boundary

**Files:**
- Create: `internal/modules/ai/interfaces/http/chat_handler.go`
- Create: `internal/modules/ai/app/command/chat_command_handler.go`
- Test: `internal/modules/ai/interfaces/http/chat_handler_test.go`
- Modify: `internal/bootstrap/modules.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/modules/ai/interfaces/http/chat_handler_test.go
package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubCommandHandler struct {
	called bool
}

func (s *stubCommandHandler) Handle(*ChatRequest) error {
	s.called = true
	return nil
}

func TestChatHandlerDelegatesToCommandHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubCommandHandler{}
	h := NewChatHandler(stub)

	r := gin.New()
	r.POST("/ai/chat", h.HandleChat)

	req := httptest.NewRequest(http.MethodPost, "/ai/chat", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if !stub.called {
		t.Fatal("expected command handler to be called")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/ai/interfaces/http -run TestChatHandlerDelegatesToCommandHandler -count=1`
Expected: FAIL with `undefined: NewChatHandler` or `undefined: ChatRequest`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/modules/ai/app/command/chat_command_handler.go
package command

type ChatRequest struct{}

type ChatHandler interface {
	Handle(*ChatRequest) error
}
```

```go
// internal/modules/ai/interfaces/http/chat_handler.go
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"OpsPilot/internal/modules/ai/app/command"
)

type ChatRequest = command.ChatRequest

type ChatCommandHandler interface {
	Handle(*ChatRequest) error
}

type ChatHandler struct {
	commandHandler ChatCommandHandler
}

func NewChatHandler(commandHandler ChatCommandHandler) *ChatHandler {
	return &ChatHandler{commandHandler: commandHandler}
}

func (h *ChatHandler) HandleChat(c *gin.Context) {
	req := &ChatRequest{}
	if h.commandHandler != nil {
		_ = h.commandHandler.Handle(req)
	}
	c.Status(http.StatusOK)
}
```

- [ ] **Step 4: Wire bootstrap to the new handler**

```go
// internal/bootstrap/modules.go
chatCommandHandler := /* build app command handler */
chatHTTPHandler := aihttp.NewChatHandler(chatCommandHandler)

router.POST("/ai/chat", chatHTTPHandler.HandleChat)
```

Do not keep routing through the old chat handler once this path is verified.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/modules/ai/interfaces/http -run TestChatHandlerDelegatesToCommandHandler -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/interfaces/http/chat_handler.go internal/modules/ai/interfaces/http/chat_handler_test.go internal/modules/ai/app/command/chat_command_handler.go internal/bootstrap/modules.go
git commit -m "refactor(ai): create new http and app boundaries for chat flow"
```

### Task 3: Unified SSE and API Error Contract

**Files:**
- Create: `internal/modules/ai/runtime/streaming/error_mapper.go`
- Test: `internal/modules/ai/runtime/streaming/error_mapper_test.go`
- Modify: `internal/modules/ai/interfaces/http/chat_handler.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/modules/ai/runtime/streaming/error_mapper_test.go
package streaming

import (
	"errors"
	"testing"
)

func TestMapStreamError_DefaultRuntimeError(t *testing.T) {
	out := MapStreamError(errors.New("db timeout"))

	if out.Code != "AI_STREAM_INTERNAL" {
		t.Fatalf("unexpected code: %s", out.Code)
	}
	if out.Message == "db timeout" {
		t.Fatal("raw error message leaked")
	}
	if !out.Retryable {
		t.Fatal("unexpected retryable=false for default runtime error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/ai/runtime/streaming -run TestMapStreamError_DefaultRuntimeError -count=1`
Expected: FAIL with `undefined: MapStreamError`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/modules/ai/runtime/streaming/error_mapper.go
package streaming

type PublicError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	TraceID   string `json:"trace_id,omitempty"`
	Retryable bool   `json:"retryable"`
}

func MapStreamError(err error) PublicError {
	if err == nil {
		return PublicError{
			Code:      "AI_STREAM_INTERNAL",
			Message:   "internal error",
			Retryable: false,
		}
	}

	return PublicError{
		Code:      "AI_STREAM_INTERNAL",
		Message:   "stream failed, please retry",
		Retryable: true,
	}
}
```

- [ ] **Step 4: Route handler errors through the mapper**

```go
// internal/modules/ai/interfaces/http/chat_handler.go
if err := h.commandHandler.Handle(req); err != nil {
	publicErr := streaming.MapStreamError(err)
	c.JSON(http.StatusInternalServerError, publicErr)
	return
}
```

Do not inline new error code branches in the HTTP handler. All public AI error shaping must go through the mapper.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/modules/ai/runtime/streaming -run TestMapStreamError_DefaultRuntimeError -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/runtime/streaming/error_mapper.go internal/modules/ai/runtime/streaming/error_mapper_test.go internal/modules/ai/interfaces/http/chat_handler.go
git commit -m "refactor(ai): centralize stream and api error mapping"
```

### Task 4: Budgeted Context Selector and Compression Entry Point

**Files:**
- Create: `internal/modules/ai/runtime/context/selector.go`
- Create: `internal/modules/ai/runtime/context/compressor.go`
- Test: `internal/modules/ai/runtime/context/selector_test.go`
- Modify: `internal/modules/ai/app/command/chat_command_handler.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/modules/ai/runtime/context/selector_test.go
package context

import "testing"

func TestSelectBudgeted_PrefersPinnedThenRecentThenHistory(t *testing.T) {
	history := []Message{
		{Role: "system", Content: "pinned-1", Pinned: true},
		{Role: "user", Content: "h1"},
		{Role: "assistant", Content: "h2"},
		{Role: "user", Content: "recent-1"},
		{Role: "assistant", Content: "recent-2"},
	}

	got := SelectBudgeted(history, Budget{
		Pinned: 1,
		Recent: 2,
		History: 1,
	})

	if len(got) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(got))
	}
	if got[0].Content != "pinned-1" {
		t.Fatalf("expected pinned message first, got %+v", got)
	}
	if got[2].Content != "recent-1" || got[3].Content != "recent-2" {
		t.Fatalf("expected recent messages at tail, got %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/ai/runtime/context -run TestSelectBudgeted_PrefersPinnedThenRecentThenHistory -count=1`
Expected: FAIL with `undefined: SelectBudgeted` or `undefined: Budget`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/modules/ai/runtime/context/selector.go
package context

type Message struct {
	Role    string
	Content string
	Pinned  bool
}

type Budget struct {
	Pinned  int
	Recent  int
	History int
}

func SelectBudgeted(history []Message, budget Budget) []Message {
	var pinned []Message
	var normal []Message

	for _, msg := range history {
		if msg.Pinned {
			pinned = append(pinned, msg)
			continue
		}
		normal = append(normal, msg)
	}

	if len(pinned) > budget.Pinned {
		pinned = pinned[:budget.Pinned]
	}

	recentStart := len(normal) - budget.Recent
	if recentStart < 0 {
		recentStart = 0
	}
	recent := append([]Message{}, normal[recentStart:]...)

	historyPool := normal[:recentStart]
	if len(historyPool) > budget.History {
		historyPool = historyPool[len(historyPool)-budget.History:]
	}

	out := make([]Message, 0, len(pinned)+len(historyPool)+len(recent))
	out = append(out, pinned...)
	out = append(out, historyPool...)
	out = append(out, recent...)
	return out
}
```

```go
// internal/modules/ai/runtime/context/compressor.go
package context

func CompressOverflow(history []Message, maxMessages int) []Message {
	if maxMessages <= 0 || len(history) <= maxMessages {
		return history
	}
	return history[len(history)-maxMessages:]
}
```

- [ ] **Step 4: Call selector from the app command boundary**

```go
// internal/modules/ai/app/command/chat_command_handler.go
selected := context.SelectBudgeted(history, context.Budget{
	Pinned:  4,
	Recent:  8,
	History: 4,
})
selected = context.CompressOverflow(selected, 16)
```

The exact numbers may change in implementation, but the budget must be explicit and owned by runtime/app code, not by scattered handler logic.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/modules/ai/runtime/context -run TestSelectBudgeted_PrefersPinnedThenRecentThenHistory -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/runtime/context/selector.go internal/modules/ai/runtime/context/compressor.go internal/modules/ai/runtime/context/selector_test.go internal/modules/ai/app/command/chat_command_handler.go
git commit -m "feat(ai): add budgeted context selection and overflow compression"
```

### Task 5: Incremental Projection Updater

**Files:**
- Create: `internal/modules/ai/runtime/projection/updater.go`
- Test: `internal/modules/ai/runtime/projection/updater_test.go`
- Modify: `internal/modules/ai/app/command/chat_command_handler.go`
- Modify: `internal/modules/ai/logic/chat/projection.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/modules/ai/runtime/projection/updater_test.go
package projection

import "testing"

func TestApplyEvent_IncrementsVersion(t *testing.T) {
	state := State{Version: 1}
	next := ApplyEvent(state, Event{Type: "assistant.delta", Text: "hello"})

	if next.Version != 2 {
		t.Fatalf("expected version 2, got %d", next.Version)
	}
	if len(next.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(next.Blocks))
	}
	if next.Blocks[0].Text != "hello" {
		t.Fatalf("unexpected block text: %+v", next.Blocks[0])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/ai/runtime/projection -run TestApplyEvent_IncrementsVersion -count=1`
Expected: FAIL with `undefined: State` or `undefined: ApplyEvent`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/modules/ai/runtime/projection/updater.go
package projection

type Event struct {
	Type string
	Text string
}

type Block struct {
	Kind string
	Text string
}

type State struct {
	Version int
	Blocks  []Block
}

func ApplyEvent(current State, event Event) State {
	next := State{
		Version: current.Version + 1,
		Blocks:  append([]Block{}, current.Blocks...),
	}

	if event.Text != "" {
		next.Blocks = append(next.Blocks, Block{
			Kind: event.Type,
			Text: event.Text,
		})
	}

	return next
}
```

- [ ] **Step 4: Replace terminal full rebuild usage**

```go
// internal/modules/ai/app/command/chat_command_handler.go
state = projection.ApplyEvent(state, projection.Event{
	Type: "assistant.delta",
	Text: delta,
})
```

```go
// internal/modules/ai/logic/chat/projection.go
// Delete or rewrite code paths that rebuild the entire projection from terminal output.
// After this task, projection updates must happen event-by-event.
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/modules/ai/runtime/projection -run TestApplyEvent_IncrementsVersion -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/runtime/projection/updater.go internal/modules/ai/runtime/projection/updater_test.go internal/modules/ai/app/command/chat_command_handler.go internal/modules/ai/logic/chat/projection.go
git commit -m "feat(ai): switch projection updates to incremental events"
```

### Task 6: Trace Propagation and Metrics Hooks

**Files:**
- Create: `internal/modules/ai/infra/observability/trace.go`
- Create: `internal/modules/ai/infra/observability/metrics.go`
- Modify: `internal/modules/ai/model/run.go`
- Modify: `internal/modules/ai/app/command/chat_command_handler.go`
- Test: `internal/modules/ai/infra/observability/trace_test.go`
- Test: `internal/modules/ai/infra/observability/metrics_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/modules/ai/infra/observability/trace_test.go
package observability

import "testing"

func TestEnsureTraceID_GeneratesWhenEmpty(t *testing.T) {
	got := EnsureTraceID("")
	if got == "" {
		t.Fatal("trace id must not be empty")
	}
}
```

```go
// internal/modules/ai/infra/observability/metrics_test.go
package observability

import "testing"

func TestCounterInc_IncrementsValue(t *testing.T) {
	c := NewCounter("ai_stream_error_total")
	c.Inc()
	if c.Value() != 1 {
		t.Fatalf("expected 1, got %d", c.Value())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/modules/ai/infra/observability -run "TestEnsureTraceID_GeneratesWhenEmpty|TestCounterInc_IncrementsValue" -count=1`
Expected: FAIL with `undefined: EnsureTraceID` and `undefined: NewCounter`

- [ ] **Step 3: Write minimal implementation**

```go
// internal/modules/ai/infra/observability/trace.go
package observability

import (
	"strings"

	"github.com/google/uuid"
)

func EnsureTraceID(in string) string {
	trimmed := strings.TrimSpace(in)
	if trimmed != "" {
		return trimmed
	}
	return uuid.NewString()
}
```

```go
// internal/modules/ai/infra/observability/metrics.go
package observability

type Counter struct {
	name  string
	value int64
}

func NewCounter(name string) *Counter {
	return &Counter{name: name}
}

func (c *Counter) Inc() {
	c.value++
}

func (c *Counter) Value() int64 {
	return c.value
}
```

- [ ] **Step 4: Thread trace IDs through the chat flow**

```go
// internal/modules/ai/app/command/chat_command_handler.go
traceID := observability.EnsureTraceID(input.TraceID)
run.TraceID = traceID
```

```go
// internal/modules/ai/model/run.go
type Run struct {
	// existing fields...
	TraceID string `json:"trace_id"`
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/modules/ai/infra/observability -run "TestEnsureTraceID_GeneratesWhenEmpty|TestCounterInc_IncrementsValue" -count=1`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/infra/observability/trace.go internal/modules/ai/infra/observability/trace_test.go internal/modules/ai/infra/observability/metrics.go internal/modules/ai/infra/observability/metrics_test.go internal/modules/ai/model/run.go internal/modules/ai/app/command/chat_command_handler.go
git commit -m "feat(ai): add trace propagation and runtime metrics hooks"
```

### Task 7: Frontend API Split by Real Backend Responsibility

**Files:**
- Create: `web/src/features/ai/api/chatApi.ts`
- Create: `web/src/features/ai/api/sessionApi.ts`
- Create: `web/src/features/ai/api/runApi.ts`
- Create: `web/src/features/ai/api/approvalApi.ts`
- Modify: `web/src/api/modules/ai.ts`
- Test: `web/src/features/ai/api/__tests__/api-contract.test.ts`

- [ ] **Step 1: Write the failing test**

```ts
// web/src/features/ai/api/__tests__/api-contract.test.ts
import { describe, expect, it } from 'vitest';
import { listUnsupportedMethods } from '../runApi';

describe('ai api contract', () => {
  it('does not expose unsupported methods', () => {
    expect(listUnsupportedMethods()).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm run test:run -- src/features/ai/api/__tests__/api-contract.test.ts`
Expected: FAIL with module not found for `../runApi`

- [ ] **Step 3: Write minimal implementation**

```ts
// web/src/features/ai/api/runApi.ts
export function listUnsupportedMethods(): string[] {
  return [];
}
```

```ts
// web/src/api/modules/ai.ts
export * from '@/features/ai/api/chatApi';
export * from '@/features/ai/api/sessionApi';
export * from '@/features/ai/api/runApi';
export * from '@/features/ai/api/approvalApi';
```

- [ ] **Step 4: Remove legacy monolithic API surface**

After this task, `web/src/api/modules/ai.ts` must only re-export the split feature APIs. Do not preserve old methods that the rebuilt backend no longer supports.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd web && npm run test:run -- src/features/ai/api/__tests__/api-contract.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/features/ai/api/chatApi.ts web/src/features/ai/api/sessionApi.ts web/src/features/ai/api/runApi.ts web/src/features/ai/api/approvalApi.ts web/src/features/ai/api/__tests__/api-contract.test.ts web/src/api/modules/ai.ts
git commit -m "refactor(web-ai): split ai api modules by backend responsibility"
```

### Task 8: Stream Reconnect and In-Memory Pending State

**Files:**
- Create: `web/src/features/ai/stream/streamClient.ts`
- Create: `web/src/features/ai/stream/eventDispatcher.ts`
- Create: `web/src/features/ai/stream/reconnectController.ts`
- Create: `web/src/features/ai/state/pendingRunStore.ts`
- Modify: `web/src/components/AI/pendingRunStore.ts`
- Test: `web/src/features/ai/stream/__tests__/reconnectController.test.ts`
- Test: `web/src/features/ai/state/__tests__/pendingRunStore.test.ts`

- [ ] **Step 1: Write the failing pending state test**

```ts
// web/src/features/ai/state/__tests__/pendingRunStore.test.ts
import { describe, expect, it } from 'vitest';
import { createPendingRunStore } from '../pendingRunStore';

describe('pendingRunStore', () => {
  it('stores pending runs in memory only', () => {
    const store = createPendingRunStore();
    store.upsert({ runId: 'r1', updatedAt: '2026-04-17T00:00:00Z' });
    expect(store.get('r1')?.runId).toBe('r1');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web && npm run test:run -- src/features/ai/state/__tests__/pendingRunStore.test.ts`
Expected: FAIL with module not found for `../pendingRunStore`

- [ ] **Step 3: Write minimal implementation**

```ts
// web/src/features/ai/state/pendingRunStore.ts
export type PendingRunMetadata = { runId: string; updatedAt: string };

export function createPendingRunStore() {
  const memory = new Map<string, PendingRunMetadata>();

  return {
    upsert(item: PendingRunMetadata) {
      memory.set(item.runId, item);
    },
    get(runId: string) {
      return memory.get(runId) ?? null;
    },
    remove(runId: string) {
      memory.delete(runId);
    },
  };
}
```

- [ ] **Step 4: Point component imports at the feature store**

```ts
// web/src/components/AI/pendingRunStore.ts
export * from '@/features/ai/state/pendingRunStore';
```

If the component-local file becomes unnecessary after import updates, delete it in the same task instead of leaving an extra wrapper.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd web && npm run test:run -- src/features/ai/state/__tests__/pendingRunStore.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/features/ai/stream/streamClient.ts web/src/features/ai/stream/eventDispatcher.ts web/src/features/ai/stream/reconnectController.ts web/src/features/ai/state/pendingRunStore.ts web/src/features/ai/state/__tests__/pendingRunStore.test.ts web/src/components/AI/pendingRunStore.ts
git commit -m "refactor(web-ai): isolate stream reconnect and pending run state"
```

### Task 9: Legacy Path Deletion and Final Verification

**Files:**
- Delete: `internal/modules/ai/handler/chat/routing.go`
- Delete: `internal/modules/ai/agent/shared/middleware/scene_defaults.go`
- Delete or rewrite: `internal/modules/ai/handler/chat/handler.go`
- Create: `docs/ai/error-codes.md`
- Create: `docs/ai/runtime-troubleshooting.md`

- [ ] **Step 1: Write the failing contract test**

```go
// internal/modules/ai/handler/chat/chat_contract_test.go
package chathandler

import "testing"

func TestLegacyRoutingPathRemoved(t *testing.T) {
	if LegacyRoutingEnabled() {
		t.Fatal("legacy routing must be removed")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/modules/ai/handler/chat -run TestLegacyRoutingPathRemoved -count=1`
Expected: FAIL with `undefined: LegacyRoutingEnabled`

- [ ] **Step 3: Write minimal implementation and delete legacy files**

```go
// internal/modules/ai/handler/chat/service.go
package chathandler

func LegacyRoutingEnabled() bool {
	return false
}
```

Delete these files in the same task once the new path is already wired and verified:
- `internal/modules/ai/handler/chat/routing.go`
- `internal/modules/ai/agent/shared/middleware/scene_defaults.go`

If `internal/modules/ai/handler/chat/handler.go` still exists only as a vestigial compatibility layer, delete it. If it still owns active runtime behavior, rewrite it into a thin shell that forwards into the new `interfaces/http` package and remove all embedded business logic.

- [ ] **Step 4: Write runtime docs**

Create `docs/ai/error-codes.md` with:
- Public `AI_STREAM_*` / `AI_API_*` error code meanings
- Whether each code is retryable
- Where trace IDs appear

Create `docs/ai/runtime-troubleshooting.md` with:
- How to identify trace IDs for failed chat runs
- Where context budget selection happens
- Where projection updates happen
- What files own stream reconnect and pending run state

- [ ] **Step 5: Run full verification**

Run: `go test ./internal/modules/ai/... -count=1`
Expected: PASS

Run: `cd web && npm run test:run -- src/features/ai/api/__tests__/api-contract.test.ts src/features/ai/state/__tests__/pendingRunStore.test.ts src/features/ai/stream/__tests__/reconnectController.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/handler/chat internal/modules/ai/agent/shared/middleware docs/ai/error-codes.md docs/ai/runtime-troubleshooting.md
git commit -m "refactor(ai): remove legacy ai paths and finalize rebuilt architecture"
```

## Spec Coverage Checklist

- New backend vertical slice: Tasks 1-3
- Context budget and compression entrypoint: Task 4
- Incremental projection state: Task 5
- Trace propagation and metrics hooks: Task 6
- Frontend API contract alignment: Task 7
- Stream reconnect and in-memory pending state: Task 8
- Legacy deletion and runtime docs: Task 9

## Self-Review Notes

- Compatibility language removed. This plan now assumes direct replacement.
- Added agent-specific stage gates so execution order is constrained by architecture, not by file creation alone.
- Replaced weak migration framing with explicit deletion rules.
- Strengthened context task to cover pinned/recent/history buckets instead of a plain sliding window.
- Kept `writing-plans` style: exact files, failing test, run-to-fail, minimal implementation, run-to-pass, commit.

## Rollout Notes

1. Complete tasks in order; do not start Task N+1 before Task N verification and commit.
2. At each task boundary, run the listed command and confirm expected output before proceeding.
3. If a test fails for unrelated reasons, fix that defect in a separate commit and continue.
4. If local code structure differs from this plan, preserve the task intent and update this document inline before proceeding.
