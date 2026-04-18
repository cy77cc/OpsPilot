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
- `POST /api/v1/ai/alert-heal/jobs/:id/retry` (optional in phase 1)

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
