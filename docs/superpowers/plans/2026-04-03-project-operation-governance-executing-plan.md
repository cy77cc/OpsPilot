# Project Operation Governance Module Executing Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec Sources:**
- `docs/superpowers/specs/2026-04-03-project-operation-governance-design.md`
- `docs/superpowers/specs/2026-04-03-governance-module-interfaces.md`

**Goal:** Extract a project-level governance module from the already-implemented cluster governance path, then make it the single source of truth for policy, approval lifecycle, canonical write envelopes, and redacted audit persistence across OpsPilot.

**Starting Point:** The cluster management design and cluster-side implementation already exist in active code. The governance module work is therefore an extraction-and-cutover effort, not a greenfield implementation.

**Key Decisions Fixed For This Plan:**
- `operation_audits` immediately replaces cluster historical records.
- Governance policies own namespace, team, and environment constraints.
- The target write envelope is the blueprint contract: `state`, `approval`, `audit_id`, `code`, `message`, `data`.

**Architecture:** Introduce `internal/service/governance` as the domain-agnostic core (`policy`, `approval`, `audit`, `envelope`, `adapter`), migrate cluster runtime to consume it through adapters, and remove cluster-local lifecycle logic after parity is proven.

**Tech Stack:** Go (Gin, GORM), existing RBAC (`httpx.Authorize`/permission aliases), existing OpsPilot cluster APIs/UI, React + TypeScript frontend compatibility layer during rollout.

---

## File Structure

### Create (Governance Core)

- Create: `internal/service/governance/types.go`
- Create: `internal/service/governance/errors.go`
- Create: `internal/service/governance/service.go`
- Create: `internal/service/governance/policy/resolver.go`
- Create: `internal/service/governance/approval/service.go`
- Create: `internal/service/governance/audit/service.go`
- Create: `internal/service/governance/envelope/mapper.go`
- Create: `internal/service/governance/adapter/cluster.go`

### Modify (Existing Cluster Extraction / Cutover)

- Modify: `internal/service/cluster/approval_policy.go`
- Modify: `internal/service/cluster/handler_operations.go`
- Modify: `internal/service/cluster/operation_response.go`
- Modify: `internal/service/cluster/logic_nodes.go`
- Modify: `internal/service/cluster/logic_advanced.go`
- Modify: `internal/service/cluster/routes.go`

### Data Model / Persistence

- Modify: `internal/model/cluster_phase1.go`
- Create: `internal/model/operation_governance.go`
- Create migration artifact(s) for:
  - `operation_approvals`
  - `operation_audits`

### Frontend (Transition Then Cleanup)

- Modify: `web/src/api/modules/cluster.ts`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`
- Modify/Create: `web/src/api/modules/cluster.operations.test.ts`

### Tests

- Create: `internal/service/governance/**/*_test.go`
- Modify/Create: `internal/service/cluster/**/*_test.go`
- Modify/Create: `web/src/api/modules/cluster.operations.test.ts`
- Modify/Create: operation center/detail frontend tests

---

## Chunk 1: Governance Contract Hardening

### Task 1: Expand governance interfaces for real policy context

**Files:**
- `internal/service/governance/types.go`
- `internal/service/governance/service.go`
- `docs/superpowers/specs/2026-04-03-governance-module-interfaces.md` (if blueprint sync is needed)

- [ ] **Step 1: Add failing tests covering policy decisions with namespace/team/environment inputs**
- [ ] **Step 2: Extend `OperationIntent` / `Scope` / policy input types to carry `team_id`, environment, and extensible context**
- [ ] **Step 3: Define which constraints are enforced in governance core vs thin adapters**
- [ ] **Step 4: Ensure the service contracts can express policy context without leaking business rules back to handlers**
- [ ] **Step 5: Run governance contract tests**

Acceptance:
- Governance interfaces can represent namespace, team, and environment constraints directly.
- Adapters only translate request/domain data; they do not become policy engines.

### Task 2: Freeze the canonical envelope contract

**Files:**
- `internal/service/governance/errors.go`
- `internal/service/governance/envelope/mapper.go`
- `internal/service/cluster/operation_response.go`
- `web/src/api/modules/cluster.ts`

- [ ] **Step 1: Add failing tests for blueprint envelope shape only (`state/approval/audit_id/code/message/data`)**
- [ ] **Step 2: Define deterministic mapping for approval, rejection, token, permission, and internal failure codes**
- [ ] **Step 3: Mark cluster-local envelope helpers as compatibility-only during migration**
- [ ] **Step 4: Document legacy field deprecation path for frontend/backend**
- [ ] **Step 5: Run governance + cluster envelope tests**

Acceptance:
- The blueprint envelope is the only target backend contract.
- Legacy fields are tolerated temporarily but are no longer design-authoritative.

---

## Chunk 2: Audit Cutover First

### Task 3: Introduce generic governance audit persistence

**Files:**
- `internal/service/governance/audit/service.go`
- `internal/model/operation_governance.go`
- migration files for `operation_audits`

- [ ] **Step 1: Add failing tests for structured redacted request/result/diagnostics persistence**
- [ ] **Step 2: Implement `operation_audits` model/schema with generic scope fields**
- [ ] **Step 3: Persist governance-shaped audit records with redaction by default**
- [ ] **Step 4: Add deterministic record status/code mapping**
- [ ] **Step 5: Run audit package tests**

Acceptance:
- `operation_audits` can represent the active cluster history surface and future multi-domain records.

### Task 4: Cut cluster history/detail endpoints over to `operation_audits`

**Files:**
- `internal/service/cluster/handler_operations.go`
- `internal/service/governance/audit/service.go`
- `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`

- [ ] **Step 1: Add failing parity tests for history/detail list semantics against current cluster behavior**
- [ ] **Step 2: Rewrite cluster history/detail queries to read `operation_audits` instead of `cluster_operation_audits`**
- [ ] **Step 3: Preserve pagination/filter defaults and operation center compatibility**
- [ ] **Step 4: Verify redacted detail payloads and approval linkage in generic audit responses**
- [ ] **Step 5: Run cluster history/detail tests and frontend operation center tests**

Acceptance:
- Cluster history/detail reads only from `operation_audits`.
- Operation center behavior remains stable after the storage cutover.

### Task 5: Define hard-cutover migration and rollback behavior for audits

**Files:**
- migration artifacts
- docs/runbook updates

- [ ] **Step 1: Decide whether legacy `cluster_operation_audits` data is backfilled or intentionally abandoned**
- [ ] **Step 2: Implement the chosen cutover procedure explicitly in migration/runbook artifacts**
- [ ] **Step 3: Define rollback behavior for reads after partial cutover**
- [ ] **Step 4: Validate non-prod cutover with representative history/detail queries**

Acceptance:
- Audit replacement is operationally defined, not implied.
- Rollback behavior is explicit for both query path and new writes.

---

## Chunk 3: Approval Service Surface Before Endpoint Migration

### Task 6: Implement generic approval lifecycle service

**Files:**
- `internal/service/governance/approval/service.go`
- `internal/model/operation_governance.go`
- migration files for `operation_approvals`

- [ ] **Step 1: Add failing tests for issue/confirm/consume/replay/expiry/scope mismatch**
- [ ] **Step 2: Implement lifecycle transitions with single-use consumption**
- [ ] **Step 3: Enforce strict scope binding across cluster/project/namespace/resource/action plus governance context fields**
- [ ] **Step 4: Validate deterministic approval error codes**
- [ ] **Step 5: Run approval tests**

Acceptance:
- Approval lifecycle is reusable and deterministic across domains.

### Task 7: Route active cluster approval APIs through governance first

**Files:**
- `internal/service/cluster/routes.go`
- cluster approval handlers in active runtime path
- `internal/service/governance/approval/service.go`

- [ ] **Step 1: Add failing tests for `POST /clusters/:id/approvals` and `POST /clusters/:id/approvals/:ticket/confirm` in the active routed module**
- [ ] **Step 2: Expose active cluster approval endpoints if missing**
- [ ] **Step 3: Back those endpoints with governance approval service semantics**
- [ ] **Step 4: Preserve current external payload compatibility where required**
- [ ] **Step 5: Run approval endpoint tests**

Acceptance:
- Active approval APIs are governed before governed write endpoints depend on them.

### Task 8: Define compatibility period for legacy approval storage

**Files:**
- `internal/service/cluster/approval_policy.go`
- `internal/service/governance/approval/service.go`

- [ ] **Step 1: Decide whether legacy `cluster_deploy_approvals` will be read-through, migrated, or retired immediately**
- [ ] **Step 2: Implement compatibility behavior explicitly**
- [ ] **Step 3: Add regression tests for replay/scope/expiry semantics under the chosen mode**

Acceptance:
- Approval storage cutover behavior is explicit and testable.

---

## Chunk 4: Cluster Extraction And Adoption

### Task 9: Extract current cluster governance behavior into governance core

**Files:**
- `internal/service/cluster/approval_policy.go`
- `internal/service/cluster/handler_operations.go`
- `internal/service/cluster/operation_response.go`
- `internal/service/governance/**/*`

- [ ] **Step 1: Add failing tests that capture current active cluster approval/audit/envelope behavior**
- [ ] **Step 2: Extract reusable logic into governance core instead of duplicating it**
- [ ] **Step 3: Leave cluster files as thin compatibility/adaptation layers**
- [ ] **Step 4: Verify parity for replay protection, scope binding, and audit IDs**
- [ ] **Step 5: Run cluster and governance package tests**

Acceptance:
- Governance module is built by extraction from active behavior, not by speculative reimplementation.

### Task 10: Introduce cluster governed executor/adapter

**Files:**
- `internal/service/governance/adapter/cluster.go`
- `internal/service/cluster/handler_operations.go`

- [ ] **Step 1: Add failing adapter tests for mapping active cluster requests into `OperationIntent`**
- [ ] **Step 2: Implement governed executor flow (`Preflight -> Run -> Finalize -> Envelope`)**
- [ ] **Step 3: Preserve existing handler signatures and endpoint contracts during transition**
- [ ] **Step 4: Prove adapters do not contain policy decisions**
- [ ] **Step 5: Run cluster adapter tests/build**

Acceptance:
- Cluster handlers consume governance through adapters and executor flow only.

### Task 11: Migrate node high-risk operations

**Files:**
- `internal/service/cluster/logic_nodes.go`

- [ ] **Step 1: Add failing conformance tests for node operations (`approval_required/completed/replayed/scope_mismatch/expired`)**
- [ ] **Step 2: Switch `cordon/uncordon/drain/taint/label/remove` to governed executor**
- [ ] **Step 3: Preserve existing permission aliases and route signatures**
- [ ] **Step 4: Validate audit linkage and blueprint envelope output**
- [ ] **Step 5: Run tests/build**

Acceptance:
- Node mutations are fully governed through shared module behavior.

### Task 12: Migrate upgrade and certificate renewal operations

**Files:**
- `internal/service/cluster/logic_advanced.go`

- [ ] **Step 1: Add failing conformance tests for upgrade/cert renew governance behavior**
- [ ] **Step 2: Switch both endpoints to governed executor flow**
- [ ] **Step 3: Preserve existing permission aliases**
- [ ] **Step 4: Verify deterministic token errors, audit IDs, and blueprint envelope output**
- [ ] **Step 5: Run tests/build**

Acceptance:
- Upgrade and certificate renewal no longer manage lifecycle logic locally.

### Task 13: Remove cluster-local lifecycle ownership after parity

**Files:**
- `internal/service/cluster/approval_policy.go`
- `internal/service/cluster/handler_operations.go`
- `internal/service/cluster/operation_response.go`

- [ ] **Step 1: Identify leftover cluster-local lifecycle code that duplicates governance core**
- [ ] **Step 2: Remove or deprecate duplicated logic after parity is proven**
- [ ] **Step 3: Keep only minimal compatibility shims where rollback requires them**
- [ ] **Step 4: Re-run full targeted cluster governance suite**

Acceptance:
- Cluster no longer owns bespoke approval/replay/redaction/envelope logic.

---

## Chunk 5: Frontend Transition And Contract Cleanup

### Task 14: Keep frontend tolerant during backend migration

**Files:**
- `web/src/api/modules/cluster.ts`
- `web/src/api/modules/cluster.operations.test.ts`

- [ ] **Step 1: Add failing tests for blueprint envelope handling and transitional compatibility handling**
- [ ] **Step 2: Ensure frontend can consume the governance blueprint contract from migrated endpoints**
- [ ] **Step 3: Preserve compatibility while backend cutover is incomplete**
- [ ] **Step 4: Run API module tests**

Acceptance:
- Frontend does not block backend cutover and correctly understands the blueprint envelope.

### Task 15: Align cluster detail and operation center with governance outputs

**Files:**
- `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`

- [ ] **Step 1: Add failing UI tests for approval modal, retry flow, and audit detail links**
- [ ] **Step 2: Validate operation center against `operation_audits`-backed responses**
- [ ] **Step 3: Validate approval create/confirm/retry flow against governance lifecycle semantics**
- [ ] **Step 4: Run frontend page tests/build**

Acceptance:
- Cluster UI is stable on top of governance-managed approvals and generic audits.

### Task 16: Remove legacy contract branches after convergence

**Files:**
- `web/src/api/modules/cluster.ts`
- frontend tests and any backend compatibility helpers

- [ ] **Step 1: Identify normalization branches that only serve retired cluster-local contracts**
- [ ] **Step 2: Remove deprecated response-shape compatibility once all migrated endpoints emit blueprint shape**
- [ ] **Step 3: Re-run frontend tests/build**

Acceptance:
- Frontend no longer depends on pre-governance cluster response variants.

---

## Chunk 6: Authorization Regression, Rollout, and Observability

### Task 17: Add governance authorization regression suite

**Files:**
- `internal/service/governance/**/*_test.go`
- `internal/service/cluster/**/*_test.go`

- [ ] **Step 1: Add regression tests for namespace/team/environment constraints**
- [ ] **Step 2: Add regression tests for existing RBAC alias acceptance**
- [ ] **Step 3: Validate no authorization weakening during adapter cutover**
- [ ] **Step 4: Run targeted CI-equivalent test matrix**

Acceptance:
- Governance ownership of policy does not weaken existing authorization behavior.

### Task 18: Rollout controls and rollback procedure

**Files:**
- governance config and integration points
- docs/runbook updates

- [ ] **Step 1: Add feature flag for governance routing where partial rollback is still possible**
- [ ] **Step 2: Add metrics (`approval_required_rate`, `approval_latency`, `replay_rejection_count`, audit latency)**
- [ ] **Step 3: Define staged rollout plan (non-prod -> canary -> full)**
- [ ] **Step 4: Document rollback procedure for approvals, audits, and frontend compatibility**

Acceptance:
- Governance rollout is observable, staged, and operationally reversible where feasible.

---

## Definition Of Done

- [ ] Governance interfaces represent namespace, team, and environment policy context directly.
- [ ] The blueprint envelope is the only target backend write contract.
- [ ] Cluster history/detail reads only from `operation_audits`.
- [ ] Active cluster approval APIs are governed before migrated write endpoints depend on them.
- [ ] Cluster high-risk operations are migrated to governance adapter/executor flow.
- [ ] Cluster-local lifecycle ownership is removed or reduced to explicit compatibility shims.
- [ ] Frontend is first compatible with both worlds, then cleaned up to rely only on blueprint-shaped payloads.
- [ ] Regression tests validate approval lifecycle, replay protection, scope binding, namespace/team/environment authorization, audit redaction, and history/detail parity.
