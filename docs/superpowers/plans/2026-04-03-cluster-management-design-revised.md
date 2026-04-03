# Cluster Management Design (Revised) Executing Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec Source:** `docs/superpowers/specs/2026-04-03-cluster-management-design.md`

**Goal:** Deliver a production-safe cluster operations console with unified approval gating, consistent write response envelope, secure approval tokens, and auditable/redacted operation history.

**Architecture:** Keep existing cluster module boundaries, introduce shared operation/approval/audit primitives in backend handler layer, and migrate `ClusterDetailPage` from read-first tabs to action-first interactions using a normalized API envelope (`state`, `approval`, `audit_id`).

**Tech Stack:** Go (Gin, GORM, Kubernetes client-go), TypeScript/React + Ant Design, Vitest, existing OpsPilot RBAC and cluster APIs.

---

## File Structure

- Modify: `internal/service/cluster/handler/policy.go`
- Modify: `internal/service/cluster/handler/handler_approval.go`
- Modify: `internal/service/cluster/logic_advanced.go`
- Modify: `internal/service/cluster/logic_nodes.go`
- Modify: `internal/service/cluster/routes.go` (or equivalent route registration file)
- Modify: `internal/model/cluster_phase1.go`
- Create: `internal/service/cluster/handler/handler_operations.go` (history/detail APIs)
- Create: `internal/service/cluster/handler/operation_response.go` (normalized write envelope helpers)
- Create: `internal/service/cluster/handler/redaction.go` (audit/diagnostic redaction)
- Modify: `web/src/api/modules/cluster.ts`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx` (if operation center is split page)
- Create: `web/src/api/modules/cluster.operations.test.ts`
- Modify: `web/src/ProtectedApp.tsx` (route wiring if new page added)
- Create/Modify tests under:
  - `internal/service/cluster/**/*_test.go`
  - `web/src/pages/Deployment/Infrastructure/*.test.tsx`

## Chunk 1: Contract And Safety Foundation

### Task 1: Add normalized write response envelope and helpers

**Files:**
- Create: `internal/service/cluster/handler/operation_response.go`
- Modify: cluster write handlers progressively

- [ ] **Step 1: Add failing backend tests asserting envelope shape**
- [ ] **Step 2: Implement helper constructors for `completed|approval_required|rejected|failed` states**
- [ ] **Step 3: Switch one representative write endpoint to new helper (golden path)**
- [ ] **Step 4: Run focused tests and fix regressions**

Acceptance:
- Every migrated write endpoint returns `state`, `approval`, and `audit_id` consistently.

### Task 2: Harden approval token lifecycle

**Files:**
- Modify: `internal/service/cluster/handler/policy.go`
- Modify: `internal/service/cluster/handler/handler_approval.go`
- Modify: `internal/model/cluster_phase1.go`

- [ ] **Step 1: Add failing tests for single-use, scope mismatch, expiry, and replay**
- [ ] **Step 2: Extend approval model fields needed for lifecycle control (consumed timestamps/status metadata)**
- [ ] **Step 3: Enforce scope binding (`cluster_id/namespace/action/resource/resource_id`)**
- [ ] **Step 4: Enforce one-time consumption and deterministic replay error**
- [ ] **Step 5: Run tests and verify all approval scenarios pass**

Acceptance:
- Reused token is rejected with stable `approval_token_replayed` semantics.

### Task 3: Implement redaction guardrail for audit and diagnostics

**Files:**
- Create: `internal/service/cluster/handler/redaction.go`
- Modify: `internal/service/cluster/handler/policy.go`

- [ ] **Step 1: Add failing tests with secret-like payloads and raw Kubernetes errors**
- [ ] **Step 2: Implement recursive redaction for known sensitive keys and secret values**
- [ ] **Step 3: Route audit/diagnostics persistence through redaction layer**
- [ ] **Step 4: Re-run tests and verify no sensitive plain text persists**

Acceptance:
- Secret values never appear in persisted audit records or API diagnostics.

## Chunk 2: Backend Operations Surface

### Task 4: Complete node high-risk operation coverage

**Files:**
- Modify: `internal/service/cluster/logic_nodes.go`
- Modify: route registration file

- [ ] **Step 1: Add failing tests for node `cordon/uncordon/drain/taint/label/remove` behavior**
- [ ] **Step 2: Ensure high-risk operations require approval gate**
- [ ] **Step 3: Add `DELETE /clusters/:id/nodes/:name` approval-gated semantics**
- [ ] **Step 4: Return normalized envelope and audit linkage**

Acceptance:
- Node removal and drain paths are approval-safe and auditable.

### Task 5: Enforce approval on upgrade and certificate renew

**Files:**
- Modify: `internal/service/cluster/logic_advanced.go`

- [ ] **Step 1: Add tests for approval-required behavior on upgrade/renew endpoints**
- [ ] **Step 2: Integrate shared approval gate and envelope response**
- [ ] **Step 3: Preserve existing permission style (`cluster:write` + `k8s:approve` compatible checks)**
- [ ] **Step 4: Verify error code mapping (`approval_required`, `approval_rejected`, token errors)**

Acceptance:
- Upgrade and cert-renew follow identical approval and response contracts.

### Task 6: Add operation history and detail APIs

**Files:**
- Create: `internal/service/cluster/handler/handler_operations.go`
- Modify: route registration file

- [ ] **Step 1: Add failing tests for history pagination/filter/sort defaults**
- [ ] **Step 2: Implement `GET /clusters/:id/operations/history` with query filters**
- [ ] **Step 3: Implement `GET /clusters/:id/operations/:audit_id` detail endpoint**
- [ ] **Step 4: Ensure redacted diagnostics in list/detail responses**

Acceptance:
- History endpoint supports `page/page_size/resource/status/operator/from/to` and returns stable pagination metadata.

## Chunk 3: Frontend Operation Console

### Task 7: Upgrade cluster API module to normalized operation types

**Files:**
- Modify: `web/src/api/modules/cluster.ts`
- Create: `web/src/api/modules/cluster.operations.test.ts`

- [ ] **Step 1: Add failing tests for envelope decoding (`completed|approval_required|rejected|failed`)**
- [ ] **Step 2: Add shared `ClusterOperationResponse` type and approval payload typing**
- [ ] **Step 3: Add API bindings for node operations + operation history/detail**
- [ ] **Step 4: Re-run API module tests**

Acceptance:
- Frontend can handle approval-required states without endpoint-specific branching hacks.

### Task 8: Refactor `ClusterDetailPage` to action-first interaction

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`

- [ ] **Step 1: Add failing UI tests for row-level operations and approval modal trigger**
- [ ] **Step 2: Add operation dropdown/actions on workload and node rows**
- [ ] **Step 3: Wire high-risk actions to approval preview + confirm flow**
- [ ] **Step 4: Show three-level feedback (toast, row refresh, audit link)**
- [ ] **Step 5: Validate mobile/desktop usability for key action flows**

Acceptance:
- Users can execute day-2 actions from resource rows without leaving page context.

### Task 9: Add operation center/history UI

**Files:**
- Create/Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`
- Modify: `web/src/ProtectedApp.tsx` (if route added)

- [ ] **Step 1: Add failing tests for history table pagination and filter form**
- [ ] **Step 2: Implement operation history list with audit state chips**
- [ ] **Step 3: Implement audit detail panel with redacted diagnostics**
- [ ] **Step 4: Link from operation results in `ClusterDetailPage` via `audit_id`**

Acceptance:
- Operation center provides traceable, filterable, redacted audit visibility.

## Chunk 4: Verification, Migration, and Rollout

### Task 10: Permission compatibility and regression validation

**Files:**
- Modify tests across `internal/service/cluster`
- Modify frontend route/permission tests if needed

- [ ] **Step 1: Add regression tests confirming existing `cluster:read/write` route access remains valid**
- [ ] **Step 2: Add regression tests for legacy alias acceptance where required**
- [ ] **Step 3: Verify no `cluster.read/cluster.operate` namespace is introduced**

Acceptance:
- No RBAC migration is required for existing users in this phase.

### Task 11: End-to-end scenario pack

**Files:**
- Modify: `e2e` cluster-related suites

- [ ] **Step 1: Add E2E case: approval approved -> execution succeeds**
- [ ] **Step 2: Add E2E case: approval rejected -> blocked**
- [ ] **Step 3: Add E2E case: replayed token -> blocked with deterministic code**
- [ ] **Step 4: Add E2E case: secret/config operation -> redacted audit payload**

Acceptance:
- All core approval-execution-audit scenarios are covered by CI.

### Task 12: Delivery controls

- [ ] **Step 1: Gate new console capabilities behind feature flag**
- [ ] **Step 2: Perform staged rollout (non-prod -> partial prod -> full prod)**
- [ ] **Step 3: Define and monitor key metrics (error rate, approval latency, audit query latency)**
- [ ] **Step 4: Publish rollback checklist for API/UI feature flag disablement**

Acceptance:
- Rollout has clear observability and rollback path.

---

## Definition Of Done

- [ ] High-risk operations uniformly require approval and use hardened token lifecycle.
- [ ] All migrated write APIs return normalized envelope with explicit `state`.
- [ ] Operation history/detail APIs are pageable, filterable, and redacted.
- [ ] `ClusterDetailPage` supports action-first workflows with approval UX.
- [ ] E2E and regression tests validate compatibility and security requirements.
