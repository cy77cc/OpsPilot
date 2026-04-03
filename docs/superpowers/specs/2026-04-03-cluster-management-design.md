# 2026-04-03 Cluster Management Design

## 1. Background And Goal

Current cluster management supports cluster onboarding and resource visibility, but lacks complete operation capability after entering a cluster. Users can mostly "see what exists" and cannot finish core day-2 operations in one place.

This change upgrades cluster management into a complete operations console covering:

1. Workload operations
2. Cluster and node operations
3. Security and configuration operations

And enforces a unified approval gate for high-risk actions.

## 2. Scope

### In Scope

1. Upgrade cluster detail page from read-only resource list to operations console
2. Add/complete backend APIs for workload, node, config/security operations
3. Enforce approval for high-risk actions
4. Provide unified operation result and audit traceability
5. Add tests for core approval-execution flows

### Out Of Scope

1. Cross-cluster orchestration
2. Multi-step workflow engine beyond current approval model
3. Non-Kubernetes runtime operations

## 3. Architecture

The implementation follows a domain-oriented structure with four layers:

1. `workload` domain: deployment/statefulset/daemonset/pod operations
2. `node` domain: cordon/drain/taint/label lifecycle
3. `config-security` domain: namespace/config/secret/quota/limitrange/upgrade/certificates
4. `approval-audit` domain: approval ticketing, confirmation, operation audit timeline

All write operations go through a unified flow:

1. UI submits operation intent
2. Backend validates permission and risk level
3. Backend creates approval ticket when required
4. Action executes only with valid approval token
5. Execution result and diagnostics are persisted to audit log
6. UI shows status and links to audit detail

## 4. API Design

All write APIs return a normalized envelope:

```json
{
  "success": true,
  "message": "operation completed",
  "diagnostics": [],
  "audit_id": "op_20260403_xxx"
}
```

### 4.1 Workload APIs

1. `POST /clusters/:id/namespaces/:ns/deployments/:name/scale`
2. `POST /clusters/:id/namespaces/:ns/deployments/:name/restart`
3. `POST /clusters/:id/namespaces/:ns/deployments/:name/rollback`
4. `DELETE /clusters/:id/namespaces/:ns/deployments/:name`
5. `DELETE /clusters/:id/namespaces/:ns/pods/:name`
6. `GET /clusters/:id/namespaces/:ns/pods/:name/logs`
7. `GET /clusters/:id/namespaces/:ns/workloads/:kind/:name/events`

### 4.2 Node APIs

1. `POST /clusters/:id/nodes/:name/cordon`
2. `POST /clusters/:id/nodes/:name/uncordon`
3. `POST /clusters/:id/nodes/:name/drain`
4. `POST /clusters/:id/nodes/:name/taints` (upsert)
5. `DELETE /clusters/:id/nodes/:name/taints`
6. `POST /clusters/:id/nodes/:name/labels` (upsert)
7. `DELETE /clusters/:id/nodes/:name/labels`

### 4.3 Config/Security APIs

1. Namespace lifecycle APIs (existing endpoints with approval enforcement for high-risk contexts)
2. ConfigMap CRUD APIs under namespace path
3. Secret CRUD APIs under namespace path
4. ResourceQuota CRUD APIs under namespace path
5. LimitRange CRUD APIs under namespace path
6. `POST /clusters/:id/upgrade` with approval enforcement
7. `POST /clusters/:id/certificates/renew` with approval enforcement

### 4.4 Approval/Audit APIs

1. Reuse `POST /clusters/:id/approvals` and extend action/object metadata
2. Reuse `POST /clusters/:id/approvals/:ticket/confirm`
3. Add `GET /clusters/:id/operations/history`
4. Add `GET /clusters/:id/operations/:audit_id`

## 5. Frontend Design

`ClusterDetailPage` becomes an operations console with three core areas:

1. Resource list and filters
2. Resource detail drawer
3. Operation panel and operation center entry

### 5.1 Interaction Model

1. Every resource row exposes operation dropdown
2. High-risk operations open approval preview modal first
3. Operation feedback is visible at three levels:
   1. Toast status
   2. Row/detail status refresh
   3. Audit-linked operation detail

### 5.2 Node UX

1. Node quick actions: cordon/uncordon/drain
2. Label and taint editor in drawer
3. Drain form supports runtime flags (daemonset handling, force policy)

### 5.3 Config/Security UX

1. Editable namespace policy, quota, and limit range panels
2. ConfigMap and Secret key-value editors
3. Secret value masking by default with controlled reveal

## 6. Approval, Permission, And Error Model

### 6.1 Mandatory Approval

High-risk actions always require approval token:

1. Delete
2. Rollback
3. Upgrade
4. Certificate renewal
5. Node removal/drain/taint mutation

### 6.2 Permission Model

1. `cluster.read`
2. `cluster.operate`
3. `cluster.admin`
4. `cluster.approve` for approval confirmation capability

### 6.3 Error Codes

1. `approval_required`
2. `approval_rejected`
3. `permission_denied`
4. `k8s_conflict`
5. `k8s_not_found`
6. `k8s_timeout`

## 7. Testing Strategy

### 7.1 Backend

1. Handler unit tests for validation, approval gate, execution path, and failure branches
2. Route-level integration tests for approval-execution chain

### 7.2 Frontend

1. Cluster detail interaction tests for approval modal and execution state updates
2. API module tests for new operations and normalized error handling

### 7.3 E2E

1. Approval approved -> execution succeeds
2. Approval rejected -> execution blocked
3. Execution failed -> diagnostics and audit trace are visible

## 8. Delivery Strategy

The first implementation target is `Flagship` level:

1. Full domain coverage (workload + node + security/config)
2. Backend and frontend both extended
3. Approval and audit flow required for high-risk operations

This design intentionally prioritizes long-term maintainability over short-term UI-only wiring.
