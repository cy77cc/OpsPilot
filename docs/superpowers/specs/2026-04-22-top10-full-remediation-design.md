# OpsPilot Top 10 Full Remediation Design

- Date: 2026-04-22
- Scope: one-shot remediation for all Top 10 issues listed in `docs/engineering-code-review-report-2026-04-21.md`
- Delivery mode: single branch, layered commits, one final integration
- Compatibility policy: hard-cut allowed; no backward-compatibility bridge required

## 1. Goal

Complete all Top 10 frontend + backend findings in one integrated delivery while keeping engineering risk controllable through layered commit sequencing.

Top-level outcomes:

1. Frontend governance boundary is enforceable (lint/type/refresh/storage rules).
2. Frontend monolith files are decomposed into clear orchestration + domain + presentation units.
3. Backend startup and AI tool initialization no longer depend on hard-fail `log.Fatalf` or runtime `panic(err)`.
4. Backend auth error semantics and RBAC handler boundaries are stable and testable.

## 2. Constraints and Non-goals

### 2.1 Constraints

1. All Top 10 items are in-scope in this single delivery.
2. No compatibility requirement for legacy behavior.
3. Final merge is one integrated closeout, but implementation proceeds as layered commits.
4. Existing public routes remain stable unless explicitly required by remediations.

### 2.2 Non-goals

1. This design does not attempt full-repo architecture modernization beyond Top 10.
2. This design does not include unrelated feature work.
3. This design does not force global `strict: true` for the entire frontend repository in this cycle.

## 3. Delivery Strategy (Option 2, Confirmed)

Single branch, layered commits, one final integration.

Execution layers:

1. Frontend governance baseline
2. Frontend structural decomposition
3. Backend stability hardening
4. Backend structural decomposition

Why this strategy:

1. Meets one-shot full remediation requirement.
2. Preserves traceability and rollback granularity.
3. Reduces integration risk versus unconstrained parallel refactors.

## 4. Architecture and Boundary Design

### 4.1 Frontend governance baseline

1. Keep global `web/tsconfig.app.json` behavior unchanged for now, but enforce strict checks on boundary and newly decomposed files.
2. Establish minimal ESLint baseline plus boundary-specific restrictions.
3. Hard-ban direct auth localStorage access from auth/request/transport boundaries.
4. Keep one refresh gate (`refreshPromise`) in `web/src/api/api.ts`; remove module-level refresh branches.

### 4.2 Frontend decomposition model

1. `ClusterDetailPage.tsx` becomes orchestration-only container plus extracted domain hooks and panels.
2. `CopilotSurface.tsx` moves session domain state to reducer-driven model; stream handling moves to dedicated hook(s).
3. `NotificationContext.tsx` is split into data/WS/approval providers to limit render fan-out.
4. K8s components separate display/editor responsibilities; API side effects move upward to hooks/service.

### 4.3 Backend stability model

1. Configuration loading path returns `error` instead of invoking `log.Fatalf`.
2. AI tool registration/initialization converts `panic(err)` into explicit failure handling with degraded registration.
3. Casbin middleware audit output uses structured logging with consistent trace fields.
4. Authentication handlers map domain errors to stable categories for frontend handling.

### 4.4 Backend structural decomposition

1. Split `internal/modules/rbac/handler/permission.go` into focused handler files (`user`, `role`, `permission`, `audit`).
2. Keep route contracts stable while reducing per-file responsibility.
3. Extract shared helpers for request binding, permission checks, and response mapping.

## 5. Unified Data and Error Flow

### 5.1 Session and scope flow

1. Frontend session is cookie + `/auth/me` driven, memory-resident for runtime state.
2. `projectId/teamId` are owned by a single `ScopeStore`; no direct page/module storage read.
3. Request context headers are injected from unified request-context layer.

### 5.2 Refresh and retry flow

1. Any auth-expired condition routes to `api.ts` refresh gate.
2. Only one in-flight refresh promise exists.
3. Refresh failure transitions session to logged-out state deterministically.

### 5.3 Error semantics

1. Backend emits stable error categories (auth/validation/permission/system).
2. Frontend consumes normalized error shape via shared API error model.
3. Mixed ad hoc patterns (`message.error`/`console.error`/silent swallow) are replaced by unified error handling path.

## 6. File-level Change Blueprint

### 6.1 Frontend

1. `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx` (decompose)
2. `web/src/components/AI/CopilotSurface.tsx` (decompose)
3. `web/src/contexts/NotificationContext.tsx` (split providers)
4. `web/src/api/api.ts` (single refresh gate hardening)
5. `web/src/components/Auth/AuthContext.tsx` and related boundary files (remove auth storage penetration)
6. `web/src/components/K8s/*.tsx` (display/editor + side effect separation)
7. `web/tsconfig.auth-scope.json`, `web/eslint.config.js`, CI workflow entries (quality gates)

### 6.2 Backend

1. `internal/core/config/config.go` + startup callers (`MustNewConfig` to error-return flow)
2. `internal/modules/ai/agent/tools/**` (panic removal and degraded registration)
3. `internal/modules/rbac/handler/permission.go` (file split by responsibility)
4. `internal/core/middleware/casbin.go` (structured audit logging)
5. `internal/modules/user/handler/auth.go` and related error mapping points (stable code semantics)

## 7. Test and Verification Plan

### 7.1 Frontend gates

1. `npm run lint:auth-scope`
2. `npm run typecheck:auth-scope`
3. Targeted regression tests for auth/scope/refresh and decomposed major components.

### 7.2 Backend gates

1. Targeted Go tests for affected modules and middleware.
2. Startup-path tests validating error-return behavior.
3. AI tool initialization tests validating non-panic degraded behavior.
4. RBAC handler routing and behavior regression tests after split.

### 7.3 Integration gates

1. CI must block on frontend governance gate failures.
2. CI must pass backend module-level tests for changed paths.
3. No unresolved Top 10 item remains without direct code evidence.

## 8. Risks and Mitigations

1. Risk: broad one-shot scope causes merge-time instability.
   Mitigation: strict layer sequence and per-layer verification before proceeding.
2. Risk: decomposition introduces behavior drift.
   Mitigation: keep route/API contracts stable and add focused regression tests.
3. Risk: panic removal can hide failures.
   Mitigation: explicit degraded state + clear logs/metrics for unavailable tools.
4. Risk: frontend boundary list drift over time.
   Mitigation: treat boundary file list updates as required when expanding auth/scope/request paths.

## 9. Exit Criteria

This remediation is complete only when all conditions hold:

1. All Top 10 findings are addressed with concrete code changes.
2. Frontend governance gates are enforced and green.
3. Backend has no runtime `panic(err)` in AI tools initialization paths and no startup `log.Fatalf` in config load path.
4. RBAC handler split is complete and behaviorally stable.
5. Unified auth refresh and error semantics are validated by tests.
