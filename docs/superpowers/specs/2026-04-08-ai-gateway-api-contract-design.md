# AI Gateway and API Contract Design

- Date: 2026-04-08
- Parent: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Inherits:
  - `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-copilot-ui-interaction-design.md`
- References:
  - `api/ai/v1/ai.go`
  - `web/src/api/modules/ai.ts`
- Scope: public AI gateway contract for HTTP, SSE, approval, replay, content, and debug access
- Goal: define the protocol projection boundary so the gateway exposes derived read models and command entry points without becoming the canonical runtime truth

## 1. Inheritance From Blueprint

This document inherits the `Copilot + Runtime` boundary from the parent blueprint.

Hard boundary:

- the gateway is a protocol projection layer
- canonical session, task, run, turn, tool, approval, and event truth stays internal
- the gateway translates requests into use cases and projects derived responses back out
- the gateway must not invent state from assistant prose, UI state, or stream fragments

If this document conflicts with the blueprint or the sibling runtime/projection specs on ownership, lifecycle, cursor semantics, or approval resume behavior, the upstream specs win.

Explicit terminology:

- `entry API` means a request that starts or resumes a user interaction
- `inspection API` means a read-only request over run state or derived artifacts
- `projection cursor` means a derived page cursor over replay blocks
- `event cursor` means the canonical persisted `event_id` cursor used for SSE attach and replay continuation

## 2. Endpoint Inventory

The current gateway surface is a compatibility projection over the internal AI module. It is split into user entry, run inspection, approval, replay/content, and debug surfaces.

| Surface | Method | Path | Purpose | Auth boundary | Cursor / idempotency note |
| --- | --- | --- | --- | --- | --- |
| Entry | `POST` | `/ai/chat` | Start or continue an interaction and stream SSE updates | JWT required | `client_request_id` is the logical idempotency key; `last_event_id` is the event cursor for attach/reconnect |
| Entry | `GET` | `/ai/sessions` | List sessions for the current user | JWT required | Optional `scene` filter only |
| Entry | `POST` | `/ai/sessions` | Create a new session shell | JWT required | No server-side idempotency key is exposed in the current contract |
| Entry | `GET` | `/ai/sessions/{id}` | Fetch one session and its message history | JWT required | Read-only |
| Entry | `DELETE` | `/ai/sessions/{id}` | Delete a session and its messages | JWT required | Read-only from the perspective of retries; repeated deletes should be safe to retry |
| Run inspection | `GET` | `/ai/runs/{runId}` | Fetch run status and linked report metadata | JWT required | Read-only |
| Replay / inspection | `GET` | `/ai/runs/{runId}/projection` | Fetch derived replay projection blocks | JWT required | `cursor` is a secondary projection paging token; canonical replay continuation uses `last_event_id` |
| Replay / content | `GET` | `/ai/run-contents/{id}` | Fetch lazily stored run content by content id | JWT required | Read-only |
| Debug | `GET` | `/ai/diagnosis/{reportId}` | Fetch a diagnosis report generated from a run | JWT required | Read-only |
| Approval | `GET` | `/ai/approvals/pending` | List pending approvals for the current user | JWT required | Read-only |
| Approval | `GET` | `/ai/approvals/{id}` | Fetch one approval snapshot | JWT required | Read-only |
| Approval | `POST` | `/ai/approvals/{id}/submit` | Submit an approval decision | JWT required | `Idempotency-Key` is required for safe retries |
| Approval | `POST` | `/ai/approvals/{id}/retry-resume` | Requeue a retryable approval resume path | JWT required | `trigger_id` is the dedupe key for the retry request |

The inventory above is the public gateway contract. Any endpoint not listed here is outside the gateway spec until another spec explicitly adds it.

## 3. Entry APIs

Entry APIs are the only user-facing write paths that can create or continue AI work.

### 3.1 `POST /ai/chat`

This is the main collaboration entry point.

Request body fields:

- `session_id`
- `client_request_id`
- `last_event_id`
- `message`
- `scene`
- `context`

Rules:

1. `message` is required.
2. `client_request_id` is the logical idempotency key for a user request retry.
3. `last_event_id` is the canonical event cursor for reattach or reconnect.
4. `scene` is routing context, not truth ownership.
5. `context` is opaque request context and must not be treated as canonical runtime state.

Transport behavior:

- the endpoint returns `text/event-stream`
- the request may include `last_event_id` in the JSON body, in the `last_event_id` query parameter, or in the `Last-Event-ID` header
- header precedence is highest, then query parameter, then request body
- if JSON binding fails before the stream starts, the gateway returns a normal JSON error envelope instead of an SSE frame

### 3.2 `GET /ai/sessions`

Returns the current user session list.

Rules:

- `scene` is an optional filter
- the gateway returns summary rows only
- this endpoint does not expose canonical run truth; it exposes a user-facing collaboration list

### 3.3 `POST /ai/sessions`

Creates a new session shell.

Rules:

- `title` and `scene` are required in the current v1 contract
- the response is a session summary projection, not a canonical session record with all internal runtime details

### 3.4 `GET /ai/sessions/{id}` and `DELETE /ai/sessions/{id}`

These endpoints are session lifecycle helpers only.

Rules:

- reads are ownership-checked against the current user
- deletion is a collaboration-container delete, not a canonical event rewrite
- deleting a session must not mutate historical canonical events for runs that already completed

## 4. Run Inspection APIs

Run inspection APIs expose durable run state and do not own execution.

### 4.1 `GET /ai/runs/{runId}`

This endpoint returns the current run snapshot.

The response should expose:

- run identity
- session linkage
- user message linkage
- assistant message linkage when present
- run status
- intent and assistant type
- risk level
- trace id
- progress summary
- linked diagnosis report metadata when present

Contract rules:

1. The payload is a read model.
2. It must not be treated as the runtime source of truth.
3. It may include linked report summary data, but the report remains a separate artifact.

### 4.2 `GET /ai/run-contents/{id}`

This endpoint fetches large payloads that are intentionally not embedded into the run snapshot or projection.

Use cases:

- tool arguments
- tool output bodies
- large executor content
- replay hydration

Contract rule:

- the content id is the durable handle; the body is a derived storage artifact, not the run truth model

### 4.3 `GET /ai/diagnosis/{reportId}`

This endpoint fetches a debug report derived from a run.

Contract rule:

- the report is a diagnostic read model, not a runtime control input
- it may be missing when the run has no diagnosis artifact

## 5. Approval APIs

Approval APIs expose the human-in-the-loop boundary.

### 5.1 `GET /ai/approvals/pending`

Returns pending approvals owned by the current user.

Rules:

- the list is a derived work queue
- each item is a frozen operation snapshot, not a live mutable tool call
- the frontend may use the list to recover from a stale approval card, but the backend remains canonical

### 5.2 `GET /ai/approvals/{id}`

Returns one approval snapshot.

The approval payload should expose:

- `approval_id`
- `checkpoint_id`
- `session_id`
- `run_id`
- `tool_name`
- `tool_call_id`
- `arguments_json`
- `preview_json`
- `status`
- decision metadata such as `approved_by`, `disapprove_reason`, `comment`, `decided_at`

Contract rules:

1. `approval_id` is the canonical public identifier.
2. `tool_call_id` is part of the frozen snapshot, not the primary lookup contract.
3. The approval snapshot must not be rewritten after decision except through a new approval record.
4. `GET /ai/approvals/{id}` is the canonical hydration path for approval details referenced by list rows or SSE cards.

### 5.2.1 Canonical External Naming And Alias Mapping

The canonical external HTTP approval response names are:

- `tool_call_id`
- `arguments_json`
- `preview_json`

This naming applies to approval read responses and approval list items exposed through the gateway.

Compatibility mapping rules:

1. Internal or legacy backend fields `call_id`, `arguments`, and `preview` map to the canonical external names above.
2. If both canonical and legacy names are present, the canonical names win.
3. The gateway may continue to accept legacy aliases during migration, but new client contracts must read the canonical names.
4. `approval_id`, `checkpoint_id`, `session_id`, `run_id`, `tool_name`, and `status` remain stable across the migration.
5. SSE `tool_approval` transport may still carry `call_id` and `preview`, but it is only a partial transport projection and must be hydrated through `GET /ai/approvals/{id}` before any canonical approval handling.

### 5.3 `POST /ai/approvals/{id}/submit`

This is the decision write path.

Rules:

- `approved` is required
- `disapprove_reason` is used when rejection needs a reason
- `comment` is optional user context
- `Idempotency-Key` is required to make retries safe

Idempotency rules:

1. Replaying the same `Idempotency-Key` for the same approval must return the same decision result.
2. The backend must not apply the transition twice.
3. The stored decision snapshot must remain stable across retries.

### 5.4 `POST /ai/approvals/{id}/retry-resume`

This endpoint requeues a retryable resume path after approval processing.

Rules:

- `trigger_id` is required
- `trigger_id` is the dedupe key for the retry request
- the request is not a new approval decision; it is a worker retry signal
- the endpoint is only for non-terminal worker recovery
- if the underlying run has reached a terminal failure or the resume failure is irrecoverable, the caller must start a fresh run through the normal run-entry path instead of retry-resume

Idempotency rule:

- repeated retry requests with the same `trigger_id` should collapse onto the same retry outcome for the same approval task

## 6. Replay And Content APIs

Replay and content APIs expose derived inspection state and large payload hydration.

### 6.1 `GET /ai/runs/{runId}/projection`

This is the replay projection endpoint.

Pagination and cursor rules:

1. `last_event_id` is the authoritative canonical replay and attach cursor for live SSE and canonical continuation.
2. `cursor` is a secondary projection paging token for this derived block list.
3. The projection `cursor` must never be treated as the canonical replay cursor.
4. The backend may encode the projection cursor however it chooses, but the token is opaque and must map deterministically to a single derived page boundary.
5. If `cursor` is omitted, the first page is returned.
6. `limit` is optional.
7. The backend clamps page size to the supported range.
8. Invalid cursors return a parameter error, not silent fallback.

Projection paging semantics:

- `ack_event_id` is the highest canonical persisted `event_id` represented in the returned page
- `has_more` indicates another page is available
- `next_cursor` is returned only when more blocks remain
- `next_cursor` is a secondary page token only; it does not replace `last_event_id`
- if the client has fully rendered the page, it should persist `ack_event_id` as the canonical continuity marker for live SSE or later replay attach
- if the client wants the next projection page, it should persist `next_cursor`
- if `ack_event_id` is missing because the page contains no canonical event-bearing rows, the client must keep the previous canonical continuity marker and must not synthesize one

Boundary rule:

- do not use the projection cursor as a reconnect cursor for SSE
- do not describe the projection cursor as the canonical replay cursor in client code or docs
- do not use `next_cursor` as the canonical attach cursor; that role belongs to `ack_event_id` and then to `last_event_id` for SSE attach

### 6.2 `GET /ai/run-contents/{id}`

This endpoint is the only public hydration path for large replay content blobs in the current contract.

Rules:

- content is fetched by content id
- the response is read-only
- the endpoint must not be used to infer the canonical order of events

## 7. Evaluation And Debug APIs

The current public contract exposes debug read models, not a separate evaluation write surface.

### 7.1 `GET /ai/diagnosis/{reportId}`

Use this to inspect generated diagnosis output.

### 7.2 No Public Evaluation Mutation API Yet

The evaluation harness remains an internal design boundary.

Contract rule:

- do not add ad hoc evaluation mutation endpoints to the gateway surface
- if a public evaluation API is needed later, it needs its own spec and migration plan

## 8. SSE Event Contract

The SSE contract is a transport projection of canonical runtime events.

### 8.1 Public Event Family

The current public event family is:

- `meta`
- `agent_handoff`
- `delta`
- `tool_call`
- `tool_result`
- `tool_approval`
- `run_state`
- `done`
- `error`
- `ops_plan_updated`
- `plan`
- `replan`

Only these transport names are in the public whitelist. Any other runtime event remains internal.

`replan` is reserved for newly created run attempts. If a legacy transport path still emits `replan`, it is a compatibility shim and must be interpreted as a new-run signal only, not as same-run refinement.

### 8.2 Transport Rules

1. SSE is derived transport, not canonical truth.
2. The gateway must preserve canonical ordering semantics when replaying or tailing.
3. The gateway must not fabricate an SSE event that cannot be derived from persisted runtime facts.
4. `event_id` on a payload becomes the SSE frame `id` field and is removed from the JSON payload body.
5. Frames without a concrete persisted `event_id` are summary-only transport frames and must not be treated as canonical cursor advances.

### 8.3 Event Payload Contract

Current payload shapes are:

- `meta`: `session_id`, `run_id`, `turn`
- `agent_handoff`: `from`, `to`, `intent`
- `delta`: `content`, optional `agent`
- `ops_plan_updated`: `todos`, optional `run_id`, optional `session_id`, optional `runtime`, optional `snapshot`
- `plan`: `iteration`, `steps`
- `replan`: `iteration`, `completed`, `is_final`, `steps`
- `tool_call`: `call_id`, `tool_name`, `arguments`
- `tool_result`: `call_id`, `tool_name`, `content`, optional `status`, optional `agent`
- `tool_approval`: `approval_id`, `target_id`, `call_id`, `tool_name`, `preview`, `timeout_seconds`
- `run_state`: `status`, optional `agent`
- `done`: `run_id`, `status`, optional `summary`, optional `iterations`
- `error`: `run_id`, `message`, optional `code`, optional `recoverable`

Rules:

1. `tool_approval` is a partial projection of the frozen approval snapshot.
2. `run_state` is a derived state snapshot, not the source of truth.
3. `done` and `error` are terminal transport summaries.
4. `event_id` is optional on the transport payload, but when present it is the canonical attach cursor for that frame.
5. `ops_plan_updated` carries the current ops todo snapshot for compatibility with the frontend activity stream.
6. `plan` is the projected ordered step list for the current run or turn.
7. `replan` is the projected label for a newly created run attempt; same-run refinements use `plan` or refinement-oriented read-model updates, not `replan`.
8. A materially new attempt is a new run in canonical truth; the `replan` transport label does not create or replace run identity by itself.
9. The `tool_approval` payload is not a lossless representation of approval state and must be hydrated through `GET /ai/approvals/{id}` before any canonical approval decision is taken.

### 8.4 SSE Cursor Rules

1. The attach cursor for live SSE is `last_event_id`.
2. `last_event_id` is a canonical persisted `event_id` pointer for the run.
3. Reattach resumes strictly after that `event_id`.
4. Unknown, expired, or inconsistent cursors must fail explicitly.
5. Cursor expiry must not trigger a silent full replay fallback.

### 8.5 SSE Error Rules

If the stream cannot be established or attach validation fails, the gateway should emit a transport error or return a normal JSON error envelope before streaming begins.

Current known stream error shape:

- `code`: `AI_STREAM_CURSOR_EXPIRED`
- `message`: human-safe refresh instruction

### 8.6 Compatibility Provenance Discipline

Compatibility events may be useful for rendering, but only canonical persisted events and canonical event ids can drive replay continuity.

| Event | Provenance discipline | Replay-authoritative | Allowed use | Forbidden use |
| --- | --- | --- | --- | --- |
| `meta` | Emit only when run/session identity is known; do not synthesize a fake continuation id | yes, if backed by persisted event id | initial stream framing | replacing `last_event_id` |
| `agent_handoff` | Derive from persisted agent-transfer facts when available; otherwise treat as best-effort UI enrichment only | no, if synthesized | display current agent routing hints | driving replay continuity or resume decisions |
| `ops_plan_updated` | Best-effort projection of runtime todo snapshots | no | UI activity refresh | canonical run progression or attach cursor advancement |
| `plan` | Derive from canonical route/turn facts | yes | current step display | replacing run state truth |
| `replan` | Compatibility shim only; maps to new-run semantics when emitted | yes only for a fresh run attempt | historical compatibility rendering | same-run refinement or continuity repair |
| `tool_call` | Derive from canonical tool selection facts | yes | tool-call display and replay | inventing tool execution that was never persisted |
| `tool_result` | Derive from canonical tool result facts | yes | tool outcome display and replay | overwriting tool-call identity |
| `tool_approval` | Partial projection of the approval snapshot | no, until hydrated through `GET /ai/approvals/{id}` | preview card and pending-state rendering | canonical approval decisions or resume binding |
| `run_state` | Derive from canonical run lifecycle facts | yes | live status display | bypassing canonical event replay or tail rules |
| `done` | Terminal summary projection | yes, if backed by persisted terminal event id | completion badge and final summary | continuing the run or advancing cursor without an event id |
| `error` | Terminal/failure summary projection | yes, if backed by persisted terminal event id | failure badge and human-safe error summary | pretending a transient error is a canonical state change |

Best-effort enrichments in this table are non-replay-authoritative and are forbidden from acting as canonical continuity drivers.

## 9. API-Level Idempotency

Idempotency is different from pagination and different from cursor replay.

### 9.1 Chat

`client_request_id` is the user-request idempotency key.

Rules:

1. The same user request should keep the same `client_request_id`.
2. Retrying the same logical request should not create two distinct logical runs when the backend can match the key.
3. `last_event_id` does not provide idempotency; it only provides replay continuity.

### 9.2 Approval Submit

`Idempotency-Key` is mandatory on `POST /ai/approvals/{id}/submit`.

Rules:

1. The first successful decision wins.
2. Replaying the same key must not reapply the decision transition.
3. The response should remain stable across retries.

### 9.3 Retry Resume

`trigger_id` deduplicates retryable resume requests.

Rules:

1. A retry request is not a new decision.
2. Repeating the same `trigger_id` should not enqueue duplicate work.
3. A different trigger id for an already-resolving approval may conflict.

### 9.4 Read APIs

All `GET` endpoints are naturally idempotent.

## 10. Error Model

The gateway uses the repo-wide `httpx.Response` envelope for JSON responses.

### 10.1 JSON Error Envelope

Shape:

- `code`
- `msg`
- `data` when present

Important business codes:

- `1000` success
- `2000` parameter error
- `2003` unauthorized
- `2004` forbidden
- `2005` not found
- `3000` server error

Rules:

1. REST endpoints should use the JSON envelope for normal errors.
2. `NotFound` should use business code `2005`.
3. Validation failures should use business code `2000`.
4. Ownership failures should use business code `2004`.
5. Auth middleware failures may return `401` or `403` at the transport layer before the business envelope is created.

### 10.2 SSE Error Envelope

SSE error frames use the transport `error` event with a compact JSON body.

The body should carry:

- `code`
- `message`
- `recoverable`
- `run_id` when available

Rules:

1. SSE errors are for live stream consumers.
2. SSE errors do not replace the JSON envelope for non-stream APIs.
3. Cursor expiry should be explicit and recoverable.

## 11. Auth Boundaries

### 11.1 User AI Surface

All `/api/v1/ai/*` user routes require JWT authentication.

Access control rules:

- session, run, content, diagnosis, and approval reads must be owned by the authenticated user
- chat requests may carry a project header for routing context, but that header does not override ownership
- no gateway endpoint may expose another user’s session, run, approval, or content without an ownership check

### 11.2 Admin Surface

`/api/v1/admin/ai/*` is a separate control plane.

Rules:

- it requires JWT plus Casbin permission checks
- it is not part of the user-facing gateway contract in this document
- model-management endpoints are separate from the collaboration and runtime gateway

### 11.3 Ownership Checks

Ownership checks must happen on the server.

Do not rely on client filtering for:

- sessions
- runs
- approvals
- run content
- diagnosis reports

## 12. Backward-Incompatible Changes

This contract intentionally draws a hard line between the current compatibility surface and the future canonical gateway.

Deliberate breaks and non-goals:

1. Do not introduce a new `/ai/approvals/{id}/decision` alias as the primary contract. The current stable write path is `/submit`.
2. Do not expose `/ai/approvals/{id}/confirm`, `/ai/approvals/{id}/resume`, or `/ai/chains/...` approval routes. They are legacy forms and must stay deleted.
3. Do not add `/ai/runs/{runId}/stream` as a new public entrypoint. The live stream stays attached through `POST /ai/chat`.
4. Do not treat the projection `cursor` as equivalent to the SSE `last_event_id`.
5. Do not allow `tool_call_id` to replace `approval_id` as the canonical public approval identifier.
6. Do not make `run_state` or UI cards the canonical source of truth.
7. Do not add new public mutation endpoints for evaluation or debug without a separate spec.

Compatibility guidance:

- existing SSE event names should stay stable unless a coordinated breaking change is approved elsewhere
- existing JSON field names should be preserved where possible
- any future alias endpoint must be documented as a migration shim, not as a new canonical surface

## 13. Migration And Deletion Guidance

The current web client contains several AI helper methods that are intentionally absent from the backend. They are compatibility stubs, not gateway contract.

Keep as current gateway contract:

- `POST /ai/chat`
- `GET /ai/sessions`
- `POST /ai/sessions`
- `GET /ai/sessions/{id}`
- `DELETE /ai/sessions/{id}`
- `GET /ai/runs/{runId}`
- `GET /ai/runs/{runId}/projection`
- `GET /ai/run-contents/{id}`
- `GET /ai/diagnosis/{reportId}`
- `GET /ai/approvals/pending`
- `GET /ai/approvals/{id}`
- `POST /ai/approvals/{id}/submit`
- `POST /ai/approvals/{id}/retry-resume`

Delete or keep out of this gateway spec:

- `GET /ai/sessions/current`
- `POST /ai/sessions/{id}/branch`
- `PUT /ai/sessions/{id}`
- `GET /ai/capabilities`
- `GET /ai/tools/{name}/params/hints`
- `POST /ai/tools/preview`
- `POST /ai/tools/execute`
- `GET /ai/executions/{id}`
- `POST /ai/feedback`
- `POST /ai/confirmations/{id}/confirm`
- `GET /ai/scene/{scene}/tools`
- `GET /ai/scene/{scene}/prompts`
- `GET /ai/usage/stats`
- `GET /ai/usage/logs`

Migration rule:

- if any of the deleted helper endpoints are reintroduced, they need a separate contract spec and a deliberate compatibility story

## 14. Related Specs

- `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
- `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
- `docs/superpowers/specs/2026-04-08-ai-copilot-ui-interaction-design.md`
