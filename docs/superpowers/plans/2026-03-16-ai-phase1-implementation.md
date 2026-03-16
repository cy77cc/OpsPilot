# AI Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Phase 1 read-only AI assistant as the primary product path, with a single chat entry, explicit run/report models, SSE streaming, and a standalone diagnosis report page.

**Architecture:** The implementation replaces the current public AI chat contract with a cleaner `session + message + run + diagnosis_report` model. Backend work lands first on persistence and HTTP/SSE contracts, then wires routing and read-only agent execution, and finally switches the frontend from drawer-centric copilot behavior to a full AI page with run recovery and report navigation.

**Implementation Notes (2026-03-16):**
- `/ai` and `/ai/diagnosis/:reportId` are implemented as protected routes for the dedicated Phase 1 flow.
- The legacy header launcher now acts as a lightweight shortcut to `/ai`.
- Legacy drawer/copilot components remain only as compatibility redirects and no longer expose a separate public chat runtime.

**Tech Stack:** Go, Gin, GORM, MySQL migrations, React 19, Vite, Ant Design, Vitest

---

## File Map

### Backend core

- Create: `internal/service/ai/routes.go`
- Create: `internal/service/ai/handler/chat.go`
- Create: `internal/service/ai/handler/session.go`
- Create: `internal/service/ai/handler/run.go`
- Create: `internal/service/ai/handler/diagnosis.go`
- Create: `internal/service/ai/deps.go`
- Create: `internal/service/ai/types.go`
- Create: `internal/ai/agents/intent/intent.go`
- Create: `internal/ai/agents/intent/types.go`
- Create: `internal/ai/agents/intent/rule_layer.go`
- Create: `internal/ai/agents/intent/model_layer.go`
- Create: `internal/ai/agents/qa/agent.go`
- Create: `internal/ai/agents/diagnosis/agent.go`
- Create: `internal/ai/agents/diagnosis/report.go`
- Create: `internal/ai/runtime/stream.go`
- Create: `internal/ai/runtime/stream_test.go`

### Persistence

- Modify: `internal/model/ai_chat.go`
- Create: `internal/model/ai_run.go`
- Create: `internal/model/ai_diagnosis_report.go`
- Create: `internal/dao/ai_chat_dao.go`
- Create: `internal/dao/ai_run_dao.go`
- Create: `internal/dao/ai_diagnosis_report_dao.go`
- Create: `storage/migrations/20260316_000039_ai_phase1_runs_and_reports.sql`

### Frontend

- Modify: `web/src/api/modules/ai.ts`
- Create: `web/src/pages/AI/Assistant/index.tsx`
- Create: `web/src/pages/AI/DiagnosisReport/index.tsx`
- Create: `web/src/components/AIAssistant/SessionList.tsx`
- Create: `web/src/components/AIAssistant/ChatTimeline.tsx`
- Create: `web/src/components/AIAssistant/Composer.tsx`
- Create: `web/src/components/AIAssistant/RunStatus.tsx`
- Create: `web/src/components/AIAssistant/DiagnosisSummaryCard.tsx`
- Create: `web/src/components/AIAssistant/DiagnosisReportView.tsx`
- Modify: `web/src/ProtectedApp.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/AI/AICopilotButton.tsx`
- Modify: `web/src/components/AI/AIAssistantDrawer.tsx`
- Modify: `web/src/components/AI/Copilot.tsx`

### Tests

- Create: `internal/service/ai/handler/chat_test.go`
- Create: `internal/service/ai/handler/session_test.go`
- Create: `internal/service/ai/handler/run_test.go`
- Create: `internal/service/ai/handler/diagnosis_test.go`
- Create: `internal/ai/agents/intent/intent_test.go`
- Create: `web/src/pages/AI/Assistant/index.test.tsx`
- Create: `web/src/pages/AI/DiagnosisReport/index.test.tsx`
- Modify: `web/src/api/modules/ai.test.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`

## Chunk 1: Persistence And Public Contracts

### Task 1: Add failing migration and model tests for Phase 1 entities

**Files:**
- Create: `storage/migrations/20260316_000039_ai_phase1_runs_and_reports.sql`
- Create: `internal/model/ai_run.go`
- Create: `internal/model/ai_diagnosis_report.go`
- Modify: `internal/model/ai_chat.go`
- Test: `internal/ai/state/chat_store_test.go`

- [x] **Step 1: Write the failing model expectations**

Document the target fields in Go structs before writing SQL:

```go
type AIRun struct {
    ID               string
    SessionID        string
    UserMessageID    string
    AssistantMessageID string
    IntentType       string
    AssistantType    string
    RiskLevel        string
    Status           string
    TraceID          string
    ErrorMessage     string
}
```

- [x] **Step 2: Add the migration with exact tables and indexes**

Create SQL for:
- `ai_runs`
- `ai_diagnosis_reports`
- any required indexes on `session_id`, `status`, and `run_id`

- [x] **Step 3: Update chat/message models only where Phase 1 needs it**

Keep `AIChatSession` and `AIChatMessage` as the conversation base. Do not add compatibility fields for old runtime contracts unless Phase 1 requires them.

- [x] **Step 4: Run migration-focused verification**

Run: `go test ./storage/migration/... ./internal/model/...`
Expected: PASS

- [x] **Step 5: Commit**

Run:

```bash
git add storage/migrations/20260316_000039_ai_phase1_runs_and_reports.sql internal/model/ai_chat.go internal/model/ai_run.go internal/model/ai_diagnosis_report.go
git commit -m "feat: add AI phase 1 run and report models"
```

### Task 2: Add DAOs for session/message/run/report

**Files:**
- Create: `internal/dao/ai_chat_dao.go`
- Create: `internal/dao/ai_run_dao.go`
- Create: `internal/dao/ai_diagnosis_report_dao.go`
- Test: `internal/service/ai/handler/session_test.go`

- [x] **Step 1: Write the failing DAO tests or handler-level expectations**

Cover:
- create/list/get/delete session
- create message timeline
- create/update/get run
- create/get report by `report_id` and by `run_id`

- [x] **Step 2: Implement minimal DAO methods**

Add focused methods only:
- `CreateSession`
- `ListSessions`
- `GetSession`
- `DeleteSession`
- `CreateMessage`
- `ListMessagesBySession`
- `CreateRun`
- `UpdateRunStatus`
- `GetRun`
- `CreateReport`
- `GetReport`
- `GetReportByRunID`

- [x] **Step 3: Run targeted tests**

Run: `go test ./internal/dao/... ./internal/service/ai/...`
Expected: PASS or only fail on not-yet-implemented handlers

- [x] **Step 4: Commit**

```bash
git add internal/dao/ai_chat_dao.go internal/dao/ai_run_dao.go internal/dao/ai_diagnosis_report_dao.go
git commit -m "feat: add AI phase 1 DAOs"
```

## Chunk 2: Backend HTTP And SSE Surface

### Task 3: Register the AI service and placeholder handlers

**Files:**
- Create: `internal/service/ai/routes.go`
- Create: `internal/service/ai/deps.go`
- Create: `internal/service/ai/types.go`
- Modify: `internal/service/service.go`
- Test: `internal/service/ai/handler/session_test.go`

- [x] **Step 1: Write the failing route registration test**

Assert these routes exist:
- `POST /api/v1/ai/chat`
- `GET /api/v1/ai/sessions`
- `POST /api/v1/ai/sessions`
- `GET /api/v1/ai/sessions/:id`
- `DELETE /api/v1/ai/sessions/:id`
- `GET /api/v1/ai/runs/:runId`
- `GET /api/v1/ai/diagnosis/:reportId`

- [x] **Step 2: Implement `internal/service/ai/routes.go` and wire it into `internal/service/service.go`**

Follow the registration style used by other service modules under `internal/service/*/routes.go`.

- [x] **Step 3: Add placeholder handlers with stable response types**

Create request/response DTOs first so the public contract is fixed before business logic expands.

- [x] **Step 4: Run tests**

Run: `go test ./internal/service/...`
Expected: route registration tests PASS

- [x] **Step 5: Commit**

```bash
git add internal/service/ai/routes.go internal/service/ai/deps.go internal/service/ai/types.go internal/service/service.go
git commit -m "feat: register AI phase 1 service routes"
```

### Task 4: Implement session and diagnosis-report handlers

**Files:**
- Create: `internal/service/ai/handler/session.go`
- Create: `internal/service/ai/handler/diagnosis.go`
- Create: `internal/service/ai/handler/session_test.go`
- Create: `internal/service/ai/handler/diagnosis_test.go`

- [x] **Step 1: Write failing handler tests**

Cover:
- create session
- list sessions
- get session with messages
- delete session
- fetch diagnosis report by `reportId`

- [x] **Step 2: Implement minimal handlers using DAOs**

Keep handlers thin. Validation and persistence orchestration live in service/dependency helpers, not in route glue.

- [x] **Step 3: Run targeted tests**

Run: `go test ./internal/service/ai/handler/...`
Expected: PASS

- [x] **Step 4: Commit**

```bash
git add internal/service/ai/handler/session.go internal/service/ai/handler/diagnosis.go internal/service/ai/handler/session_test.go internal/service/ai/handler/diagnosis_test.go
git commit -m "feat: add AI phase 1 session and report handlers"
```

### Task 5: Implement run status handler and public SSE stream contract

**Files:**
- Create: `internal/service/ai/handler/run.go`
- Create: `internal/ai/runtime/stream.go`
- Create: `internal/ai/runtime/stream_test.go`
- Create: `internal/service/ai/handler/run_test.go`

- [x] **Step 1: Write failing tests for run status and SSE event serialization**

Cover:
- `GET /api/v1/ai/runs/:runId`
- public events `init`, `intent`, `status`, `delta`, `progress`, `report_ready`, `error`, `done`
- guarantee that internal-only event names are not exposed

- [x] **Step 2: Implement the stream event envelope**

Create a small public event model in `internal/ai/runtime/stream.go`, for example:

```go
type StreamEvent struct {
    Event string `json:"event"`
    Data  any    `json:"data"`
}
```

- [x] **Step 3: Implement the run status handler**

Return:
- `run_id`
- `status`
- `assistant_type`
- `intent_type`
- `progress_summary`
- `report`

- [x] **Step 4: Run targeted tests**

Run: `go test ./internal/ai/runtime ./internal/service/ai/handler`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/service/ai/handler/run.go internal/service/ai/handler/run_test.go internal/ai/runtime/stream.go internal/ai/runtime/stream_test.go
git commit -m "feat: add AI phase 1 run status and stream contract"
```

## Chunk 3: Intent Routing And Agent Execution

### Task 6: Build the explicit Phase 1 intent router

**Files:**
- Create: `internal/ai/agents/intent/types.go`
- Create: `internal/ai/agents/intent/rule_layer.go`
- Create: `internal/ai/agents/intent/model_layer.go`
- Create: `internal/ai/agents/intent/intent.go`
- Create: `internal/ai/agents/intent/intent_test.go`
- Modify: `internal/ai/agents/router.go`

- [x] **Step 1: Write failing router tests**

Cover:
- obvious QA prompts
- obvious diagnosis prompts
- ambiguous prompts degrade to QA
- result includes `intent_type`, `assistant_type`, `risk_level`

- [x] **Step 2: Implement rule layer first**

Start with deterministic keyword and pattern matching for high-confidence diagnosis requests.

- [x] **Step 3: Implement model layer as fallback**

Wrap the existing router capabilities instead of inventing a second unrelated classification path.

- [x] **Step 4: Implement the combined router**

The exported API should be one entry point for Phase 1 consumers.

- [x] **Step 5: Run tests**

Run: `go test ./internal/ai/agents/...`
Expected: PASS

- [x] **Step 6: Commit**

```bash
git add internal/ai/agents/router.go internal/ai/agents/intent/types.go internal/ai/agents/intent/rule_layer.go internal/ai/agents/intent/model_layer.go internal/ai/agents/intent/intent.go internal/ai/agents/intent/intent_test.go
git commit -m "feat: add AI phase 1 intent router"
```

### Task 7: Add QA and diagnosis execution wrappers

**Files:**
- Create: `internal/ai/agents/qa/agent.go`
- Create: `internal/ai/agents/diagnosis/agent.go`
- Create: `internal/ai/agents/diagnosis/report.go`
- Modify: `internal/ai/tools/kubernetes/tools.go`
- Test: `internal/service/ai/handler/chat_test.go`

- [x] **Step 1: Write failing chat execution tests**

Cover:
- QA run returns streamed answer text
- diagnosis run returns progress and final report metadata
- diagnosis path uses only read-only Kubernetes tools

- [x] **Step 2: Implement QA wrapper**

Keep it focused on visible response generation. Do not expose planner internals to the frontend.

- [x] **Step 3: Implement diagnosis wrapper and report builder**

Define the exact report output shape in `internal/ai/agents/diagnosis/report.go`.

- [x] **Step 4: Run tests**

Run: `go test ./internal/ai/... ./internal/service/ai/...`
Expected: PASS or only fail on pending chat handler orchestration

- [x] **Step 5: Commit**

```bash
git add internal/ai/agents/qa/agent.go internal/ai/agents/diagnosis/agent.go internal/ai/agents/diagnosis/report.go internal/ai/tools/kubernetes/tools.go
git commit -m "feat: add AI phase 1 QA and diagnosis execution wrappers"
```

### Task 8: Implement chat orchestration handler

**Files:**
- Create: `internal/service/ai/handler/chat.go`
- Create: `internal/service/ai/handler/chat_test.go`
- Modify: `internal/service/ai/types.go`

- [x] **Step 1: Write the failing end-to-end handler tests**

Cover:
- create session on first message if missing
- create user message and run
- route to QA or diagnosis
- stream the public event set in the right order
- create assistant summary message
- create diagnosis report on success

- [x] **Step 2: Implement minimal orchestration**

Execution order:
1. resolve session
2. persist user message
3. create run
4. emit `init`
5. classify intent
6. emit `intent` and `status`
7. execute assistant path
8. persist summary/report
9. emit `done`

- [x] **Step 3: Run tests**

Run: `go test ./internal/service/ai/...`
Expected: PASS

- [x] **Step 4: Commit**

```bash
git add internal/service/ai/handler/chat.go internal/service/ai/handler/chat_test.go internal/service/ai/types.go
git commit -m "feat: add AI phase 1 chat orchestration"
```

## Chunk 4: Frontend API And New AI Pages

### Task 9: Refactor the frontend AI API client to the Phase 1 contract

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.test.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`

- [x] **Step 1: Write failing API-client tests**

Cover:
- `chatStream` consuming `init`, `intent`, `status`, `delta`, `progress`, `report_ready`, `error`, `done`
- session CRUD methods
- run status fetch
- diagnosis report fetch

- [x] **Step 2: Remove or de-emphasize old public APIs that conflict with Phase 1**

At minimum, do not let chains/approvals/execution-detail contracts remain the main path used by the UI.

- [x] **Step 3: Implement the new typed client surface**

Add explicit types for:
- `AIRun`
- `AIDiagnosisReport`
- `AIChatStreamEvent`

- [x] **Step 4: Run tests**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts
git commit -m "feat: align frontend AI API with phase 1 contract"
```

### Task 10: Build the dedicated AI Assistant page

**Files:**
- Create: `web/src/pages/AI/Assistant/index.tsx`
- Create: `web/src/components/AIAssistant/SessionList.tsx`
- Create: `web/src/components/AIAssistant/ChatTimeline.tsx`
- Create: `web/src/components/AIAssistant/Composer.tsx`
- Create: `web/src/components/AIAssistant/RunStatus.tsx`
- Create: `web/src/components/AIAssistant/DiagnosisSummaryCard.tsx`
- Create: `web/src/pages/AI/Assistant/index.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`
- Modify: `web/src/App.tsx`

- [x] **Step 1: Write failing page tests**

Cover:
- open `/ai` as a real page instead of redirect
- show session list and active conversation
- submit message and render stream updates
- show diagnosis summary card with report entry link
- restore run status after refresh

- [x] **Step 2: Implement the new page and focused child components**

Prefer small components over one large page file.

- [x] **Step 3: Wire routing**

Add:
- a real `/ai` route in `web/src/App.tsx` or `web/src/ProtectedApp.tsx`
- any required permission gating

- [x] **Step 4: Run tests**

Run: `npm run test:run -- web/src/pages/AI/Assistant/index.test.tsx`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add web/src/pages/AI/Assistant/index.tsx web/src/components/AIAssistant/SessionList.tsx web/src/components/AIAssistant/ChatTimeline.tsx web/src/components/AIAssistant/Composer.tsx web/src/components/AIAssistant/RunStatus.tsx web/src/components/AIAssistant/DiagnosisSummaryCard.tsx web/src/pages/AI/Assistant/index.test.tsx web/src/ProtectedApp.tsx web/src/App.tsx
git commit -m "feat: add AI phase 1 assistant page"
```

### Task 11: Build the diagnosis report page

**Files:**
- Create: `web/src/pages/AI/DiagnosisReport/index.tsx`
- Create: `web/src/components/AIAssistant/DiagnosisReportView.tsx`
- Create: `web/src/pages/AI/DiagnosisReport/index.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`

- [x] **Step 1: Write the failing report-page test**

Cover:
- fetch report by route param
- render summary, evidence, root causes, and recommendations
- handle missing or failed report states

- [x] **Step 2: Implement the report page**

Keep layout report-centric, not chat-centric.

- [x] **Step 3: Add route registration**

Add a protected route such as `/ai/diagnosis/:reportId`.

- [x] **Step 4: Run tests**

Run: `npm run test:run -- web/src/pages/AI/DiagnosisReport/index.test.tsx`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add web/src/pages/AI/DiagnosisReport/index.tsx web/src/components/AIAssistant/DiagnosisReportView.tsx web/src/pages/AI/DiagnosisReport/index.test.tsx web/src/ProtectedApp.tsx
git commit -m "feat: add AI phase 1 diagnosis report page"
```

## Chunk 5: Retirement Of Conflicting Old AI Paths

### Task 12: Retire drawer-first and redirect-first AI entry points

**Files:**
- Modify: `web/src/components/AI/AICopilotButton.tsx`
- Modify: `web/src/components/AI/AIAssistantDrawer.tsx`
- Modify: `web/src/components/AI/Copilot.tsx`
- Modify: `web/src/components/Layout/AppLayout.tsx`

- [x] **Step 1: Write failing behavior checks**

Cover:
- `/ai` becomes the primary entry
- old launcher points users to the new page or becomes a thin shortcut
- old drawer does not remain the main Phase 1 experience

- [x] **Step 2: Implement the minimal retirement path**

Preferred outcome:
- keep a lightweight launcher button if product wants it
- make it navigate to `/ai`
- remove duplicated chat logic from the old drawer flow

- [x] **Step 3: Run targeted frontend tests**

Run: `npm run test:run -- web/src/pages/AI/Assistant/index.test.tsx`
Expected: PASS

- [x] **Step 4: Commit**

```bash
git add web/src/components/AI/AICopilotButton.tsx web/src/components/AI/AIAssistantDrawer.tsx web/src/components/AI/Copilot.tsx web/src/components/Layout/AppLayout.tsx
git commit -m "refactor: retire legacy AI drawer-first path"
```

### Task 13: Retire conflicting backend runtime exposure

**Files:**
- Modify: `internal/ai/events/events.go`
- Modify: `internal/ai/state/chat_store.go`
- Modify: `internal/ai/state/state.go`
- Test: `internal/ai/runtime/stream_test.go`

- [x] **Step 1: Write failing regression tests**

Assert that Phase 1 public consumers no longer depend on internal chain/thought/runtime event names.

- [x] **Step 2: Remove or isolate legacy event dependencies**

Keep internal event richness only if it stays behind the public SSE contract adapter.

- [x] **Step 3: Run tests**

Run: `go test ./internal/ai/...`
Expected: PASS

- [x] **Step 4: Commit**

```bash
git add internal/ai/events/events.go internal/ai/state/chat_store.go internal/ai/state/state.go internal/ai/runtime/stream_test.go
git commit -m "refactor: isolate legacy AI runtime from phase 1 public contract"
```

## Chunk 6: End-To-End Verification

### Task 14: Run backend verification

**Files:**
- No code changes required unless failures are found

- [x] **Step 1: Run backend tests**

Run: `go test ./internal/ai/... ./internal/service/ai/... ./internal/dao/... ./storage/migration/...`
Expected: PASS

- [x] **Step 2: Fix the smallest failing unit first**

Do not batch speculative fixes.

- [x] **Step 3: Re-run the exact failing command**

Run the same command until PASS.

- [x] **Step 4: Commit if fixes were required**

```bash
git add <files you changed>
git commit -m "test: fix AI phase 1 backend regressions"
```

### Task 15: Run frontend verification

**Files:**
- No code changes required unless failures are found

- [x] **Step 1: Run targeted frontend tests**

Run: `npm run test:run -- web/src/api/modules/ai.test.ts web/src/api/modules/ai.streamChunk.test.ts web/src/pages/AI/Assistant/index.test.tsx web/src/pages/AI/DiagnosisReport/index.test.tsx`
Expected: PASS

- [x] **Step 2: Build the frontend**

Run: `npm run build`
Expected: PASS

- [x] **Step 3: Fix only confirmed failures**

Keep fixes small and scoped.

- [x] **Step 4: Commit if fixes were required**

```bash
git add <files you changed>
git commit -m "test: fix AI phase 1 frontend regressions"
```

### Task 16: Final cleanup and implementation handoff

**Files:**
- Modify: `docs/superpowers/specs/2026-03-16-ai-phase1-design.md`
- Modify: `docs/superpowers/plans/2026-03-16-ai-phase1-implementation.md`

- [x] **Step 1: Update the spec and plan if implementation changed a contract**

Only document real deviations.

- [x] **Step 2: Capture the final removed paths**

List which old AI entry points, event names, or models are no longer public.

- [x] **Step 3: Run final smoke verification**

Run:
- `go test ./internal/service/ai/...`
- `npm run test:run -- web/src/pages/AI/Assistant/index.test.tsx`

Expected: PASS

- [x] **Step 4: Commit the handoff**

```bash
git add docs/superpowers/specs/2026-03-16-ai-phase1-design.md docs/superpowers/plans/2026-03-16-ai-phase1-implementation.md
git commit -m "docs: finalize AI phase 1 implementation handoff"
```
