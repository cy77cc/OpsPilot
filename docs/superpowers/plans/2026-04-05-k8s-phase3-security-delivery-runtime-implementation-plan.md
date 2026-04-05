# Phase 3 Security Delivery Runtime Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver Phase 3 (`A -> C -> B`) with real component integrations (Harbor/Trivy/ArgoCD/Falco/Tetragon), governance-native approval/audit flow reuse, and production-gate controls for both `platform_managed` and `external_managed` clusters.

**Architecture:** Build Phase 3 in `internal/service/cluster` by extending existing `Repository` + `Handler` patterns, reuse `governance.Service` (`Preflight`/`Finalize`) as the gate engine, and persist domain state with dedicated Phase 3 tables. Integration clients are explicit adapters (`harbor`, `trivy`, `argocd`, `runtime_ingest`) so API handlers are orchestration-only. `external_managed` clusters run detect-and-govern mode: automatic containment downgrades to audited suggestion + manual approval flow.

**Tech Stack:** Go + Gin + GORM + SQL migrations, existing governance service modules, HTTP clients to Harbor/ArgoCD/Trivy gateways, Falco/Tetragon event ingestion endpoints, React + TypeScript + Vitest.

---

## File Structure (Lock Before Tasking)

- Create: `storage/migrations/20260405_0002_create_phase3_security_tables.sql`
- Create: `internal/model/phase3_security.go`
- Create: `internal/service/cluster/phase3_types.go`
- Create: `internal/service/cluster/phase3_repository.go`
- Create: `internal/service/cluster/phase3_audit.go`
- Create: `internal/service/cluster/integration/harbor_client.go`
- Create: `internal/service/cluster/integration/trivy_client.go`
- Create: `internal/service/cluster/integration/argocd_client.go`
- Create: `internal/service/cluster/integration/runtime_ingest.go`
- Create: `internal/service/cluster/handler_phase3_admission.go`
- Create: `internal/service/cluster/handler_phase3_gitops.go`
- Create: `internal/service/cluster/handler_phase3_runtime.go`
- Create: `internal/service/cluster/logic_phase3_consistency.go`
- Create: `internal/service/cluster/logic_phase3_slo.go`
- Modify: `internal/service/cluster/repository.go`
- Modify: `internal/service/cluster/routes.go`
- Modify: `internal/service/cluster/handler.go`
- Test: `internal/service/cluster/phase3_repository_test.go`
- Test: `internal/service/cluster/handler_phase3_admission_test.go`
- Test: `internal/service/cluster/handler_phase3_gitops_test.go`
- Test: `internal/service/cluster/handler_phase3_runtime_test.go`
- Test: `internal/service/cluster/integration_clients_test.go`
- Test: `internal/service/cluster/logic_phase3_consistency_test.go`
- Test: `internal/service/cluster/logic_phase3_slo_test.go`
- Create: `web/src/api/modules/cluster.phase3.ts`
- Create: `web/src/api/modules/cluster.phase3.test.ts`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`
- Create: `docs/runbooks/phase3-security-delivery-runtime-operations.md`
- Modify: `docs/superpowers/specs/2026-04-05-k8s-phase3-security-delivery-runtime-design.md`

## Task 1: Lock Repository Extension Pattern and Dependency Injection

**Files:**
- Modify: `internal/service/cluster/repository.go`
- Modify: `internal/service/cluster/handler.go`
- Test: `internal/service/cluster/phase3_repository_test.go`

- [ ] **Step 1: Write failing architecture test**

```go
func TestPhase3Repository_ExtendsBaseRepository(t *testing.T) {
	repo := NewRepository(testDB(t))
	if repo == nil {
		t.Fatalf("repo is nil")
	}
	if repo.Phase3 == nil {
		t.Fatalf("phase3 repository extension missing")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Repository_ExtendsBaseRepository"`  
Expected: FAIL with missing `Phase3` extension or missing constructor wiring.

- [ ] **Step 3: Implement extension wiring**

```go
type Repository struct {
	db *gorm.DB
	Phase3 *Phase3Repository
}
```

```go
func NewRepository(db *gorm.DB) *Repository {
	r := &Repository{db: db}
	r.Phase3 = NewPhase3Repository(db)
	return r
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Repository_ExtendsBaseRepository"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/repository.go internal/service/cluster/handler.go internal/service/cluster/phase3_repository_test.go
git commit -m "Establish phase-3 repository extension model for cluster service"
```

## Task 2: Add Phase 3 Schema with Verifiable Structure

**Files:**
- Create: `storage/migrations/20260405_0002_create_phase3_security_tables.sql`
- Test: `internal/service/cluster/phase3_repository_test.go`

- [ ] **Step 1: Write failing schema structure tests**

```go
func TestPhase3Schema_AdmissionPolicyColumns(t *testing.T) {}
func TestPhase3Schema_RuntimeEventIndexes(t *testing.T) {}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Schema_"`  
Expected: FAIL because tables/columns/indexes are missing.

- [ ] **Step 3: Add migration SQL**

```sql
CREATE TABLE admission_policies (..., cluster_id BIGINT, policy_name VARCHAR(191), version VARCHAR(64), status VARCHAR(32), ...);
CREATE INDEX idx_admission_policies_cluster_name ON admission_policies(cluster_id, policy_name);
CREATE TABLE runtime_security_events (..., cluster_id BIGINT, severity VARCHAR(32), source VARCHAR(32), ...);
CREATE INDEX idx_runtime_security_events_cluster_severity ON runtime_security_events(cluster_id, severity);
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Schema_"`  
Expected: PASS with column and index assertions.

- [ ] **Step 5: Commit**

```bash
git add storage/migrations/20260405_0002_create_phase3_security_tables.sql internal/service/cluster/phase3_repository_test.go
git commit -m "Add validated phase-3 schema for admission gitops and runtime domains"
```

## Task 3: Define Models and Domain Types

**Files:**
- Create: `internal/model/phase3_security.go`
- Create: `internal/service/cluster/phase3_types.go`
- Test: `internal/service/cluster/phase3_repository_test.go`

- [ ] **Step 1: Write failing enum/model tests**

```go
func TestPhase3Model_RequiredEnums(t *testing.T) {}
func TestPhase3Model_JSONRoundTrip(t *testing.T) {}
```

- [ ] **Step 2: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Model_"`  
Expected: FAIL with undefined model/enum names.

- [ ] **Step 3: Implement models and DTO enums**

```go
const (
	DisposalModeAuto        = "auto"
	DisposalModeManual      = "manual"
	DisposalModeSuggestOnly = "suggest_only"
)
type AdmissionExemption struct { ScopeType string; ScopeRef string; ExpiresAt time.Time; ApprovalID uint }
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Model_"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/model/phase3_security.go internal/service/cluster/phase3_types.go internal/service/cluster/phase3_repository_test.go
git commit -m "Define phase-3 data models and stable domain enums"
```

## Task 4: Implement Real Integration Clients (Harbor/Trivy/ArgoCD/Falco-Tetragon Ingest)

**Files:**
- Create: `internal/service/cluster/integration/harbor_client.go`
- Create: `internal/service/cluster/integration/trivy_client.go`
- Create: `internal/service/cluster/integration/argocd_client.go`
- Create: `internal/service/cluster/integration/runtime_ingest.go`
- Test: `internal/service/cluster/integration_clients_test.go`

- [ ] **Step 1: Write failing integration adapter tests**

```go
func TestHarborClient_ListProjects(t *testing.T) {}
func TestTrivyClient_ScanImage(t *testing.T) {}
func TestArgoCDClient_SyncApplication(t *testing.T) {}
func TestRuntimeIngest_ParseFalcoAndTetragon(t *testing.T) {}
```

- [ ] **Step 2: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HarborClient_|TrivyClient_|ArgoCDClient_|RuntimeIngest_"`  
Expected: FAIL with missing clients/parsers.

- [ ] **Step 3: Implement integration clients**

```go
type HarborClient interface { ListProjects(ctx context.Context) ([]HarborProject, error) }
type TrivyClient interface { ScanImage(ctx context.Context, image string) (TrivyScanResult, error) }
type ArgoCDClient interface { Sync(ctx context.Context, app string) (ArgoSyncResult, error) }
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HarborClient_|TrivyClient_|ArgoCDClient_|RuntimeIngest_"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/integration/harbor_client.go internal/service/cluster/integration/trivy_client.go internal/service/cluster/integration/argocd_client.go internal/service/cluster/integration/runtime_ingest.go internal/service/cluster/integration_clients_test.go
git commit -m "Integrate phase-3 external clients for harbor trivy argocd and runtime event ingest"
```

## Task 5: Implement Governance Service Bridge (Real Preflight/Finalize)

**Files:**
- Create: `internal/service/cluster/phase3_audit.go`
- Modify: `internal/service/cluster/handler.go`
- Test: `internal/service/cluster/handler_phase3_admission_test.go`

- [ ] **Step 1: Write failing governance bridge tests**

```go
func TestPhase3Governance_PreflightApprovalRequired(t *testing.T) {}
func TestPhase3Governance_FinalizeWritesAuditEnvelope(t *testing.T) {}
```

- [ ] **Step 2: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Governance_"`  
Expected: FAIL with missing `Preflight`/`Finalize` bridge.

- [ ] **Step 3: Implement governance calls**

```go
decision, err := h.svcCtx.GovernanceService.Preflight(ctx, intent)
out, err := h.svcCtx.GovernanceService.Finalize(ctx, finalizeInput)
```

```go
intent.Action = "admission.apply" // or gitops.sync/runtime.contain
intent.Scope = governance.Scope{ClusterID: clusterID, Namespace: namespace}
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3Governance_|PolicyReleaseApproval"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/phase3_audit.go internal/service/cluster/handler.go internal/service/cluster/handler_phase3_admission_test.go
git commit -m "Bridge phase-3 gates to governance preflight and finalize pipeline"
```

## Parallel Execution Window (After Task 5)

The following tasks are independent and SHOULD run in parallel with separate workers:

- Task 6 (A Domain)
- Task 7 (C Domain)
- Task 8 (B Domain)

Merge order after completion: Task 6 -> Task 7 -> Task 8.

## Task 6: Implement Admission Domain with Harbor/Trivy Gate Decisions (A)

**Files:**
- Create: `internal/service/cluster/handler_phase3_admission.go`
- Modify: `internal/service/cluster/routes.go`
- Test: `internal/service/cluster/handler_phase3_admission_test.go`

- [ ] **Step 1: Write failing A-domain tests**

```go
func TestHandlerPhase3Admission_RegistersPolicyAndVersion(t *testing.T) {}
func TestHandlerPhase3Admission_BlocksCriticalVulnFromTrivy(t *testing.T) {}
func TestHandlerPhase3Admission_ExemptionRequiresApproval(t *testing.T) {}
```

- [ ] **Step 2: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Admission_|Route.*Admission"`  
Expected: FAIL missing handler logic and route registration.

- [ ] **Step 3: Implement handlers using clients and governance**

```go
clusterGroup.POST("/:id/admission/policies", h.UpsertAdmissionPolicy)
clusterGroup.GET("/:id/admission/results", h.ListAdmissionResults)
clusterGroup.POST("/:id/admission/exemptions", h.CreateAdmissionExemption)
clusterGroup.POST("/:id/admission/exemptions/:exemption_id/revoke", h.RevokeAdmissionExemption)
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Admission_|Route.*Admission"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/handler_phase3_admission.go internal/service/cluster/routes.go internal/service/cluster/handler_phase3_admission_test.go
git commit -m "Implement admission APIs with trivy-backed gating and approved exemptions"
```

## Task 7: Implement GitOps Domain with ArgoCD and Drift Control (C)

**Files:**
- Create: `internal/service/cluster/handler_phase3_gitops.go`
- Create: `internal/service/cluster/logic_phase3_consistency.go`
- Modify: `internal/service/cluster/routes.go`
- Test: `internal/service/cluster/handler_phase3_gitops_test.go`
- Test: `internal/service/cluster/logic_phase3_consistency_test.go`

- [ ] **Step 1: Write failing C-domain tests**

```go
func TestHandlerPhase3GitOps_SyncCallsArgoAndRecordsRelease(t *testing.T) {}
func TestConsistency_DetectsDriftAndSchedulesRemediation(t *testing.T) {}
func TestHandlerPhase3GitOps_TripsCircuitBreakerOnConsecutiveFailures(t *testing.T) {}
```

- [ ] **Step 2: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3GitOps_|Consistency_|CircuitBreaker_"`  
Expected: FAIL missing logic.

- [ ] **Step 3: Implement handlers and drift logic**

```go
POST /:id/apps
GET  /:id/apps/:name
POST /:id/apps/:name/sync
POST /:id/apps/:name/rollback
```

```go
func EvaluateDrift(ctx context.Context, clusterID uint, appName string, desiredRevision string) (DriftResult, error)
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3GitOps_|Consistency_|CircuitBreaker_|Route.*apps"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/handler_phase3_gitops.go internal/service/cluster/logic_phase3_consistency.go internal/service/cluster/routes.go internal/service/cluster/handler_phase3_gitops_test.go internal/service/cluster/logic_phase3_consistency_test.go
git commit -m "Implement gitops APIs with argocd sync drift detection and failure breaker"
```

## Task 8: Implement Runtime Domain with Event Ingestion and Containment Modes (B)

**Files:**
- Create: `internal/service/cluster/handler_phase3_runtime.go`
- Modify: `internal/service/cluster/routes.go`
- Test: `internal/service/cluster/handler_phase3_runtime_test.go`

- [ ] **Step 1: Write failing B-domain tests**

```go
func TestHandlerPhase3Runtime_IngestFalcoEvent(t *testing.T) {}
func TestHandlerPhase3Runtime_ContainPlatformManagedAuto(t *testing.T) {}
func TestHandlerPhase3Runtime_ContainExternalManagedSuggestOnlyWithAudit(t *testing.T) {}
```

- [ ] **Step 2: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Runtime_|Route.*security"`  
Expected: FAIL missing ingestion/containment flows.

- [ ] **Step 3: Implement runtime handlers**

```go
POST /:id/security/events/ingest
GET  /:id/security/alerts
GET  /:id/security/events/:event_id
POST /:id/security/alerts/:alert_id/resolve
POST /:id/security/alerts/:alert_id/contain
```

- [ ] **Step 4: Implement external-managed downgrade with audited manual path**

```go
if cluster.Source != "platform_managed" {
	action.Mode = model.DisposalModeSuggestOnly
	// create OperationAudit + optional approval ticket for manual execute endpoint
}
```

- [ ] **Step 5: Run tests and commit**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Runtime_|external_managed|platform_managed|Route.*security"`  
Expected: PASS.

```bash
git add internal/service/cluster/handler_phase3_runtime.go internal/service/cluster/routes.go internal/service/cluster/handler_phase3_runtime_test.go
git commit -m "Implement runtime ingest and containment with managed-mode specific behavior"
```

## Task 9: Implement SLO/SLA Metrics and Alert Grading (G2)

**Files:**
- Create: `internal/service/cluster/logic_phase3_slo.go`
- Modify: `internal/service/cluster/policy_metrics.go`
- Test: `internal/service/cluster/logic_phase3_slo_test.go`

- [ ] **Step 1: Write failing SLO tests**

```go
func TestPhase3SLO_AdmissionLatencyBudget(t *testing.T) {}
func TestPhase3SLO_RuntimeDetectToAlertP95(t *testing.T) {}
```

- [ ] **Step 2: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3SLO_"`  
Expected: FAIL missing SLO calculator/metrics.

- [ ] **Step 3: Implement SLO calculators and metric emitters**

```go
func EvaluateAdmissionLatencyBudget(samples []time.Duration) SLOResult
func EvaluateContainmentSuccessRate(total, success int) SLOResult
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3SLO_|PolicyMetrics"`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/logic_phase3_slo.go internal/service/cluster/policy_metrics.go internal/service/cluster/logic_phase3_slo_test.go
git commit -m "Add phase-3 SLO evaluation and alert grading metrics"
```

## Task 10: Frontend Phase 3 API + Security Center UI (Happy + Error Paths)

**Files:**
- Create: `web/src/api/modules/cluster.phase3.ts`
- Create: `web/src/api/modules/cluster.phase3.test.ts`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`

- [ ] **Step 1: Write failing frontend tests**

```ts
it('maps suggest_only response as warning state');
it('maps approval_rejected response payload');
it('renders external_managed containment as manual suggestion');
```

- [ ] **Step 2: Run tests**

Run: `cd web && npx vitest run src/api/modules/cluster.phase3.test.ts src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx --testTimeout=60000`  
Expected: FAIL module/page missing.

- [ ] **Step 3: Implement API client and page**

```ts
containAlert(clusterId, alertId) // returns mode + audit_id + approval
```

```tsx
<Tag color="gold">suggest_only</Tag>
```

- [ ] **Step 4: Run tests**

Run: `cd web && npx vitest run src/api/modules/cluster.phase3.test.ts src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx --testTimeout=60000`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/cluster.phase3.ts web/src/api/modules/cluster.phase3.test.ts web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.tsx web/src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx web/src/ProtectedApp.tsx
git commit -m "Add phase-3 frontend API and security center with error-aware state mapping"
```

## Task 11: Runbooks and Structural Doc Checks

**Files:**
- Create: `docs/runbooks/phase3-security-delivery-runtime-operations.md`
- Modify: `docs/superpowers/specs/2026-04-05-k8s-phase3-security-delivery-runtime-design.md`

- [ ] **Step 1: Run failing doc structure check**

Run: `rg -n "^## " docs/runbooks/phase3-security-delivery-runtime-operations.md`  
Expected: FAIL file missing.

- [ ] **Step 2: Add required sections**

```md
## Prerequisites
## Release Flow
## Rollback Flow
## Runtime Containment
## external_managed Downgrade
## SLO/SLA Verification
```

- [ ] **Step 3: Add quantified acceptance mapping**

```md
CVSS>=9.0 block rule, admission p95<800ms, runtime detect->alert p95<30s
```

- [ ] **Step 4: Re-run doc checks**

Run: `rg -n "^## (Prerequisites|Release Flow|Rollback Flow|Runtime Containment|external_managed Downgrade|SLO/SLA Verification)$" docs/runbooks/phase3-security-delivery-runtime-operations.md`  
Expected: 6 matched headings.

- [ ] **Step 5: Commit**

```bash
git add docs/runbooks/phase3-security-delivery-runtime-operations.md docs/superpowers/specs/2026-04-05-k8s-phase3-security-delivery-runtime-design.md
git commit -m "Document phase-3 operational runbook with quantified gate verification"
```

## Task 12: Verification, A5 Multi-Cluster Consistency Evidence, and Release Record

**Files:**
- Modify: `docs/superpowers/specs/2026-04-05-k8s-phase3-security-delivery-runtime-design.md`
- Modify: `docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md`

- [ ] **Step 1: Run backend focused suite**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster/... ./internal/service/governance/... -run "Phase3|Admission|GitOps|Runtime|SLO|Consistency"`  
Expected: PASS.

- [ ] **Step 2: Run frontend focused suite**

Run: `cd web && npx vitest run src/api/modules/cluster.phase3.test.ts src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx --testTimeout=60000`  
Expected: PASS.

- [ ] **Step 3: Run full-gate visibility**

Run: `GOCACHE=/tmp/go-build-cache go test ./... && cd web && npm run test && npm run build`  
Expected: Record and classify non-phase3 blockers if any.

- [ ] **Step 4: Write readiness evidence**

```md
Focused pass matrix, A5 consistency evidence, C4 drift evidence, G2 SLO snapshots, unresolved global blockers
```

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/specs/2026-04-05-k8s-phase3-security-delivery-runtime-design.md docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md
git commit -m "Record phase-3 implementation evidence including consistency drift and slo gates"
```

## Self-Review Summary

- Spec coverage: includes A1/A2/A5, C2/C4/C5, B1/B2/B5, and G2.
- Integration reality: explicit Harbor/Trivy/ArgoCD/Falco/Tetragon integration tasks are present before API orchestration tasks.
- Governance alignment: plan calls `governance.Service.Preflight/Finalize` and reuses approval/audit primitives.
- Placeholder scan: no `TBD`/`TODO`; every task includes commands and expected outcomes.
