# AI Tool Approval Unification Design

Date: 2026-03-22
Status: Proposed
Owner: AI Runtime / Tooling

## 1. Background

Current approval behavior is inconsistent across layers:

- `host_exec_readonly` may return `status=suspended` from tool layer without producing a real approval interrupt.
- Runtime may then classify subsequent errors as terminal and emit `EXECUTION_FAILED`, causing the run to end with "生成中断，请稍后重试。"
- Some agents mount approval middleware consistently (`change`, `diagnosis`, `inspection`), while `qa` currently mounts only arg normalizer.

This causes a user-visible mismatch: the system appears to be "waiting approval" logically, but has no unified interrupt entity and may terminate instead of pausing.

## 2. Goals

1. Make approval middleware the single approval gateway for all high-risk tool calls.
2. For `host_exec_readonly`, require approval only when whitelist/policy validation fails (or cannot prove safety).
3. Enforce unified approval semantics across all agents and high-risk tools.
4. Eliminate approval-wait paths that end up as terminal `EXECUTION_FAILED`.

## 3. Non-Goals

1. No redesign of approval UI interaction model in this change.
2. No outbox workflow redesign beyond keeping compatibility with current `approval_requested` / `approval_decided` processing.
3. No unrelated refactors to planner/replanner logic.

## 4. Desired Runtime Semantics

For any tool call requiring approval:

1. Emit `tool_approval` and enter `waiting_approval`.
2. Do not emit `tool_result` for that call before approval.
3. On approval, resume same call and execute normally.
4. On rejection, return rejection feedback (non-fatal), not terminal runtime failure.

For `host_exec_readonly` specifically:

1. If host policy decision is `allow_readonly_execute`: run directly.
2. Otherwise: go through middleware `StatefulInterrupt` and approval task flow.

### 4.1 Approval TTL

Pending approvals must not hang indefinitely:

1. Each approval task has explicit TTL (default 24h, configurable).
2. If TTL expires before decision, task transitions to `expired` (or `timeout_rejected` compatible terminal state).
3. Expiration must converge run state and release retry/outbox pressure.
4. Expiration path must be visible to user as approval timeout, not runtime crash.

## 5. Architecture

### 5.1 Single Approval Gateway

Approval decisions must be owned by `ApprovalMiddleware` only:

- Tool layer must not represent "pending approval" via pseudo-results (`status=suspended`).
- Approval wait state must always correspond to an actual interrupt (`StatefulInterrupt`) with approval task/outbox entities.

### 5.2 Two-Layer Risk Model (Code Default + DB Override)

Decision source stack:

1. DB policy (`ai_tool_risk_policies`) when matched by `tool_name + scene + command_class`.
2. Code default risk registry as fallback.
3. Fail-closed for uncertain safety classification.

This allows dynamic policy updates while keeping safe defaults even when DB config is missing/incomplete.

### 5.4 DB Unavailability Fallback

When policy DB is unavailable, decision behavior must remain fail-closed:

1. If DB lookup times out/unavailable, immediately fallback to code default registry.
2. If code default cannot prove safe execution, require approval interrupt.
3. DB errors must be recorded in decision metadata/audit logs for postmortem.

### 5.3 `host_exec_readonly` Policy Reuse

Middleware reuses host command policy engine/whitelist classification before execution:

- `allow_readonly_execute` => no approval.
- any other decision (`require_approval_interrupt`, parse/allowlist/operator violations, unknown) => approval interrupt.

This preserves the whitelist safety model and avoids blanket approval for all readonly host commands.

## 6. Full Tool Scope (High-Risk Coverage)

All high-risk tool families must be governed by unified middleware semantics:

1. Host:
   - `host_exec_readonly` (conditional approval)
   - `host_exec_change`
   - `host_batch`, `host_batch_exec_apply`, `host_batch_status_update`
   - legacy wrappers: `host_ssh_exec_readonly`, `host_exec`, `host_exec_by_target`
2. Kubernetes writes:
   - scale/restart/delete/rollback/write-class operations
3. CICD writes:
   - trigger/cancel pipeline operations
4. Service/Deployment writes:
   - tools that mutate runtime/service/deployment state

Readonly tools remain no-approval by default unless overridden by DB policy.

## 7. Agent Tool-Mount Governance

All tool-calling agents must mount the same safety middleware chain:

1. arg normalizer middleware
2. approval middleware

Current target alignment:

1. `change`: already mounts both, keep.
2. `diagnosis`: already mounts both, keep.
3. `inspection`: already mounts both, keep.
4. `qa`: currently missing approval middleware; add approval middleware for consistency and future high-risk tool safety.

Router agent does not execute tools directly; no approval logic should be placed there.

## 8. Data Flow

### 8.1 Pre-Call

1. Tool call enters middleware.
2. Middleware evaluates risk decision (host policy for host exec + global policy stack).
3. If approval required: raise `StatefulInterrupt` and persist approval task/outbox.
4. If not required: invoke tool endpoint directly.

### 8.2 Waiting Approval

1. Stream emits `tool_approval`.
2. Run state becomes `waiting_approval`.
3. No pre-approval `tool_result` for that blocked call.

### 8.3 Resume

1. Approval submitted via existing approval submit API.
2. Resume with params targets same call id.
3. Approved => execute and emit `tool_result`.
4. Rejected => emit readable rejection result and converge run without terminal runtime failure.
5. Rejection feedback must be injected into agent-visible context as explicit system/tool outcome signal (for example: `Tool execution rejected by user/admin policy`) to prevent blind retries.

### 8.4 Concurrent Approval Requests

For steps producing multiple high-risk calls:

1. Runtime keeps one interrupt boundary at a time per run checkpoint.
2. Multiple approval-required calls from a logical batch should be represented as a grouped approval unit when possible (single user-facing approval entity with per-call details).
3. If grouping is not possible in a given execution path, interrupts must be sequenced deterministically with stable call ordering.
4. Grouping/sequencing choice must be explicit in event payload (`group_id` or per-call metadata) for UI consistency.

### 8.5 Audit and Security Logging

Approval middleware is the security choke point and must emit standardized audit records:

1. Required fields: `run_id`, `session_id`, `call_id`, `tool_name`, `arguments_digest`, decision source, matched DB policy/rule id, command class, requester identity.
2. On resume decision: approver identity, decision timestamp, decision outcome, reject reason/comment.
3. Logs must be machine-parseable and trace-correlated with outbox/task IDs.

## 9. Migration Plan

1. Remove pseudo-approval semantics from host tool execution path:
   - stop using `status=suspended` as approval signal.
2. Add host readonly policy-aware precheck to approval middleware.
3. Introduce/centralize code default risk registry and DB override merge logic.
4. Align all agent tool middleware chains (including `qa`).
5. Keep backward-compatible API contracts (`tool_approval`, `waiting_approval`, submit approval).
6. Deployment transition compatibility:
   - during rollout, keep legacy suspended-state parser for in-flight runs created before migration.
   - once in-flight legacy sessions drain, remove legacy parser path in a cleanup change.

## 10. Test Plan

### 10.1 Core Semantics

1. Approval-required call emits `tool_approval` + `waiting_approval`, no immediate `tool_result`.
2. Approved resume executes same call and emits `tool_result`.
3. Rejected approval yields non-fatal rejection output.
4. Approval wait path never degrades into terminal `EXECUTION_FAILED`.
5. Approval TTL expiry transitions to timeout/expired terminal approval outcome without runtime fatal error.

### 10.2 `host_exec_readonly`

1. Whitelist pass command => executes without approval.
2. Whitelist fail/parse fail/disallowed operator => approval interrupt.
3. Approved resume executes successfully.
4. Rejected resume does not emit terminal runtime failure.

### 10.3 High-Risk Tool Families

1. k8s/cicd/service/deployment write tools are all approval-gated.
2. DB override can promote/demote approval requirement where allowed by policy model.

### 10.4 Agent Wiring

1. `change`, `diagnosis`, `inspection`, `qa` all mount `normalizer + approval`.
2. No agent-specific bypass for high-risk tools.

## 11. Acceptance Criteria

1. No reproduction of "risk command should approve but stream ends with EXECUTION_FAILED".
2. Approval waiting always maps to actionable approval entity.
3. All high-risk tools share one approval behavior model across agents.
4. `host_exec_readonly` approval behavior is whitelist-driven, not blanket interruption.

## 12. Risks and Mitigations

1. Risk: Over-blocking due to fail-closed defaults.
   - Mitigation: DB override + explicit command class tuning + staged rollout.
2. Risk: Behavior drift between tool package and middleware.
   - Mitigation: single decision source in middleware; tool layer no longer models approval wait.
3. Risk: Partial agent adoption.
   - Mitigation: add agent wiring conformance tests.
