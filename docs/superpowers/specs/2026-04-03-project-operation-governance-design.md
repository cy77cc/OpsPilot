# 2026-04-03 Project-Wide Operation Governance Module Design

## 1. Goal

Establish a single governance module for the entire OpsPilot project to standardize:

1. Risk assessment
2. Permission/auth guard behavior
3. Approval lifecycle
4. Write-operation response envelope
5. Audit and redaction

This module is project-level and domain-agnostic. Cluster is only the first migration target.

## 2. Current-State Analysis (As-Is)

## 2.1 Active Runtime Path

Current active cluster routes are registered in `internal/service/cluster/routes.go` and bound to handlers in package `cluster`.

High-risk endpoints currently include:

1. `POST /clusters/:id/nodes/:name/cordon`
2. `POST /clusters/:id/nodes/:name/uncordon`
3. `POST /clusters/:id/nodes/:name/drain`
4. `POST /clusters/:id/nodes/:name/taints`
5. `DELETE /clusters/:id/nodes/:name/taints`
6. `POST /clusters/:id/nodes/:name/labels`
7. `DELETE /clusters/:id/nodes/:name/labels`
8. `DELETE /clusters/:id/nodes/:name`
9. `POST /clusters/:id/upgrade`
10. `POST /clusters/:id/certificates/renew`
11. `GET /clusters/:id/operations/history`
12. `GET /clusters/:id/operations/:audit_id`

## 2.2 Current Governance Components

In the active `cluster` package, governance logic is partially centralized but still endpoint-coupled:

1. Approval lifecycle primitives:
   - `internal/service/cluster/approval_policy.go`
   - scope binding (`cluster_id + namespace + action + resource + resource_id`)
   - single-use consume + replay detection
2. Write envelope helpers:
   - `internal/service/cluster/operation_response.go`
3. Audit and gate helper:
   - `internal/service/cluster/handler_operations.go`
4. Redaction utility:
   - `internal/service/cluster/redaction.go`

## 2.3 Legacy/Parallel Governance Path

A separate legacy path still exists under `internal/service/cluster/handler/*` with overlapping but different logic:

1. `requireProdApproval` and `createAudit` in `internal/service/cluster/handler/policy.go`
2. approval create/confirm in `internal/service/cluster/handler/handler_approval.go`

These are not the main routed runtime path in current cluster routes.

## 2.4 Key Problems

1. Governance is still coupled with business handlers:
   - endpoints decide when and how to call gate logic
2. Duplicate/parallel stacks:
   - active `cluster/*` and legacy `cluster/handler/*`
3. Response-contract drift risk:
   - multiple operation response structs and mapping logic
4. Domain-specific approval persistence:
   - `cluster_deploy_approvals` limits cross-domain reuse
5. Audit schema is thin for project-wide governance:
   - mostly message-centric, limited structured payload

## 3. Design Principles (Project-Level)

1. Single policy source of truth
2. Domain-agnostic governance core
3. Deterministic and testable lifecycle/state transitions
4. Backward-compatible rollout by adapter migration
5. Security-first defaults (single-use token, strict scope, redaction-by-default)

## 4. Target Architecture

## 4.1 Module Location

Introduce a project-level module:

1. `internal/service/governance` (preferred)

Sub-packages:

1. `policy` - risk + permission decision
2. `approval` - issue/confirm/consume/replay lifecycle
3. `audit` - persistence + query + redaction integration
4. `envelope` - normalized response mapping
5. `adapter` - per-domain thin adapters

## 4.2 Runtime Flow

All governed write operations follow:

1. Build `OperationIntent`
2. `Preflight(intent)`
3. If approval required: return envelope `state=approval_required`
4. If allowed: execute business function
5. `Finalize(intent, executionResult)`
6. Return normalized envelope with `audit_id`

No domain handler should implement custom approval/replay/redaction logic directly.

## 5. Core Contracts

## 5.1 OperationIntent

Required fields:

1. `domain` (cluster/deployment/service/host/config/...)
2. `resource`
3. `resource_id`
4. `action`
5. `namespace` (optional)
6. `target_scope` (cluster/project/team/global)
7. `operator_id`
8. `approval_token` (optional)
9. `request_summary` (redactable)

## 5.2 Preflight Decision

Possible outcomes:

1. `allowed`
2. `approval_required`
3. `rejected`
4. `failed` (policy/system errors)

Each decision includes stable `code` and optional approval metadata.

## 5.3 Unified Write Envelope

Canonical payload:

```json
{
  "state": "completed",
  "approval": {
    "ticket": "",
    "expires_at": ""
  },
  "audit_id": 123,
  "code": "success",
  "message": "",
  "data": {}
}
```

Allowed states:

1. `completed`
2. `approval_required`
3. `rejected`
4. `failed`

## 6. Data Model (Project-Wide)

## 6.1 Approvals

Introduce generic `operation_approvals`:

1. `id`
2. `ticket` (unique)
3. `domain`
4. `scope_cluster_id` (nullable)
5. `scope_project_id` (nullable)
6. `namespace`
7. `resource`
8. `resource_id`
9. `action`
10. `status` (`pending|approved|rejected`)
11. `request_by`
12. `review_by`
13. `expires_at`
14. `consumed_at`
15. `consumed_by`
16. `replay_count`
17. `replay_at`
18. `replay_by`
19. `replay_code`
20. timestamps

## 6.2 Audits

Introduce/upgrade generic `operation_audits`:

1. `id`
2. `domain`
3. `resource`
4. `resource_id`
5. `action`
6. `scope_cluster_id`/`scope_project_id`
7. `namespace`
8. `operator_id`
9. `status`
10. `code`
11. `message`
12. `request_summary_json` (redacted)
13. `result_summary_json` (redacted)
14. `diagnostics_json` (redacted)
15. `approval_ticket` (optional)
16. `latency_ms`
17. timestamps

## 7. Policy Model

## 7.1 Risk Classification

Policy key:

1. `domain + resource + action + context`

Policy output:

1. `risk_level` (`low|medium|high|critical`)
2. `approval_required` boolean
3. `required_permissions` list
4. optional constraints (time-window/team/env)

## 7.2 Permission Compatibility

Keep existing permission naming and aliases:

1. `resource:action` canonical
2. alias compatibility in policy resolver (for current RBAC continuity)

## 8. Security Requirements

1. Approval token single-use by default
2. Strict scope binding check
3. Deterministic replay code: `approval_token_replayed`
4. Expired token deterministic code: `approval_token_expired`
5. Redaction before persistence for request/result/diagnostics
6. No secret/plain token persisted

## 9. Migration Strategy

## Phase 1: Foundation

1. Build governance module API and internal adapters
2. Add generic schema (no switch-over yet)
3. Add compatibility bridge from old tables where needed

## Phase 2: Cluster Adoption

1. Move cluster high-risk handlers to governance module calls
2. Route approval confirm APIs through governance approval service
3. Keep endpoint contracts stable

## Phase 3: Other Domains

1. Deployment and service write ops
2. Host and config high-risk ops
3. Automation operations

## Phase 4: Cleanup

1. Remove duplicate per-module approval/risk code
2. Deprecate legacy domain-specific approval tables after full cutover

## 10. Testing Strategy (Project-Level)

## 10.1 Conformance Suite

Define reusable governance contract tests for every governed endpoint:

1. requires approval when policy says so
2. accepts approved token once
3. rejects replay with deterministic code
4. rejects scope mismatch
5. rejects expired token
6. persists redacted audit payload
7. returns envelope state consistently

## 10.2 Domain Adapter Tests

Each domain validates:

1. correct intent mapping
2. correct resource/action scope mapping
3. no handler-local approval bypass paths

## 11. Rollout Controls

1. Feature flag per domain for governance routing
2. staged rollout: non-prod -> canary -> full
3. observe metrics:
   1. approval_required rate
   2. approval latency
   3. replay rejection count
   4. audit write/query latency
   5. governed endpoint error rate

## 12. Definition Of Done

1. Governance core is shared at project level
2. Cluster no longer owns bespoke approval lifecycle logic
3. At least one additional non-cluster domain is migrated
4. Conformance tests are green for all migrated endpoints
5. Legacy duplicate governance code paths are retired or behind explicit compatibility adapters
