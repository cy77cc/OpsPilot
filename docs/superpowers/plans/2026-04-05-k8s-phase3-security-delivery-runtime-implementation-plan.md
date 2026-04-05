# Phase 3 Security Delivery Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver Phase 3 (`A -> C -> B`) with production-gate controls: admission and image scanning, GitOps + app catalog delivery controls, and runtime security detection/disposal, while preserving existing governance and external cluster boundaries.

**Architecture:** Extend the existing cluster/governance service surfaces instead of creating parallel frameworks. Add Phase 3 resources and APIs under `internal/service/cluster`, persist policy/release/security artifacts in new tables, and reuse `OperationApproval`/`OperationAudit` for gate decisions and traceability. Split behavior by cluster source (`platform_managed` vs `external_managed`) so external clusters run detect-and-govern mode.

**Tech Stack:** Go + Gin + GORM + SQLite/MySQL migrations, existing governance service (`approval`, `audit`), Kubernetes/cluster service modules, React + TypeScript + Vitest for control-plane UI.

---

## File Structure (Lock Before Tasking)

- Create: `storage/migrations/20260405_0002_create_phase3_security_tables.sql`  
  Responsibility: persistent schema for admission policies/exemptions, scan reports, gitops releases, runtime events/actions.
- Create: `internal/model/phase3_security.go`  
  Responsibility: GORM models and enums for Phase 3 entities.
- Create: `internal/service/cluster/phase3_types.go`  
  Responsibility: API/domain DTOs and shared enums (`cluster_mode`, `disposal_mode`, `gate_decision`).
- Create: `internal/service/cluster/phase3_repository.go`  
  Responsibility: CRUD/query methods for Phase 3 tables and audit correlation.
- Create: `internal/service/cluster/handler_phase3_admission.go`  
  Responsibility: admission policy + exemption APIs.
- Create: `internal/service/cluster/handler_phase3_gitops.go`  
  Responsibility: app registration/sync/rollback APIs and release records.
- Create: `internal/service/cluster/handler_phase3_runtime.go`  
  Responsibility: runtime alert query, resolve, contain behavior with cluster-source branching.
- Modify: `internal/service/cluster/routes.go`  
  Responsibility: register Phase 3 routes under existing `/clusters/:id/...` namespace.
- Create: `internal/service/cluster/phase3_audit.go`  
  Responsibility: centralized helpers to write `OperationAudit` and gate `OperationApproval`.
- Create: `internal/service/cluster/handler_phase3_admission_test.go`
- Create: `internal/service/cluster/handler_phase3_gitops_test.go`
- Create: `internal/service/cluster/handler_phase3_runtime_test.go`
- Create: `internal/service/cluster/phase3_repository_test.go`
- Create: `web/src/api/modules/cluster.phase3.ts`
- Create: `web/src/api/modules/cluster.phase3.test.ts`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`
- Modify: `docs/runbooks/cluster-policy-release-and-rollback.md`
- Create: `docs/runbooks/phase3-security-delivery-runtime-operations.md`

## Task 1: Add Phase 3 Persistent Schema

**Files:**
- Create: `storage/migrations/20260405_0002_create_phase3_security_tables.sql`
- Test: `internal/service/cluster/phase3_repository_test.go`

- [ ] **Step 1: Write failing repository migration test**

```go
func TestPhase3Repository_AutoMigrateShape(t *testing.T) {
	db := testSQLite(t)
	if err := db.AutoMigrate(&model.AdmissionPolicy{}, &model.AdmissionExemption{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Repository_AutoMigrateShape"`  
Expected: FAIL with missing model/table references.

- [ ] **Step 3: Create migration SQL**

```sql
CREATE TABLE IF NOT EXISTS admission_policies (...);
CREATE TABLE IF NOT EXISTS admission_exemptions (...);
CREATE TABLE IF NOT EXISTS image_scan_reports (...);
CREATE TABLE IF NOT EXISTS gitops_app_releases (...);
CREATE TABLE IF NOT EXISTS runtime_security_events (...);
CREATE TABLE IF NOT EXISTS runtime_disposal_actions (...);
```

- [ ] **Step 4: Run targeted tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Repository_AutoMigrateShape"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add storage/migrations/20260405_0002_create_phase3_security_tables.sql internal/service/cluster/phase3_repository_test.go
git commit -m "Persist phase-3 entities for admission gitops and runtime workflows"
```

## Task 2: Add GORM Models and Shared Types

**Files:**
- Create: `internal/model/phase3_security.go`
- Create: `internal/service/cluster/phase3_types.go`
- Test: `internal/service/cluster/phase3_repository_test.go`

- [ ] **Step 1: Write failing type-usage test**

```go
func TestPhase3Model_EnumsStable(t *testing.T) {
	if model.DisposalModeSuggestOnly == "" {
		t.Fatalf("enum missing")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Model_EnumsStable"`  
Expected: FAIL undefined enum/type.

- [ ] **Step 3: Add models and enums**

```go
type AdmissionPolicy struct { /* policy_name, cluster_id, version, status */ }
type AdmissionExemption struct { /* scope_type, scope_ref, expires_at, approval_id */ }
type RuntimeSecurityEvent struct { /* severity, workload, raw_payload */ }
const DisposalModeSuggestOnly = "suggest_only"
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Model_EnumsStable|Phase3Repository_AutoMigrateShape"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/phase3_security.go internal/service/cluster/phase3_types.go internal/service/cluster/phase3_repository_test.go
git commit -m "Define phase-3 models and enums for gate and runtime records"
```

## Task 3: Implement Phase 3 Repository with Governance Correlation

**Files:**
- Create: `internal/service/cluster/phase3_repository.go`
- Modify: `internal/service/cluster/repository.go`
- Test: `internal/service/cluster/phase3_repository_test.go`

- [ ] **Step 1: Write failing repository behavior tests**

```go
func TestPhase3Repository_SaveAdmissionExemption(t *testing.T) { /* save + fetch */ }
func TestPhase3Repository_LinkRuntimeActionToAudit(t *testing.T) { /* audit_id roundtrip */ }
```

- [ ] **Step 2: Run tests to confirm failure**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Repository_SaveAdmissionExemption|Phase3Repository_LinkRuntimeActionToAudit"`  
Expected: FAIL undefined repository methods.

- [ ] **Step 3: Implement repository methods**

```go
func (r *Repository) CreateAdmissionExemption(...) error
func (r *Repository) ListRuntimeSecurityEvents(...) ([]model.RuntimeSecurityEvent, error)
func (r *Repository) CreateRuntimeDisposalAction(...) error
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Repository_"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/phase3_repository.go internal/service/cluster/repository.go internal/service/cluster/phase3_repository_test.go
git commit -m "Add phase-3 repository primitives with audit correlation support"
```

## Task 4: Centralize Phase 3 Audit and Approval Reuse

**Files:**
- Create: `internal/service/cluster/phase3_audit.go`
- Modify: `internal/service/cluster/approval_policy.go`
- Test: `internal/service/cluster/handler_phase3_admission_test.go`

- [ ] **Step 1: Write failing audit/approval helper test**

```go
func TestPhase3Gate_ApprovalRequiredCreatesPendingResponse(t *testing.T) { /* expect approval_required */ }
```

- [ ] **Step 2: Run test to verify failure**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Gate_ApprovalRequiredCreatesPendingResponse"`  
Expected: FAIL missing helper behavior.

- [ ] **Step 3: Implement helper functions**

```go
func (h *Handler) phase3RecordAudit(...)
func (h *Handler) phase3RequireApproval(...)
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Gate_|PolicyReleaseApproval"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/phase3_audit.go internal/service/cluster/approval_policy.go internal/service/cluster/handler_phase3_admission_test.go
git commit -m "Reuse governance approval and audit flows for phase-3 gate decisions"
```

## Task 5: Implement Admission Policy and Exemption APIs (A Domain)

**Files:**
- Create: `internal/service/cluster/handler_phase3_admission.go`
- Modify: `internal/service/cluster/routes.go`
- Test: `internal/service/cluster/handler_phase3_admission_test.go`

- [ ] **Step 1: Write failing handler tests for A APIs**

```go
func TestHandlerPhase3Admission_RegisterPolicy(t *testing.T) {}
func TestHandlerPhase3Admission_CreateExemptionRequiresApproval(t *testing.T) {}
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Admission"`  
Expected: FAIL missing routes/handlers.

- [ ] **Step 3: Implement handlers and route registration**

```go
clusterGroup.POST("/:id/admission/policies", h.UpsertAdmissionPolicy)
clusterGroup.POST("/:id/admission/exemptions", h.CreateAdmissionExemption)
clusterGroup.GET("/:id/admission/results", h.ListAdmissionResults)
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Admission|Route"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/handler_phase3_admission.go internal/service/cluster/routes.go internal/service/cluster/handler_phase3_admission_test.go
git commit -m "Expose phase-3 admission and exemption APIs with governance gating"
```

## Task 6: Implement GitOps App APIs (C Domain)

**Files:**
- Create: `internal/service/cluster/handler_phase3_gitops.go`
- Test: `internal/service/cluster/handler_phase3_gitops_test.go`
- Modify: `internal/service/cluster/routes.go`

- [ ] **Step 1: Write failing GitOps handler tests**

```go
func TestHandlerPhase3GitOps_RegisterApp(t *testing.T) {}
func TestHandlerPhase3GitOps_SyncAndRollback(t *testing.T) {}
```

- [ ] **Step 2: Run tests to confirm failure**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3GitOps"`  
Expected: FAIL missing handlers.

- [ ] **Step 3: Implement GitOps handlers**

```go
POST /:id/apps
POST /:id/apps/:name/sync
POST /:id/apps/:name/rollback
GET  /:id/apps/:name
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3GitOps|Route"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/handler_phase3_gitops.go internal/service/cluster/handler_phase3_gitops_test.go internal/service/cluster/routes.go
git commit -m "Add phase-3 gitops app registration sync and rollback APIs"
```

## Task 7: Implement Runtime Security APIs (B Domain) with Cluster-Source Branching

**Files:**
- Create: `internal/service/cluster/handler_phase3_runtime.go`
- Modify: `internal/service/cluster/logic_resources.go`
- Test: `internal/service/cluster/handler_phase3_runtime_test.go`

- [ ] **Step 1: Write failing runtime behavior tests**

```go
func TestHandlerPhase3Runtime_ContainPlatformManagedExecutesAction(t *testing.T) {}
func TestHandlerPhase3Runtime_ContainExternalManagedReturnsSuggestOnly(t *testing.T) {}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Runtime"`  
Expected: FAIL missing runtime handlers/branching.

- [ ] **Step 3: Implement runtime handlers**

```go
GET  /:id/security/alerts
GET  /:id/security/events/:event_id
POST /:id/security/alerts/:alert_id/resolve
POST /:id/security/alerts/:alert_id/contain
```

- [ ] **Step 4: Enforce cluster-source mode**

```go
if cluster.Source != "platform_managed" {
  action.Mode = "suggest_only"
}
```

- [ ] **Step 5: Run tests and commit**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Runtime|external_managed|platform_managed"`  
Expected: PASS.

```bash
git add internal/service/cluster/handler_phase3_runtime.go internal/service/cluster/logic_resources.go internal/service/cluster/handler_phase3_runtime_test.go
git commit -m "Implement runtime alert APIs with external managed suggest-only containment"
```

## Task 8: Add Frontend API Client for Phase 3 Surfaces

**Files:**
- Create: `web/src/api/modules/cluster.phase3.ts`
- Create: `web/src/api/modules/cluster.phase3.test.ts`
- Modify: `web/src/api/modules/cluster.ts`

- [ ] **Step 1: Write failing API decode tests**

```ts
it('maps runtime contain suggest_only mode');
it('maps admission exemption approval_required response');
```

- [ ] **Step 2: Run test and verify failure**

Run: `cd web && npx vitest run src/api/modules/cluster.phase3.test.ts`  
Expected: FAIL module missing.

- [ ] **Step 3: Implement API module**

```ts
export const phase3Api = {
  createAdmissionPolicy, createAdmissionExemption, listAdmissionResults,
  registerApp, syncApp, rollbackApp,
  listSecurityAlerts, resolveAlert, containAlert
}
```

- [ ] **Step 4: Run tests**

Run: `cd web && npx vitest run src/api/modules/cluster.phase3.test.ts src/api/modules/cluster.policy.test.ts`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/cluster.phase3.ts web/src/api/modules/cluster.phase3.test.ts web/src/api/modules/cluster.ts
git commit -m "Add phase-3 cluster API client for admission gitops and runtime flows"
```

## Task 9: Build Cluster Security Center Page

**Files:**
- Create: `web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`

- [ ] **Step 1: Write failing page tests**

```ts
it('renders admission gitops runtime tabs');
it('shows suggest_only badge for external_managed containment');
```

- [ ] **Step 2: Run tests and confirm failure**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx --testTimeout=60000`  
Expected: FAIL page missing.

- [ ] **Step 3: Implement page and route**

```tsx
<Tabs items={[AdmissionTab, GitOpsTab, RuntimeTab]} />
```

- [ ] **Step 4: Run tests**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx --testTimeout=60000`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.tsx web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx web/src/ProtectedApp.tsx
git commit -m "Introduce phase-3 security center UI for admission gitops and runtime governance"
```

## Task 10: Document Runbooks and Production Gate Checklist

**Files:**
- Create: `docs/runbooks/phase3-security-delivery-runtime-operations.md`
- Modify: `docs/runbooks/cluster-policy-release-and-rollback.md`
- Modify: `docs/superpowers/specs/2026-04-05-k8s-phase3-security-delivery-runtime-design.md`

- [ ] **Step 1: Write failing doc-check test command**

Run: `rg -n "suggest_only|admission_exemptions|gitops_app_releases|runtime_security_events" docs/runbooks docs/superpowers/specs -S`  
Expected: Missing matches.

- [ ] **Step 2: Add runbook sections**

```md
Prerequisites | Release Flow | Rollback Flow | Runtime Containment | external_managed Downgrade Rules
```

- [ ] **Step 3: Add production gate checklist**

```md
CVSS gate thresholds, p95 latency targets, rollback SLO, drill evidence checklist
```

- [ ] **Step 4: Re-run doc checks**

Run: `rg -n "suggest_only|admission_exemptions|gitops_app_releases|runtime_security_events|CVSS" docs/runbooks docs/superpowers/specs -S`  
Expected: Non-empty matches for each key.

- [ ] **Step 5: Commit**

```bash
git add docs/runbooks/phase3-security-delivery-runtime-operations.md docs/runbooks/cluster-policy-release-and-rollback.md docs/superpowers/specs/2026-04-05-k8s-phase3-security-delivery-runtime-design.md
git commit -m "Document phase-3 operating model and production gate acceptance checklist"
```

## Task 11: Focused Verification and Release Record

**Files:**
- Modify: `docs/superpowers/specs/2026-04-05-k8s-phase3-security-delivery-runtime-design.md`
- Modify: `docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md`

- [ ] **Step 1: Run backend focused regression**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster/... ./internal/service/governance/...`  
Expected: PASS.

- [ ] **Step 2: Run frontend focused regression**

Run: `cd web && npx vitest run src/api/modules/cluster.phase3.test.ts src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx --testTimeout=60000`  
Expected: PASS.

- [ ] **Step 3: Run full-gate visibility commands**

Run: `GOCACHE=/tmp/go-build-cache go test ./... && cd web && npm run test && npm run build`  
Expected: Record existing out-of-scope failures explicitly if present.

- [ ] **Step 4: Write release-readiness notes**

```md
Focused suites pass/fail, full-gate blockers, rollback readiness, unresolved risks.
```

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/specs/2026-04-05-k8s-phase3-security-delivery-runtime-design.md docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md
git commit -m "Record phase-3 readiness evidence and non-scope gate blockers"
```

## Self-Review Summary

- Spec coverage: A/C/B + governance reuse + external cluster mode + API/model/deployment/DoD are mapped to tasks.
- Placeholder scan: no `TBD`/`TODO` steps; each task has explicit commands and expected outcomes.
- Type consistency: enum/resource names are consistent across tasks (`suggest_only`, `admission_exemptions`, `gitops_app_releases`, `runtime_security_events`).
