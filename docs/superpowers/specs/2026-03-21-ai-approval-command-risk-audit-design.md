# AI Approval Command Risk Audit Design

- Date: 2026-03-21
- Scope: `tool_call` approval only
- Goal: Ship a resilient approval gate quickly, with recovery-first semantics, while leaving clear extension points for a future risk-approval agent

## 1. Context And Decision Summary

This design focuses only on approval gating for high-risk `tool_call` executions. It intentionally does not migrate all stream events to an outbox pipeline in this phase.

Chosen constraints:
- Approvals apply only to high-risk tools.
- Risk evaluation source of truth is a DB policy table, not hardcoded tool metadata.
- Delivery prioritizes practical reliability over perfect first-pass correctness: failures are acceptable if recovery is deterministic and auditable.
- Architecture must reserve extension points for a future dedicated risk-approval agent.

## 2. Architecture

### 2.1 Components

1. `RiskPolicyStore` (new)
- Reads risk policy from DB.
- Matches by `tool_name`, optionally refined by `scene` and `command_class`.
- Returns `approval_required`, `risk_level`, `policy_version`, and `matched_rule_id`.
- On lookup failure, enforces safe fallback (`approval_required=true`) for risk-sensitive tools.

2. `ApprovalOrchestrator` (new)
- Single entrypoint for runtime approval decisions.
- API concept: `Evaluate(toolCallContext) -> decision`.
- Current decision source: `RiskPolicyStore` only.
- Future extension: optional second-stage evaluator (`RiskApprovalAgent`) without changing callers.

3. `ApprovalTaskService` (refactor)
- Owns task lifecycle and invariants:
  - `CreateFromInterrupt`
  - `Decide`
  - `CanResume`
  - `BuildResumeParams`
- Enforces decision-source integrity (resume params must come from persisted decision state).

4. `ResumeWorker` (new)
- Consumes approval decision events asynchronously.
- Calls `ResumeWithParams` only when persisted task status is resumable (`approved`).
- Persists projected events and run convergence updates.
- Supports retry with bounded backoff.

### 2.2 Boundary Rules

- Handlers do not directly run resume execution after decision submission.
- `SubmitApproval` is write-only for decision state.
- Runtime resume side effects happen in `ResumeWorker` only.
- Rejected or expired decisions are terminal and non-resumable.

## 3. Data Model And Contracts

### 3.1 Risk Policy Table (new)

Suggested table: `ai_tool_risk_policies`

Fields:
- `id`
- `tool_name` (required)
- `scene` (nullable)
- `command_class` (nullable)
- `argument_rules` (json/jsonb, nullable; argument-sensitive policy match rules)
- `approval_required` (bool)
- `risk_level` (`low|medium|high|critical`)
- `priority` (int; higher wins for overlaps)
- `enabled` (bool)
- `policy_version` (string/int)
- `created_at`, `updated_at`

Matching semantics:
- Filter by `enabled=true`.
- Exact `tool_name` match required.
- `scene`/`command_class` are exact if present, wildcard when null.
- `argument_rules` (when present) must match request arguments (exact match and/or regex strategy defined by policy engine contract).
- Highest `priority` wins.
- For same priority, prefer the more specific rule (argument-aware > command_class > scene-only > tool-only).

### 3.2 Approval Task Snapshot (existing table extension)

`ai_approval_tasks` must include policy snapshot fields for deterministic replay:
- `matched_rule_id`
- `policy_version`
- `risk_level`
- `decision_source` (e.g. `db_policy`)
- `expires_at` (approval TTL boundary)
- `locked_at` (state-machine lock timestamp once decision is accepted for processing)
- `lock_owner` (worker id or logical consumer id)

This ensures resumed execution remains tied to the policy that triggered approval, even if policy rows change later.

### 3.3 Outbox Event Table (new)

Suggested table: `ai_approval_outbox_events`

Fields:
- `id`
- `event_type` (`approval_requested|approval_decided|resume_started|resume_finished|resume_failed`)
- `approval_id`
- `run_id`
- `session_id`
- `payload_json`
- `status` (`pending|processing|done|failed`)
- `retry_count`
- `next_retry_at`
- `created_at`, `updated_at`

Idempotency key:
- `(approval_id, event_type, retry_count window)` plus worker-side dedupe on terminal status.

## 4. Execution Flow

### 4.1 Tool Call Interception

1. Agent emits `tool_call`.
2. `ApprovalOrchestrator` evaluates risk via `RiskPolicyStore`.
3. If approval not required, execution proceeds.
4. If approval required:
- trigger interrupt,
- create approval task with checkpoint/tool/run/session/user and policy snapshot,
- append outbox event `approval_requested`.

### 4.2 User Decision

1. Frontend submits approve/reject via `SubmitApproval`.
2. Service validates ownership and task state.
3. Service updates task status (`approved/rejected/expired`) idempotently.
4. Transition to `approved` must atomically set lock fields (`locked_at`, `lock_owner`) so later user mutations are rejected.
5. Once locked for resume processing, task becomes immutable to further user decisions.
6. Service appends outbox event `approval_decided`.

### 4.3 Asynchronous Resume

1. `ResumeWorker` consumes `approval_decided`.
2. Service checks `CanResume`.
3. Worker checks task TTL. If `now() > expires_at`, transition to `expired` and do not resume.
4. If `approved`:
- append `resume_started`,
- build resume params from persisted decision,
- execute `ResumeWithParams`,
- persist projected events and final convergence,
- append `resume_finished`.
5. If `rejected/expired`:
- mark non-resumable terminal outcome,
- no resume call.
6. On retryable failures:
- append `resume_failed`,
- increase retry counter and schedule backoff.

## 5. State Machine

Approval task states:
- `pending` -> `approved`
- `pending` -> `rejected`
- `pending` -> `expired`
- `approved` -> `approved_locked` (internal processing lock before resume execution)
- terminal: `approved|rejected|expired`

Run statuses (approval-related subset):
- `running`
- `waiting_approval`
- `resuming` (optional explicit transitional status)
- `completed|completed_with_tool_errors`
- `resume_failed_retryable`

Invariants:
- `rejected` and `expired` never transition to resume execution.
- Resume params are derived from persisted task decision only.
- Ownership validation applies on both decision write and worker resume processing.
- `waiting_approval` runs must not be finalized as `completed` before resume outcome.
- Once approval task is locked for worker processing, user-facing decision mutation is forbidden.
- Expired approval tasks must never be resumed, even if previously marked approved.

## 6. Error Handling And Recovery

- Policy read failure: safe fallback to approval required for sensitive tool paths.
- Duplicate decisions: return current task status; no hard failure.
- Double-submit race: decision transition and lock acquisition must be atomic; second writer receives immutable-state response.
- Resume execution failures: retriable with bounded backoff, terminally visible as `resume_failed_retryable`.
- Stream transport failures: do not lose durable progress; event persistence remains source of truth.
- Policy drift after task creation: task snapshot governs resume behavior.
- Stale approval context: worker expires tasks beyond `expires_at` instead of attempting resume.

## 7. Testing Strategy (Minimum Viable Coverage)

1. Policy evaluation
- Matches DB rules by priority.
- Defaults to approval-required on fallback path.

2. Interrupt and task creation
- High-risk tool call emits interrupt and creates task snapshot.

3. Authorization and decision integrity
- Non-owner cannot submit approval decision.
- Resume ignores request-side decision overrides and uses persisted status.

4. Resume gate correctness
- `rejected/expired` cannot resume.
- `approved` resumes and converges run state.

5. Worker reliability
- Retry scheduling on transient failures.
- Idempotent behavior under duplicate outbox deliveries.

6. API contract
- Frontend uses `/ai/approvals/:id/submit` and no direct synchronous resume side effects from submit call.

## 8. Rollout Plan

### P0
- Enforce core invariants:
  - no fake completion when waiting approval,
  - no decision override on resume,
  - ownership check on decision submit.

### P1
- Implement policy-table-driven approval evaluation.
- Add approval task snapshots and outbox event creation.
- Introduce worker-driven resume flow.

### P2
- Harden retries, observability, and full coverage tests.
- Add explicit extension contract for future `RiskApprovalAgent` stage.

## 9. Future Extension: Risk Approval Agent

Reserved extension point: `ApprovalOrchestrator` can invoke a second-stage evaluator after DB policy match.

Planned contract:
- Input: tool metadata, arguments summary, matched policy snapshot, runtime context.
- Output: `allow|require_manual|deny`, with rationale and confidence.
- Safety rule: agent can only tighten controls by default unless explicitly configured for controlled relaxations.

This keeps current implementation stable while enabling gradual intelligent risk adjudication later.
