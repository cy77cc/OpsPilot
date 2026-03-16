# AI Phase 1 Design

Date: 2026-03-16
Status: Draft for review
Scope: OpsPilot AI assistant Phase 1

## 1. Goal

Phase 1 delivers a production-usable read-only AI assistant MVP for Kubernetes troubleshooting. Users enter through a single AI chat page, ask either knowledge or diagnosis questions in natural language, and receive either a streamed answer or a generated diagnosis report.

This phase is intentionally broader than a backend-only MVP. The design covers backend APIs, SSE streaming, routing, run/report modeling, and the first complete frontend experience.

## 2. Design Principles

The following constraints are fixed for this phase:

- Documentation wins over current implementation.
- If current AI structures conflict with the Phase 1 target design, they should be replaced or removed rather than preserved for compatibility.
- Phase 1 uses a single chat entry point. QA and diagnosis are routed automatically by the backend.
- Diagnosis output is centered on a standalone report page. The chat timeline shows a short summary and a link to the report instead of embedding the full report body.
- Phase 1 is read-only. No write tools, approval flows, or mutation-capable operations are in scope.

## 3. Product Scope

### In Scope

- AI assistant chat page
- Session create/list/detail/delete
- Intent routing between QA and diagnosis
- Streaming chat response over SSE
- Read-only diagnosis execution using Kubernetes read tools
- Standalone diagnosis report page
- Run status query and recovery after refresh/reconnect

### Out of Scope

- Approval flow
- Kubernetes write operations
- Multi-cluster switching inside a conversation
- Advanced risk scoring
- Hybrid tasks that combine diagnosis and change execution

## 4. Recommended Architecture

Phase 1 uses a single chat entry point with backend intent routing and two execution paths.

1. The user sends a message from the AI Assistant page.
2. The backend creates or resolves a session, persists the user message, creates a run record, and classifies the request as `qa` or `diagnosis`.
3. QA requests stream visible answer content back to the chat page.
4. Diagnosis requests stream progress updates and produce a structured diagnosis report at completion.
5. The chat timeline stores only a concise assistant summary plus a report entry point. The full diagnosis content is read from the report page.

This architecture keeps the user experience simple while separating conversation, execution, and report rendering concerns.

## 5. Data Model

Phase 1 should model four primary entities.

### 5.1 Session

`session` represents conversation metadata:

- `id`
- `user_id`
- `title`
- `scene`
- `created_at`
- `updated_at`

The session is the conversation container, not the execution state holder.

### 5.2 Message

`message` represents user-visible chat timeline entries:

- user message
- assistant answer summary
- assistant diagnosis summary with report link

The message list is the primary data source for rendering the chat timeline.

### 5.3 Run

`run` represents one AI request lifecycle. This is the key new domain object for Phase 1 and must be explicit in the design.

Recommended fields:

- `id`
- `session_id`
- `user_message_id`
- `assistant_message_id`
- `intent_type`
- `assistant_type`
- `risk_level`
- `status`
- `trace_id`
- `error_message`
- `started_at`
- `finished_at`
- `created_at`
- `updated_at`

Responsibilities:

- represent one QA or diagnosis execution
- power run status lookup
- support SSE reconnect/recovery
- link chat activity to diagnosis reports

`run` must not be conflated with tool execution logs.

### 5.4 Diagnosis Report

`diagnosis_report` is created only for successful diagnosis runs.

Recommended fields:

- `id`
- `run_id`
- `session_id`
- `summary`
- `impact_scope`
- `suspected_root_causes`
- `evidence`
- `recommendations`
- `raw_tool_refs`
- `status`
- `created_at`
- `updated_at`

The report page reads directly from this entity. The chat page only renders a lightweight projection of it.

## 6. Migration Strategy From Current AI Structures

Phase 1 is a replacement design, not a compatibility layer.

### Keep

- session and message persistence as the foundation of conversation history, with field changes if needed
- read-only Kubernetes tools
- RAG and existing model capabilities when they do not conflict with the target architecture
- execution logging for audit and cost tracking

### Replace or Remove

- treating current `turn/block` structures as the primary public model for Phase 1 chat and diagnosis rendering
- treating `AIExecution` as the equivalent of a run
- exposing old SSE/runtime event contracts that leak internal orchestration details into the frontend

### Result

After Phase 1, the primary external model should be:

- session
- message
- run
- diagnosis_report

Current internal structures may survive only if they are implementation details and do not shape the public contract.

## 7. Backend Design

### 7.1 Main API Surface

Recommended endpoints:

- `POST /api/v1/ai/chat`
- `GET /api/v1/ai/sessions`
- `POST /api/v1/ai/sessions`
- `GET /api/v1/ai/sessions/:id`
- `DELETE /api/v1/ai/sessions/:id`
- `GET /api/v1/ai/runs/:runId`
- `GET /api/v1/ai/diagnosis/:reportId`

### 7.2 Chat Flow

`POST /api/v1/ai/chat` should:

1. validate request and resolve session
2. create the user message
3. create a run
4. route the intent
5. invoke QA or diagnosis execution
6. stream visible events through SSE
7. persist the assistant summary message
8. persist the diagnosis report if diagnosis succeeds

### 7.3 Intent Routing

The router returns:

- `intent_type`
- `assistant_type`
- `risk_level`
- optional extracted target metadata

Phase 1 routes only between:

- `qa`
- `diagnosis`

If classification fails, the request should degrade to `qa` rather than hard fail.

### 7.4 Run Status

`GET /api/v1/ai/runs/:runId` should return:

- current status
- assistant type
- intent type
- progress summary
- failure reason if any
- linked report metadata if available

This endpoint is required so the frontend can recover from refresh or SSE interruption.

## 8. SSE Contract

Phase 1 should use a deliberately small public event set.

Recommended events:

- `init`
- `intent`
- `status`
- `delta`
- `progress`
- `report_ready`
- `error`
- `done`

Event responsibilities:

- `init`: emits `session_id` and `run_id`
- `intent`: emits routed assistant type
- `status`: emits lifecycle state such as `queued`, `running`, `completed`, or `failed`
- `delta`: emits only user-visible answer text
- `progress`: emits diagnosis-phase progress summaries
- `report_ready`: emits report identity and summary payload
- `error`: emits recoverable or terminal failure
- `done`: emits final completion metadata

Internal orchestration events should not become part of the public frontend contract unless they are intentionally productized.

## 9. Frontend Design

### 9.1 Primary Entry

The user enters through a single AI Assistant page.

Main regions:

- session list
- active chat timeline
- composer
- current run status area

### 9.2 Chat Timeline

The chat page renders:

- user messages
- streamed QA responses
- diagnosis summaries
- report entry cards

The chat page should not render the full structured diagnosis report body inline as the primary experience.

### 9.3 Diagnosis Report Page

The diagnosis report page is the main result surface for diagnosis requests.

Recommended sections:

- summary
- impact scope
- suspected root causes
- evidence
- recommendations
- report metadata

### 9.4 Recovery Behavior

If the page refreshes during a running diagnosis, the frontend should use `run_id` to restore state and continue showing progress or completion outcome.

## 10. Error Handling

Phase 1 should prefer visible and recoverable failure modes.

- intent routing failure degrades to `qa`
- tool or diagnosis failure marks the run failed or partially failed with a visible explanation
- SSE interruption triggers status recovery through `GET /runs/:runId`
- report generation failure must be surfaced explicitly and must not be shown as success

## 11. Testing Strategy

### Backend

- route and handler tests
- intent routing tests
- run lifecycle tests
- diagnosis report persistence tests
- SSE event ordering and payload tests

### Frontend

- chat submit and stream rendering tests
- session switching tests
- run recovery tests after refresh
- diagnosis summary to report navigation tests
- failure state rendering tests

### Integration Success Criteria

- users can complete QA or read-only diagnosis from one chat page
- every diagnosis request creates a run
- every successful diagnosis exposes a standalone report page
- frontend can recover run state after refresh
- old conflicting AI contracts are no longer the primary Phase 1 path

## 12. Implementation Sequence

Recommended implementation order:

1. Define the target public contracts for session, message, run, report, and SSE.
2. Add or revise persistence models and migrations to fit the target contracts.
3. Implement backend handlers and intent routing flow.
4. Implement diagnosis report generation and report retrieval.
5. Build the AI Assistant page and session management UI.
6. Build run recovery and report page UI.
7. Remove or retire conflicting old Phase 1 paths.
8. Complete integration and regression testing.

## 13. Open Decisions Resolved In This Design

The following decisions are fixed by this document:

- Phase 1 covers both backend and frontend.
- The product uses one chat entry point.
- Diagnosis is primarily represented by a standalone report page.
- Conflicting old AI structures should be replaced rather than preserved.
- Run modeling is explicit and distinct from execution logging.
