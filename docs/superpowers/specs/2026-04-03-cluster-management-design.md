# 2026-04-03 Cluster Management Design (Revised)

## 1. Background And Goal

Current cluster management supports cluster onboarding and resource visibility, but lacks complete operation capability after entering a cluster. Users can mostly "see what exists" and cannot finish core day-2 operations in one place.

This change upgrades cluster management into a complete operations console covering:

1. Workload operations
2. Cluster and node operations
3. Security and configuration operations

And enforces a unified approval gate for high-risk actions.

## 2. Scope

### In Scope

1. Upgrade cluster detail page from resource-centric view to operations console
2. Add/complete backend APIs for workload, node, and config/security operations
3. Enforce approval for high-risk actions with unified token flow
4. Provide unified operation result envelope and auditable operation timeline
5. Add tests for core approval-execution and audit redaction flows

### Out Of Scope

1. Cross-cluster orchestration
2. Multi-step workflow engine beyond current approval model
3. Non-Kubernetes runtime operations

## 3. Compatibility And Migration

### 3.1 Permission Compatibility

To avoid RBAC breakage, this design follows existing permission naming style (`resource:action`) and keeps backward-compatible checks:

1. Read operations: `cluster:read` or compatible legacy read aliases
2. Write operations: `cluster:write` and/or operation-specific `k8s:*` permissions
3. Approval confirmation: `k8s:approve` or `kubernetes:approve`
4. No new `cluster.read/cluster.operate/cluster.admin/cluster.approve` namespace is introduced in this phase

### 3.2 Endpoint Compatibility

1. Existing endpoints remain stable where possible
2. New operation endpoints are additive
3. Existing `/clusters/:id/approvals` and `/clusters/:id/approvals/:ticket/confirm` are reused and extended

## 4. Architecture

The implementation follows a domain-oriented structure with four layers:

1. `workload` domain: deployment/statefulset/daemonset/pod operations
2. `node` domain: cordon/drain/taint/label/remove lifecycle
3. `config-security` domain: namespace/config/secret/quota/limitrange/upgrade/certificates
4. `approval-audit` domain: approval ticketing, confirmation, operation audit timeline

All write operations go through a unified flow:

1. UI submits operation intent with request idempotency key
2. Backend validates permission and risk level
3. Backend creates approval ticket when required
4. Action executes only with valid approval token
5. Execution result and diagnostics are persisted to audit log
6. UI shows status and links to audit detail

## 5. API Design

### 5.1 Unified Response Envelope (Write APIs)

All write APIs return a normalized envelope with explicit approval-state handling:

```json
{
  "success": true,
  "state": "completed",
  "message": "operation completed",
  "error_code": "",
  "diagnostics": [],
  "audit_id": "op_20260403_xxx",
  "approval": {
    "required": false,
    "ticket": "",
    "expires_at": "",
    "reason": ""
  }
}
```

`state` values:

1. `completed`
2. `approval_required`
3. `rejected`
4. `failed`

When `state=approval_required`, backend MUST include `approval.ticket`, `approval.expires_at`, and `approval.reason`.

### 5.2 Workload APIs

1. `POST /clusters/:id/namespaces/:ns/deployments/:name/scale`
2. `POST /clusters/:id/namespaces/:ns/deployments/:name/restart`
3. `POST /clusters/:id/namespaces/:ns/deployments/:name/rollback`
4. `DELETE /clusters/:id/namespaces/:ns/deployments/:name`
5. `DELETE /clusters/:id/namespaces/:ns/pods/:name`
6. `GET /clusters/:id/namespaces/:ns/pods/:name/logs`
7. `GET /clusters/:id/namespaces/:ns/workloads/:kind/:name/events`

### 5.3 Node APIs

1. `POST /clusters/:id/nodes/:name/cordon`
2. `POST /clusters/:id/nodes/:name/uncordon`
3. `POST /clusters/:id/nodes/:name/drain`
4. `POST /clusters/:id/nodes/:name/taints` (upsert)
5. `DELETE /clusters/:id/nodes/:name/taints`
6. `POST /clusters/:id/nodes/:name/labels` (upsert)
7. `DELETE /clusters/:id/nodes/:name/labels`
8. `DELETE /clusters/:id/nodes/:name` (node removal; high-risk, approval-gated)

### 5.4 Config/Security APIs

1. Namespace lifecycle APIs (existing endpoints with approval enforcement for high-risk contexts)
2. ConfigMap CRUD APIs under namespace path
3. Secret CRUD APIs under namespace path
4. ResourceQuota CRUD APIs under namespace path
5. LimitRange CRUD APIs under namespace path
6. `POST /clusters/:id/upgrade` with approval enforcement
7. `POST /clusters/:id/certificates/renew` with approval enforcement

### 5.5 Approval/Audit APIs

1. Reuse `POST /clusters/:id/approvals` and extend action/object metadata
2. Reuse `POST /clusters/:id/approvals/:ticket/confirm`
3. Add `GET /clusters/:id/operations/history?page=:page&page_size=:size&resource=:resource&status=:status&operator=:uid&from=:ts&to=:ts`
4. Add `GET /clusters/:id/operations/:audit_id`

History endpoint requirements:

1. Default pagination (`page=1`, `page_size=20`, max `page_size=100`)
2. Sort by `created_at desc` by default
3. Filter fields are optional and composable

## 6. Frontend Design

`ClusterDetailPage` becomes an operations console with three core areas:

1. Resource list and filters
2. Resource detail drawer/editor
3. Operation panel and operation center entry

### 6.1 Interaction Model

1. Every resource row exposes operation dropdown
2. High-risk operations open approval preview modal first
3. Operation feedback is visible at three levels:
   1. Toast status
   2. Row/detail status refresh
   3. Audit-linked operation detail
4. If API returns `state=approval_required`, UI must render approval ticket state and provide confirm/retry entry

### 6.2 Node UX

1. Node quick actions: cordon/uncordon/drain/remove
2. Label and taint editor in drawer
3. Drain form supports runtime flags (daemonset handling, force policy, grace period)

### 6.3 Config/Security UX

1. Editable namespace policy, quota, and limit range panels
2. ConfigMap and Secret key-value editors
3. Secret value masking by default with controlled reveal and explicit audit-safe mode

## 7. Approval, Permission, And Error Model

### 7.1 Mandatory Approval

High-risk actions always require approval token:

1. Delete
2. Rollback
3. Upgrade
4. Certificate renewal
5. Node remove/drain/taint mutation

### 7.2 Approval Token Security

1. Token is single-use by default
2. Token is scope-bound to `cluster_id + namespace + action + resource + resource_id`
3. Token includes expiration (`expires_at`) and cannot be extended in-place
4. Token is invalidated immediately after successful execution or explicit rejection
5. Replay attempts return deterministic error code

### 7.3 Permission Model

1. Route/view access follows existing permissions (`cluster:read`, `cluster:write`, related `k8s:*` permissions)
2. Approval confirmation requires `k8s:approve` or compatible alias
3. Backward-compatible aliases may be accepted at backend guard layer, but new policy entries follow `resource:action`

### 7.4 Error Codes

1. `approval_required`
2. `approval_rejected`
3. `approval_token_invalid`
4. `approval_token_expired`
5. `approval_token_replayed`
6. `permission_denied`
7. `k8s_conflict`
8. `k8s_not_found`
9. `k8s_timeout`

## 8. Audit, Diagnostics, And Data Safety

### 8.1 Audit Requirements

1. Every write operation emits one audit record with stable `audit_id`
2. Audit record stores: actor, action, target, request summary, result state, timing, diagnostics summary
3. Audit detail endpoint includes approval linkage (`ticket`, approver, decision timestamp)

### 8.2 Secret/PII Redaction Rules

1. Secret values MUST NEVER be persisted in audit or diagnostics payloads
2. Sensitive fields are stored as masked placeholders (for example `"***"`)
3. Raw backend/Kubernetes errors must pass redaction before persistence or response
4. UI detail views use redacted data only

### 8.3 Retention And Query

1. Operation history supports retention policy configuration
2. Purge/archival jobs run out-of-band and preserve index integrity
3. Query path remains paginated and filterable for large clusters

## 9. Testing Strategy

### 9.1 Backend

1. Handler unit tests for validation, approval gate, token security (single-use/scope/expiry), execution path, and failure branches
2. Route-level integration tests for approval-execution chain and audit record completeness
3. Redaction tests for secret/config diagnostics paths

### 9.2 Frontend

1. Cluster detail interaction tests for approval modal and execution state updates
2. API module tests for unified envelope handling (`completed/approval_required/rejected/failed`)
3. Secret reveal/masking behavior tests

### 9.3 E2E

1. Approval approved -> execution succeeds
2. Approval rejected -> execution blocked
3. Execution failed -> diagnostics and audit trace are visible
4. Approval token replay -> blocked with `approval_token_replayed`
5. Secret operation -> audit and API payloads remain redacted

## 10. Delivery Strategy

The first implementation target is `Flagship` level:

1. Full domain coverage (workload + node + security/config)
2. Backend and frontend both extended
3. Approval and audit flow required for high-risk operations
4. Compatibility-first rollout (no permission namespace break)

This design intentionally prioritizes operational safety and backward compatibility before UI surface expansion.
