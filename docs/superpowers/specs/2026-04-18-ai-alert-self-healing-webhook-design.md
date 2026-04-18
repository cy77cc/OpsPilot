# 2026-04-18 AI Alert Self-Healing Webhook Design

## 1. Context

OpsPilot currently has:

- Monitoring webhook ingestion at `/api/v1/alerts/receiver` for Alertmanager alerts.
- Rebuilt AI runtime with approval middleware, tool risk policies, and async worker infrastructure.

What is missing is a dedicated AI self-healing intake path: external alert systems cannot directly trigger an AI decision flow that determines whether to auto-repair or require manual approval.

## 2. Confirmed Decisions

The following decisions are confirmed:

- Execution policy: low-risk auto-fix, high-risk actions go through approval.
- Payload compatibility: support Alertmanager payload and OpsPilot unified alert payload in the same endpoint.
- Auto-fix scope (phase 1): includes Kubernetes and host-side operations.
- Processing mode: webhook responds immediately (`202 Accepted`), execution is asynchronous.
- Idempotency key: `source + fingerprint + status`.
- Approval visibility: add global approval pool (not only per-user pending list).
- Resolved handling: mark/cancel related active healing jobs; do not execute rollback/cleanup actions.
- Webhook auth: HMAC-SHA256 signature header (`X-OpsPilot-Signature`).
- Retry strategy: exponential backoff up to 3 retries, then escalate to approval.

## 3. Goals and Non-Goals

### 3.1 Goals

1. Add a dedicated AI webhook for alert-driven self-healing.
2. Build a durable async healing job pipeline with idempotency.
3. Reuse existing AI tool risk/approval architecture to enforce guarded execution.
4. Add global pending-approval query capability with explicit permission.
5. Provide traceable state transitions and operational observability.

### 3.2 Non-Goals

1. Replace monitoring module alert ingestion and notification flow.
2. Build a generic event bus for all modules in phase 1.
3. Implement automatic rollback/compensation when alerts resolve.
4. Cover all external alert protocols beyond the two specified payload formats.

## 4. Architecture

Recommended architecture: dedicated AI webhook + async healing pipeline.

### 4.1 High-Level Flow

1. External system sends alert webhook to `/api/v1/ai/alerts/webhook`.
2. Endpoint validates signature, payload size, and protocol shape.
3. Payload is normalized and persisted as canonical ingest event (`idempotent upsert`).
4. Healing job is created (or deduplicated) and queued.
5. Background worker consumes pending jobs and starts AI decision run.
6. Decision branch:
   - auto-fix path for low risk.
   - approval path for high risk.
   - no-action path when intervention is unnecessary.
7. `resolved` events cancel/close active related jobs.

### 4.2 Module Placement

- `internal/modules/ai/interfaces/http`: webhook and query endpoints.
- `internal/modules/ai/app/command`: ingestion command handler.
- `internal/modules/ai/domain` (or model+logic split per current style): alert-heal state model and transition rules.
- `internal/modules/ai/infra/workers`: alert-heal worker runner.
- `internal/modules/ai/dao`: ingest/job/attempt persistence accessors.

## 5. Data Model

### 5.1 `ai_alert_ingest_events`

Canonical normalized event storage.

Key fields:

- `id`
- `source`
- `protocol` (`alertmanager` / `opspilot.alert.v1`)
- `fingerprint`
- `status` (`firing` / `resolved`)
- `dedupe_key` (unique)
- `severity`
- `title`
- `target`
- `labels_json`
- `annotations_json`
- `raw_payload_json`
- `starts_at`
- `ends_at`
- `received_at`
- `created_at`
- `updated_at`

Constraint:

- `UNIQUE(dedupe_key)` where `dedupe_key = source + ":" + fingerprint + ":" + status`.
- `fingerprint` is required for both supported payload formats; missing fingerprint is rejected as invalid payload.

### 5.2 `ai_alert_heal_jobs`

Persistent async healing job state.

Key fields:

- `id`
- `event_id` (FK to ingest event)
- `scene`
- `status`
- `decision`
- `retry_count`
- `max_retry` (default `3`)
- `next_retry_at`
- `last_error`
- `latest_run_id`
- `created_at`
- `updated_at`

Indexes:

- queue index on `(status, next_retry_at, created_at)`
- lookup index on `(event_id)`
- correlation index on `(scene, status)`

### 5.3 `ai_alert_heal_attempts` (recommended)

Attempt-level audit and diagnostics.

Key fields:

- `id`
- `job_id`
- `attempt_no`
- `run_id`
- `outcome`
- `error_message`
- `started_at`
- `finished_at`

## 6. State Machine

### 6.1 Job Status

Core statuses:

- `pending`
- `analyzing`
- `auto_fixing`
- `waiting_approval`
- `retry_wait`
- `succeeded`
- `no_action`
- `canceled_resolved`
- `failed_manual`

### 6.2 Transition Rules

1. `firing` ingest:
   - normalize + idempotent insert.
   - create `pending` job if no active equivalent job.
2. Worker picks `pending`/`retry_wait`:
   - move to `analyzing`.
3. Decision outcomes:
   - low-risk actionable -> `auto_fixing`.
   - high-risk actionable -> `waiting_approval`.
   - not actionable -> `no_action`.
4. Auto-fix result:
   - success -> `succeeded`.
   - failure with retry budget -> `retry_wait` (`next_retry_at` with exponential backoff).
   - failure exhausted -> `waiting_approval`.
5. `resolved` ingest:
   - mark active related jobs as `canceled_resolved`.

## 7. API Contract

### 7.1 Webhook Ingress

`POST /api/v1/ai/alerts/webhook`

Auth:

- No JWT.
- Require `X-OpsPilot-Signature` HMAC-SHA256.
- Reject if secret not configured.

Behavior:

- Validate payload size and protocol.
- Normalize and persist idempotently.
- Enqueue/dedupe healing job.
- Return immediately.

Response:

- `202 Accepted` with `{ accepted: true, event_id, job_id }`.

### 7.2 Query APIs

- `GET /api/v1/ai/alert-heal/jobs`
- `GET /api/v1/ai/alert-heal/jobs/:id`
- `POST /api/v1/ai/alert-heal/jobs/:id/retry` (required in phase 1)

Auth:

- JWT + RBAC.

### 7.3 Approval APIs

Keep existing APIs and add:

- `GET /api/v1/ai/approvals/pending/global`

This returns global pending approvals with pagination and explicit permission checks.

## 8. Approval and Risk Policy Integration

1. Reuse existing approval middleware + DB risk policy matching.
2. Reuse command class behavior for host/k8s tools.
3. Self-healing worker must choose scene explicitly by alert domain so the runtime loads the correct tool set:
   - host/domain alerts -> host-capable scene.
   - k8s/workload alerts -> kubernetes or deployment-capable scene.
4. Keep fallback safe strategy (unknown/mutation-like actions require approval).
5. Global pending approvals should not remove per-user pending endpoint; both coexist.

## 9. RBAC Permissions

Add new permission codes:

- `ai:alert:read`
- `ai:alert:write`
- `ai:approval:read`
- `ai:approval:write` (or reuse existing approval submit permission if already modeled)

`ai:approval:read` gates global pending approval list.

## 10. Error Contract

Public error codes to add:

- `AI_ALERT_WEBHOOK_INVALID_SIGNATURE`
- `AI_ALERT_WEBHOOK_UNSUPPORTED_PAYLOAD`
- `AI_ALERT_HEAL_JOB_NOT_FOUND`
- `AI_ALERT_HEAL_RETRY_EXHAUSTED`

Webhook endpoint should avoid leaking internal execution errors. Runtime and tool-level details stay in logs and persisted job/attempt diagnostics.

## 11. Observability

Metrics:

- `ai_alert_webhook_received_total{protocol,status}`
- `ai_alert_heal_job_total{decision,status}`
- `ai_alert_heal_retry_total`
- `ai_alert_heal_approval_total`
- `ai_alert_heal_decision_latency_seconds`
- `ai_alert_heal_repair_latency_seconds`

Tracing:

- propagate `trace_id` from ingress to job record and AI run.

Logs:

- structured logs by `event_id`, `job_id`, `run_id`, `approval_id`.

## 12. Testing Strategy

Unit tests:

1. Signature verification and payload limit.
2. Protocol parsing (Alertmanager + unified format).
3. Idempotency behavior (`source+fingerprint+status`).
4. State transition correctness.
5. Retry backoff and exhaustion behavior.
6. Global approval pool authorization.

Integration tests:

1. webhook -> ingest -> job -> worker -> run/approval path.
2. resolved event cancels active jobs.
3. high-risk host action transitions to approval rather than direct execution.

Acceptance criteria:

1. duplicate `firing` events do not create duplicate active healing jobs.
2. resolved event consistently closes active jobs.
3. auto-fix failure escalates to approval after 3 retries.
4. users with `ai:approval:read` can query global pending approvals.

## 13. Rollout Plan

1. Add schema + DAOs + model-level tests.
2. Add webhook endpoint and parser with contract tests.
3. Add worker loop and transition tests.
4. Add approval global query endpoint + RBAC integration.
5. Add metrics and dashboard/ops visibility hooks.
6. Staged enablement behind config flag if needed.

## 14. Open Questions Resolved in This Design

1. Processing sync vs async: async (`202` immediate return).
2. Resolved behavior: cancel/close only, no compensating action.
3. Risk policy source: reuse existing AI approval/risk policy system.
4. Scope: includes host-side operations in phase 1 under guarded policy.

## 15. Frontend Design (Alert-Centric)

This section defines the UI model after visual validation: alert remains the primary object, and AI healing is a processing track under that alert.

### 15.1 Information Architecture

1. Keep the existing observability menu entry (`/monitor`); do not add a new top-level "AI self-healing" menu.
2. Use alert list as the main entry (`/monitor/alerts`).
3. Add an alert detail sub-page (`/monitor/alerts/:alertId`) where AI healing lifecycle is shown in full context.
4. Add a quick link from alert list row to detail page.

### 15.2 Route and Page Structure

Target route shape:

- `GET /monitor/alerts` -> alert list page (main).
- `GET /monitor/alerts/:alertId` -> alert detail page with AI healing panel.

Implementation note:

- Current `MonitorPage` can be split into focused pages incrementally.
- Phase 1 should prioritize `alerts` and `alert detail` flows first; rule/channel tabs can remain unchanged temporarily.

### 15.3 Alert List UX

Add two columns:

1. `处理状态` (primary process state).
2. `自愈状态` (AI/manual handling result state).

List features:

1. Filter by `自愈状态`.
2. Quick toggle: "仅看 AI 介入告警".
3. Keep existing severity/status filtering behavior.

### 15.4 Alert Detail UX

Detail page sections:

1. Alert basic info card:
   - title, severity, source, firing/resolved status, first trigger time, latest update.
2. AI healing card:
   - current healing state badge.
   - decision summary and risk level.
   - last execution result and retry count.
   - action buttons.
3. Attempt timeline:
   - each attempt start/end/result/error.
4. Approval linkage:
   - show approval id and direct jump when in approval-required status.

### 15.5 Action Rules

Buttons:

1. `手动重试`:
   - enabled only for failed terminal states and unresolved alerts.
2. `查看审批`:
   - visible when state is approval-related.
3. `查看执行轨迹`:
   - always available when attempts exist.

Disable rules:

1. Disable retry while `auto_fixing` or `analyzing`.
2. Disable retry when alert becomes `resolved`.
3. Prevent duplicate trigger clicks (frontend optimistic lock + backend idempotency).

### 15.6 Display Vocabulary

Use a two-level status model.

Primary status (`处理状态`):

- `待处理`
- `处理中`
- `待人工`
- `已处理`

Secondary status (`自愈状态`):

- `AI自愈成功`
- `AI修复失败`
- `转人工审批`
- `AI判定无需处理`
- `告警恢复已取消`
- `自动修复中`

Design rule:

- Primary badge sits near alert title.
- Secondary tag shows AI/manual path outcome.

### 15.7 Frontend API Integration

Frontend modules to add:

- `web/src/api/modules/aiAlertHeal.ts` (or equivalent in `web/src/features/ai/api/` if feature-localized):
  - list jobs by alert.
  - get job detail by id.
  - trigger manual retry.
  - get global approvals (when authorized).

Backend endpoints consumed:

- `/api/v1/ai/alert-heal/jobs`
- `/api/v1/ai/alert-heal/jobs/:id`
- `/api/v1/ai/alert-heal/jobs/:id/retry`
- `/api/v1/ai/approvals/pending/global`

### 15.8 Permissions in UI

Frontend permission gates:

1. read pages: `ai:alert:read` (or compatible fallback permission).
2. retry action: `ai:alert:write`.
3. global approval list/link: `ai:approval:read`.

When permission is missing:

- show status read-only.
- hide action buttons and approval navigation.

### 15.9 Frontend State and Refresh

1. Alert list remains source-of-truth for alert-level status.
2. Detail page polls healing job state at a short interval while in active states.
3. When state reaches terminal (`succeeded`, `no_action`, `canceled_resolved`, `failed_manual`), downgrade polling frequency or stop.
4. Keep reconciling with backend to avoid stale optimistic UI.

### 15.10 Frontend Test Plan

Unit/component tests:

1. status badge/tag mapping and vocabulary consistency.
2. retry button enable/disable matrix.
3. approval link visibility by status and permission.
4. list filter behavior for AI-related states.

Integration/e2e tests:

1. click alert row -> open detail -> see AI healing panel.
2. failed alert allows retry and shows pending/processing transition.
3. approval-required state navigates to approval detail/pool.
4. resolved alert disables retry and shows canceled status.
