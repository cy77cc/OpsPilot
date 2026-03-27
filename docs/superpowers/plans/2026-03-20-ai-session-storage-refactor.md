# AI Session Storage Refactor Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the old `message.content + runtime_json` persistence model with `event log + run projection + content store`, then switch history reads to the new run-based model and remove the old message-runtime path.

**Architecture:** Persist every run event into `ai_run_events` as the single source of truth, generate a run-scoped `projection_json` in `ai_run_projections`, and store lazy-loadable large bodies in `ai_run_contents`. History reads must prefer persisted projections, rebuild from events when projections are missing or invalid, and fall back to plain text only for pre-refactor legacy data.

**Tech Stack:** Go, Gin, GORM, SQL migrations, Vitest, React, TypeScript, Ant Design X

---

## File Map

### Existing files to modify

- `storage/migrations/20260319_0001_add_runtime_json_to_messages.sql`
  Remove or supersede the old runtime-json migration with a new migration path that creates new run storage tables and drops obsolete runtime storage.
- `storage/migration/dev_auto.go`
  Register new migrations in development auto-run flow.
- `internal/model/ai.go`
  Add models for `AIRunEvent`, `AIRunProjection`, `AIRunContent`; remove `RuntimeJSON` from `AIChatMessage`.
- `internal/dao/ai/run_dao.go`
  Extend run data access where needed for projection status and run summaries.
- `internal/dao/ai/chat_dao.go`
  Remove message-runtime responsibilities; keep message shell persistence only.
- `internal/service/ai/logic/logic.go`
  Change chat write path to persist events first, build projection/content artifacts, and stop writing message runtime snapshots.
- `internal/service/ai/handler/session.go`
  Remove message-runtime endpoint logic and change session/history responses to new run-centric reads.
- `internal/service/ai/routes.go`
  Wire new run/projection/content routes and remove old message-runtime route.
- `web/src/api/modules/ai.ts`
  Replace message-runtime API types with run projection/content API types.
- `web/src/components/AI/CopilotSurface.tsx`
  Load history by run projection rather than by message runtime.
- `web/src/components/AI/historyRuntime.ts`
  Delete or replace old message-runtime hydration logic.
- `web/src/components/AI/historyRuntime.test.ts`
  Delete or rewrite tests around the new projection hydration path.

### New files to create

- `storage/migrations/20260320_0002_refactor_ai_session_storage.sql`
  Create `ai_run_events`, `ai_run_projections`, `ai_run_contents`, migrate message schema, and drop obsolete runtime storage.
- `internal/dao/ai/run_event_dao.go`
  CRUD for `ai_run_events`.
- `internal/dao/ai/run_event_dao_test.go`
  DAO tests for ordered event writes and lookups.
- `internal/dao/ai/run_projection_dao.go`
  CRUD for `ai_run_projections`.
- `internal/dao/ai/run_projection_dao_test.go`
  DAO tests for projection upsert, read, and status handling.
- `internal/dao/ai/run_content_dao.go`
  CRUD for `ai_run_contents`.
- `internal/dao/ai/run_content_dao_test.go`
  DAO tests for text/json content storage.
- `internal/ai/runtime/event_types.go`
  Strongly typed event payload structs keyed by event type.
- `internal/ai/runtime/event_types_test.go`
  Tests for payload encode/decode by event type.
- `internal/ai/runtime/projection_builder.go`
  Deterministic builder that converts ordered events into the new run projection structure.
- `internal/ai/runtime/projection_builder_test.go`
  Tests for executor chunk splitting, tool nesting, summary inlining, and error states.
- `internal/service/ai/handler/projection.go`
  Handlers for `GET /runs/:id/projection` and `GET /run-contents/:id`.
- `internal/service/ai/handler/projection_test.go`
  HTTP tests for projection reads, fallback rebuild, and lazy content loading.
- `web/src/components/AI/historyProjection.ts`
  Frontend history hydration from run projections plus lazy content fetch.
- `web/src/components/AI/historyProjection.test.ts`
  Tests for projection hydration and legacy fallback.

### Existing tests to update

- `internal/service/ai/logic/logic_test.go`
- `internal/service/ai/handler/session_test.go`
- `internal/service/ai/handler/chat_test.go`
- `internal/dao/ai/chat_dao_test.go`
- `web/src/api/modules/ai.test.ts`
- `web/src/api/modules/ai.streamChunk.test.ts`

## Implementation Order

Implement in this order only:

1. Add schema and models for the new storage objects.
2. Add strong event payload typing and projection builder tests.
3. Change chat write path to persist events and projections.
4. Add run projection/content read APIs with rebuild compensation.
5. Switch frontend history reads to run projections.
6. Remove old message-runtime storage and APIs.
7. Run verification and update docs/tests.

## Chunk 1: Schema And Persistence Backbone

### Task 1: Create database schema for new run storage

**Files:**
- Create: `storage/migrations/20260320_0002_refactor_ai_session_storage.sql`
- Modify: `storage/migration/dev_auto.go`
- Test: `internal/dao/ai/run_event_dao_test.go`
- Test: `internal/dao/ai/run_projection_dao_test.go`
- Test: `internal/dao/ai/run_content_dao_test.go`

- [ ] **Step 1: Write the failing DAO migration smoke test**

```go
func TestAIRunStorageTablesExist(t *testing.T) {
	db := newTestDB(t)
	if !db.Migrator().HasTable("ai_run_events") {
		t.Fatal("expected ai_run_events table")
	}
	if !db.Migrator().HasTable("ai_run_projections") {
		t.Fatal("expected ai_run_projections table")
	}
	if !db.Migrator().HasTable("ai_run_contents") {
		t.Fatal("expected ai_run_contents table")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/dao/ai -run TestAIRunStorageTablesExist -v`
Expected: FAIL because the new tables do not exist yet.

- [ ] **Step 3: Write the migration**

Create `storage/migrations/20260320_0002_refactor_ai_session_storage.sql` with:

```sql
CREATE TABLE ai_run_events (
  id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  seq INT NOT NULL,
  event_type VARCHAR(32) NOT NULL,
  agent_name VARCHAR(64) DEFAULT '',
  tool_call_id VARCHAR(64) DEFAULT '',
  payload_json LONGTEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_ai_run_events_run_seq (run_id, seq),
  KEY idx_ai_run_events_session_created (session_id, created_at),
  KEY idx_ai_run_events_tool_call_id (tool_call_id),
  KEY idx_ai_run_events_run_type (run_id, event_type)
);

CREATE TABLE ai_run_projections (
  id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  version INT NOT NULL DEFAULT 1,
  status VARCHAR(32) NOT NULL,
  projection_json LONGTEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_ai_run_projections_run_id (run_id),
  KEY idx_ai_run_projections_session_id (session_id)
);

CREATE TABLE ai_run_contents (
  id VARCHAR(64) PRIMARY KEY,
  run_id VARCHAR(64) NOT NULL,
  session_id VARCHAR(64) NOT NULL,
  content_kind VARCHAR(32) NOT NULL,
  encoding VARCHAR(16) NOT NULL,
  summary_text VARCHAR(500) DEFAULT '',
  body_text LONGTEXT,
  body_json LONGTEXT,
  size_bytes BIGINT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_ai_run_contents_run_id (run_id),
  KEY idx_ai_run_contents_session_id (session_id),
  KEY idx_ai_run_contents_kind (content_kind)
);

ALTER TABLE ai_chat_messages DROP COLUMN runtime_json;
```

- [ ] **Step 4: Register the migration**

Add the new filename in `storage/migration/dev_auto.go` using the same pattern as existing AI migrations.

- [ ] **Step 5: Run DAO migration smoke test**

Run: `go test ./internal/dao/ai -run TestAIRunStorageTablesExist -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add storage/migrations/20260320_0002_refactor_ai_session_storage.sql storage/migration/dev_auto.go internal/dao/ai/run_event_dao_test.go internal/dao/ai/run_projection_dao_test.go internal/dao/ai/run_content_dao_test.go
git commit -m "feat(ai): add run storage schema"
```

### Task 2: Add new GORM models and DAOs

**Files:**
- Modify: `internal/model/ai.go`
- Modify: `internal/dao/ai/chat_dao.go`
- Create: `internal/dao/ai/run_event_dao.go`
- Create: `internal/dao/ai/run_event_dao_test.go`
- Create: `internal/dao/ai/run_projection_dao.go`
- Create: `internal/dao/ai/run_projection_dao_test.go`
- Create: `internal/dao/ai/run_content_dao.go`
- Create: `internal/dao/ai/run_content_dao_test.go`
- Test: `internal/dao/ai/chat_dao_test.go`

- [ ] **Step 1: Write the failing DAO behavior tests**

```go
func TestRunEventDAO_ListByRunOrdered(t *testing.T) {}
func TestRunProjectionDAO_UpsertAndGetByRunID(t *testing.T) {}
func TestRunContentDAO_CreateAndGet(t *testing.T) {}
func TestChatDAO_DoesNotExposeRuntimeJSON(t *testing.T) {}
```

- [ ] **Step 2: Run the new DAO tests to verify failure**

Run: `go test ./internal/dao/ai -run 'TestRunEventDAO|TestRunProjectionDAO|TestRunContentDAO|TestChatDAO_DoesNotExposeRuntimeJSON' -v`
Expected: FAIL because models and DAOs do not exist yet.

- [ ] **Step 3: Add the new models**

Update `internal/model/ai.go` with:

```go
type AIRunEvent struct {
	ID         string         `gorm:"column:id;type:varchar(64);primaryKey"`
	RunID      string         `gorm:"column:run_id;type:varchar(64);not null;index:idx_ai_run_events_run_type,priority:1"`
	SessionID  string         `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_run_events_session_created,priority:1"`
	Seq        int            `gorm:"column:seq;not null;uniqueIndex:uk_ai_run_events_run_seq,priority:2"`
	EventType  string         `gorm:"column:event_type;type:varchar(32);not null;index:idx_ai_run_events_run_type,priority:2"`
	AgentName  string         `gorm:"column:agent_name;type:varchar(64)"`
	ToolCallID string         `gorm:"column:tool_call_id;type:varchar(64);index"`
	PayloadJSON string        `gorm:"column:payload_json;type:longtext;not null"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index"`
}
```

Add corresponding `AIRunProjection` and `AIRunContent` structs. Remove `RuntimeJSON` from `AIChatMessage`.

- [ ] **Step 4: Implement minimal DAOs**

Implement:

```go
func (d *AIRunEventDAO) Create(ctx context.Context, event *model.AIRunEvent) error
func (d *AIRunEventDAO) ListByRun(ctx context.Context, runID string) ([]model.AIRunEvent, error)

func (d *AIRunProjectionDAO) Upsert(ctx context.Context, projection *model.AIRunProjection) error
func (d *AIRunProjectionDAO) GetByRunID(ctx context.Context, runID string) (*model.AIRunProjection, error)

func (d *AIRunContentDAO) Create(ctx context.Context, content *model.AIRunContent) error
func (d *AIRunContentDAO) Get(ctx context.Context, id string) (*model.AIRunContent, error)
```

- [ ] **Step 5: Update ChatDAO tests and implementation**

Remove `runtime_json` writes and reads from `internal/dao/ai/chat_dao.go` and its tests.

- [ ] **Step 6: Run DAO tests**

Run: `go test ./internal/dao/ai -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/model/ai.go internal/dao/ai/chat_dao.go internal/dao/ai/chat_dao_test.go internal/dao/ai/run_event_dao.go internal/dao/ai/run_event_dao_test.go internal/dao/ai/run_projection_dao.go internal/dao/ai/run_projection_dao_test.go internal/dao/ai/run_content_dao.go internal/dao/ai/run_content_dao_test.go
git commit -m "feat(ai): add run storage models and daos"
```

## Chunk 2: Event Typing And Projection Builder

### Task 3: Add strongly typed event payloads

**Files:**
- Create: `internal/ai/runtime/event_types.go`
- Create: `internal/ai/runtime/event_types_test.go`

- [ ] **Step 1: Write the failing payload round-trip tests**

```go
func TestDecodeEventPayload_ToolCall(t *testing.T) {}
func TestDecodeEventPayload_Delta(t *testing.T) {}
func TestDecodeEventPayload_RejectsUnknownShape(t *testing.T) {}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/ai/runtime -run 'TestDecodeEventPayload' -v`
Expected: FAIL because typed payload definitions do not exist.

- [ ] **Step 3: Define payload structs by event type**

Implement in `internal/ai/runtime/event_types.go`:

```go
type EventType string

const (
	EventTypeMeta        EventType = "meta"
	EventTypeAgentHandoff EventType = "agent_handoff"
	EventTypePlan        EventType = "plan"
	EventTypeReplan      EventType = "replan"
	EventTypeDelta       EventType = "delta"
	EventTypeToolCall    EventType = "tool_call"
	EventTypeToolResult  EventType = "tool_result"
	EventTypeDone        EventType = "done"
	EventTypeError       EventType = "error"
)

type MetaPayload struct { RunID string; SessionID string; Turn int }
type DeltaPayload struct { Agent string; Content string }
type ToolCallPayload struct { Agent string; CallID string; ToolName string; Arguments map[string]any }
type ToolResultPayload struct { Agent string; CallID string; ToolName string; Content string; Status string }
```

Add typed encode/decode helpers:

```go
func MarshalEventPayload(eventType EventType, payload any) (string, error)
func UnmarshalEventPayload(eventType EventType, raw string) (any, error)
```

- [ ] **Step 4: Run runtime payload tests**

Run: `go test ./internal/ai/runtime -run 'TestDecodeEventPayload' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/runtime/event_types.go internal/ai/runtime/event_types_test.go
git commit -m "feat(ai): add typed run event payloads"
```

### Task 4: Build the run projection builder with executor chunking

**Files:**
- Create: `internal/ai/runtime/projection_builder.go`
- Create: `internal/ai/runtime/projection_builder_test.go`

- [ ] **Step 1: Write failing projection builder tests**

```go
func TestBuildProjection_SplitsExecutorContentOnToolCall(t *testing.T) {}
func TestBuildProjection_NestsToolResultUnderToolCall(t *testing.T) {}
func TestBuildProjection_InlinesSummary(t *testing.T) {}
func TestBuildProjection_MarksNonSteadyStatus(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/ai/runtime -run 'TestBuildProjection' -v`
Expected: FAIL because builder does not exist.

- [ ] **Step 3: Implement minimal projection types and builder**

Create types such as:

```go
type RunProjection struct {
	Version   int               `json:"version"`
	RunID     string            `json:"run_id"`
	SessionID string            `json:"session_id"`
	Status    string            `json:"status"`
	Summary   *ProjectionSummary `json:"summary,omitempty"`
	Blocks    []ProjectionBlock `json:"blocks"`
}
```

Add a builder:

```go
func BuildProjection(events []model.AIRunEvent) (*RunProjection, []*model.AIRunContent, error)
```

Behavior:

- merge consecutive executor deltas into a single `content` item
- flush content item on `tool_call`
- attach `tool_result` to the matching `tool_call.result`
- inline summary content
- preserve `event_id` or start/end event references

- [ ] **Step 4: Run builder tests**

Run: `go test ./internal/ai/runtime -run 'TestBuildProjection' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ai/runtime/projection_builder.go internal/ai/runtime/projection_builder_test.go
git commit -m "feat(ai): add run projection builder"
```

## Chunk 3: Backend Write Path And Compensation Reads

### Task 5: Refactor chat write path to persist events and projections

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `internal/service/ai/handler/chat_test.go`

- [ ] **Step 1: Write failing logic tests for new write path**

```go
func TestChat_PersistsOrderedRunEvents(t *testing.T) {}
func TestChat_PersistsProjectionAndContentsOnCompletion(t *testing.T) {}
func TestChat_DoesNotWriteMessageRuntimeJSON(t *testing.T) {}
func TestChat_ProjectionVisibleAfterDone(t *testing.T) {}
```

- [ ] **Step 2: Run logic tests to verify failure**

Run: `go test ./internal/service/ai/logic -run 'TestChat_' -v`
Expected: FAIL because logic still writes the old runtime snapshot.

- [ ] **Step 3: Introduce event persistence during stream consumption**

Refactor `logic.go` so every projected stream event writes a typed `AIRunEvent` record before any in-memory projection aggregation is considered complete.

Use a helper such as:

```go
func (l *Logic) appendRunEvent(ctx context.Context, runID, sessionID string, seq int, eventType airuntime.EventType, payload any) error
```

Within a single run execution, instantiate an in-memory counter owned by that run handler only:

```go
seqCounter := 0
nextSeq := func() int {
	seqCounter++
	return seqCounter
}
```

Every persisted event must consume `nextSeq()` exactly once. Do not derive `seq` from the frontend or from wall-clock time.

If the streaming path is ever parallelized, this helper must be guarded so `seq` stays strictly increasing for the current `run_id`.

- [ ] **Step 4: Replace message runtime snapshot writes**

Delete old logic that marshals `projector.GetPersistedState()` into `assistantMessage.RuntimeJSON`.

At run completion:

1. load ordered events for the run
2. build projection and contents
3. persist `ai_run_projections`
4. persist `ai_run_contents`
5. update assistant message summary text
6. update run status

These completion writes must run inside one database transaction:

```go
err := db.Transaction(func(tx *gorm.DB) error {
	// upsert projection
	// insert contents
	// update assistant message summary
	// update run status
	return nil
})
```

If any write fails, roll back the entire completion transaction and rely on the projection rebuild path to compensate on reads.

- [ ] **Step 5: Handle read-after-write consistency explicitly**

Make projection persistence happen before the terminal `done` event is emitted whenever possible. If projection creation fails, do not block `done`; rely on the rebuild path to compensate.

For high-frequency `delta` traffic, keep the design correct first, but document a bounded optimization path:

- `plan`, `tool_call`, `tool_result`, `done`, `error`, `agent_handoff` must flush immediately
- `delta` events may be micro-batched in memory if write amplification becomes a bottleneck
- batching windows must stay small and deterministic, for example `500ms` or `10` delta payloads
- any pending delta batch must flush before a non-delta event is written

Do not batch across runs, and do not batch in a way that changes event order.

- [ ] **Step 6: Run backend logic tests**

Run: `go test ./internal/service/ai/logic -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go internal/service/ai/handler/chat_test.go
git commit -m "feat(ai): persist run events and projections"
```

### Task 6: Add projection and content read APIs with rebuild fallback

**Files:**
- Create: `internal/service/ai/handler/projection.go`
- Create: `internal/service/ai/handler/projection_test.go`
- Modify: `internal/service/ai/handler/handler.go`
- Modify: `internal/service/ai/routes.go`
- Modify: `internal/service/ai/handler/session.go`
- Modify: `internal/service/ai/handler/session_test.go`

- [ ] **Step 1: Write failing handler tests**

```go
func TestGetRunProjection_ReturnsPersistedProjection(t *testing.T) {}
func TestGetRunProjection_RebuildsFromEventsWhenProjectionMissing(t *testing.T) {}
func TestGetRunProjection_RebuildsWhenProjectionStatusNotSteady(t *testing.T) {}
func TestGetRunContent_ReturnsLazyPayload(t *testing.T) {}
func TestGetMessageRuntimeRouteRemoved(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test ./internal/service/ai/handler -run 'TestGetRunProjection|TestGetRunContent|TestGetMessageRuntimeRouteRemoved' -v`
Expected: FAIL because handlers and routes do not exist yet.

- [ ] **Step 3: Implement projection read handler**

Add:

```go
func (h *Handler) GetRunProjection(c *gin.Context)
func (h *Handler) GetRunContent(c *gin.Context)
```

`GetRunProjection` must:

1. read `ai_run_projections`
2. detect missing/invalid/non-steady projection
3. rebuild from ordered `ai_run_events`
4. return rebuilt projection immediately
5. asynchronously upsert rebuilt projection

Protect the rebuild path against concurrent cache-miss storms. Use one of these approaches:

1. preferred: `singleflight` keyed by `run_id`
2. minimum fallback: rely on `uk_ai_run_projections_run_id` plus idempotent upsert semantics

Do not allow N concurrent requests for the same missing projection to trigger N full rebuilds.

- [ ] **Step 4: Remove old message-runtime endpoint**

Delete `GetMessageRuntime` from `session.go`, remove the route, and update tests accordingly.

- [ ] **Step 5: Run handler tests**

Run: `go test ./internal/service/ai/handler -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/handler/projection.go internal/service/ai/handler/projection_test.go internal/service/ai/handler/handler.go internal/service/ai/handler/session.go internal/service/ai/handler/session_test.go internal/service/ai/routes.go
git commit -m "feat(ai): add projection read apis"
```

## Chunk 4: Frontend History Switch And Legacy Fallback

### Task 7: Switch frontend API client to run projection/content endpoints

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.test.ts`

- [ ] **Step 1: Write failing API client tests**

```ts
it('fetches run projection by run id', async () => {})
it('fetches run content by content id', async () => {})
it('does not call getMessageRuntime', async () => {})
```

- [ ] **Step 2: Run API tests to verify failure**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts`
Expected: FAIL because new API methods are absent.

- [ ] **Step 3: Add run projection/content types and methods**

Add in `web/src/api/modules/ai.ts`:

```ts
export interface AIRunProjection { /* blocks, summary, status */ }
export interface AIRunContent { id: string; content_kind: string; body_text?: string; body_json?: unknown }

getRunProjection(runId: string)
getRunContent(contentId: string)
```

Remove `getMessageRuntime`.

- [ ] **Step 4: Run API tests**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.test.ts
git commit -m "feat(ai): add run projection api client"
```

### Task 8: Replace history runtime hydration with projection hydration

**Files:**
- Create: `web/src/components/AI/historyProjection.ts`
- Create: `web/src/components/AI/historyProjection.test.ts`
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Delete: `web/src/components/AI/historyRuntime.ts`
- Delete: `web/src/components/AI/historyRuntime.test.ts`

- [ ] **Step 1: Write failing frontend hydration tests**

```ts
it('hydrates assistant history from run projection blocks', async () => {})
it('lazy loads executor content bodies by content id', async () => {})
it('falls back to plain markdown for legacy messages without projection', async () => {})
```

- [ ] **Step 2: Run tests to verify failure**

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts`
Expected: FAIL because history projection hydration does not exist.

- [ ] **Step 3: Implement projection hydration**

In `historyProjection.ts`, add helpers such as:

```ts
export async function loadRunProjection(runId: string): Promise<AIRunProjection | null>
export async function loadRunContent(contentId: string): Promise<AIRunContent | null>
export function hydrateAssistantHistoryFromProjection(input: { message: AIMessage; projection?: AIRunProjection | null }): XChatMessage
```

Lazy content loading must be explicit:

1. inline `summary` renders immediately
2. lightweight projection metadata renders immediately
3. long `executor` content and long `tool_result` bodies load when their block first becomes visible or when the user expands that block
4. repeated expansion must reuse cached content instead of refetching

- [ ] **Step 4: Update CopilotSurface history path**

Change `defaultMessages` in `CopilotSurface.tsx` to:

1. fetch session messages and run summaries
2. fetch run projection for assistant run
3. hydrate from projection
4. if no projection/events exist, fallback to plain text content

- [ ] **Step 5: Delete old history runtime path**

Delete `historyRuntime.ts` and `historyRuntime.test.ts`.

- [ ] **Step 6: Run frontend history tests**

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts web/src/api/modules/ai.streamChunk.test.ts`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web/src/components/AI/historyProjection.ts web/src/components/AI/historyProjection.test.ts web/src/components/AI/CopilotSurface.tsx web/src/api/modules/ai.streamChunk.test.ts
git rm web/src/components/AI/historyRuntime.ts web/src/components/AI/historyRuntime.test.ts
git commit -m "feat(ai): switch history view to run projections"
```

## Chunk 5: Cleanup, Verification, And Documentation

### Task 9: Remove old backend runtime snapshot code paths

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/handler/session.go`
- Modify: `internal/model/ai.go`
- Modify: `internal/dao/ai/chat_dao_test.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write the failing cleanup regression tests**

```go
func TestLegacyRuntimeJSONIsNotWrittenAnywhere(t *testing.T) {}
func TestMessageRuntimeRouteUnavailable(t *testing.T) {}
```

- [ ] **Step 2: Run cleanup tests to verify failure**

Run: `go test ./internal/service/ai/... ./internal/dao/ai/... -run 'TestLegacyRuntimeJSONIsNotWrittenAnywhere|TestMessageRuntimeRouteUnavailable' -v`
Expected: FAIL if any old path remains.

- [ ] **Step 3: Remove old code completely**

Delete:

- any `runtime_json` field access
- any `GetMessageRuntime` references
- any history message runtime cache logic
- any tests asserting assistant message runtime snapshot persistence

- [ ] **Step 4: Run focused cleanup tests**

Run: `go test ./internal/service/ai/... ./internal/dao/ai/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/ai.go internal/service/ai/logic/logic.go internal/service/ai/handler/session.go internal/dao/ai/chat_dao_test.go internal/service/ai/logic/logic_test.go
git commit -m "refactor(ai): remove message runtime snapshot flow"
```

### Task 10: Full verification and final docs sync

**Files:**
- Modify: `docs/superpowers/specs/2026-03-20-ai-session-storage-refactor-design.md`
- Modify: `docs/superpowers/plans/2026-03-20-ai-session-storage-refactor.md`

- [ ] **Step 1: Run backend verification**

Run: `go test ./internal/ai/runtime ./internal/dao/ai ./internal/service/ai/... -v`
Expected: PASS.

- [ ] **Step 2: Run frontend verification**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts web/src/components/AI/historyProjection.test.ts`
Expected: PASS.

- [ ] **Step 3: Run targeted integration verification**

Run: `go test ./internal/service/ai/handler -run 'TestGetRunProjection|TestGetRunContent' -v`
Expected: PASS with rebuild compensation covered.

- [ ] **Step 4: Update spec and plan checkboxes**

Record any implementation-driven adjustments, but do not broaden scope beyond the approved spec.

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/specs/2026-03-20-ai-session-storage-refactor-design.md docs/superpowers/plans/2026-03-20-ai-session-storage-refactor.md
git commit -m "docs(ai): sync session storage refactor docs"
```

## Notes For The Implementer

- Keep payload typing strict. `payload_json` is storage format only; code must use typed Go structs per event type.
- Do not reintroduce `runtime_json` or any message-scoped runtime API as a shortcut.
- Prefer synchronous projection persistence before sending terminal `done`, but keep rebuild compensation as the correctness safety net.
- Use a transaction for completion-time writes so projection, contents, assistant summary, and run status commit atomically.
- Keep a per-run `seqCounter` owned by the active logic instance; do not derive `seq` from request retries or frontend state.
- If delta write amplification becomes visible in profiling, micro-batch only delta events and flush before any state-transition event.
- Guard projection rebuilds against concurrent misses with `singleflight` or an equally strict deduplication mechanism.
- Keep old-data fallback minimal: render plain text only, hide steps/tools, and avoid rebuilding fake structure from unreliable legacy snapshots.
- When in doubt, bias toward simpler projection shapes over feature-rich UI metadata. The spec already defines the minimum needed protocol.
