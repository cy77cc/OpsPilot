# Frontend Governance Stopgap Design (A + 1 + a)

- Date: 2026-04-22
- Scope: Frontend governance stopgap for auth/scope/request boundaries
- Baseline: Keep global `strict: false`; enforce strict gates only on boundary files

## 1. Context

`docs/engineering-code-review-report-2026-04-21.md` exposed unresolved frontend governance risks:

1. Global TypeScript strict mode is still disabled.
2. ESLint governance exists but still has a baseline gap and is not enforced in CI for this boundary.
3. Current CI only runs backend route contract checks and does not block boundary regressions in frontend auth/scope/request paths.

This change intentionally delivers a low-risk stopgap, not a full repository-wide quality reset.

## 2. Goals

1. Preserve current development throughput by avoiding full-repo strict migration.
2. Add hard quality gates for auth/scope/request boundary files.
3. Enforce boundary lint and boundary strict typecheck in CI as merge blockers.

## 3. Non-goals

1. Do not switch `web/tsconfig.app.json` to global `strict: true`.
2. Do not make full-repo lint/typecheck blocking in CI.
3. Do not include structural refactors (for example `ClusterDetailPage` or `NotificationContext`) in this stopgap.
4. Do not modify frontend business behavior.

## 4. Approaches Considered

### Option 1 (Chosen): Minimal Closeout

- Keep global `strict: false`.
- Keep and strengthen boundary-scoped strict/typecheck entry via `tsconfig.auth-scope.json`.
- Keep and strengthen boundary-scoped lint rules in `eslint.config.js`.
- Add a frontend boundary governance job in CI:
  - `npm run lint:auth-scope`
  - `npm run typecheck:auth-scope`

Why chosen:
- Maximum risk reduction per unit change.
- Minimal blast radius.
- Fits progressive-hardening strategy.

### Option 2: Observability-first

- Same as Option 1, plus non-blocking full-repo lint/typecheck reporting.

Tradeoff:
- Better visibility of global debt, but increased CI time and signal noise.

### Option 3: Two-level hard gate

- Same as Option 1, plus mandatory boundary-focused test suites.

Tradeoff:
- Better confidence, but higher maintenance burden and test brittleness risk in stopgap phase.

## 5. Design

### 5.1 Quality Boundaries

Boundary domain stays focused on auth/scope/request files, including:

- API boundary (`src/api/api.ts`, `src/api/requestContext.ts`, `src/api/modules/auth.ts`)
- Session/scope stores (`src/app/session/**/*.ts`, `src/app/scope/*.ts`)
- Auth entry points and boundary adapters (`src/components/Auth/AuthContext.tsx`, `src/features/ai/api/chatApi.ts`, `src/features/ai/api/assistApi.ts`, `src/hooks/useNotificationWebSocket.ts`, `src/utils/tokenManager.ts`)

### 5.2 TypeScript Gate

- `web/tsconfig.auth-scope.json` remains the strict gate for boundary files.
- Strict flags for this boundary:
  - `strict: true`
  - `noUnusedLocals: true`
  - `noUnusedParameters: true`

### 5.3 ESLint Gate

- Maintain boundary-specific strict rules:
  - no direct localStorage access in protected boundary files
  - `@typescript-eslint/no-explicit-any`
  - `@typescript-eslint/no-unused-vars`
- Add/retain minimal baseline rules so governance is explicit and stable rather than empty-by-default at framework baseline.

### 5.4 CI Integration

Add a dedicated frontend governance job in `.github/workflows/ci.yml`:

1. Checkout repository
2. Setup Node runtime
3. Install dependencies under `web/`
4. Run `npm run lint:auth-scope`
5. Run `npm run typecheck:auth-scope`

Failure semantics:
- Any failed step fails the job and blocks merge.
- Existing backend `ai-contract-check` job remains unchanged and runs in parallel.

## 6. Data Flow and Error Semantics

1. PR push triggers CI.
2. Frontend governance job runs boundary lint and boundary strict typecheck.
3. If either command fails, CI returns actionable compiler/linter output without suppression or fallback.
4. Successful run means boundary contract quality is preserved for this PR.

## 7. Verification Plan

Local verification:

1. `cd web`
2. `npm run lint:auth-scope`
3. `npm run typecheck:auth-scope`

CI verification:

1. Open PR with a boundary touch.
2. Confirm new frontend governance job appears and blocks on failure.
3. Confirm backend `ai-contract-check` still executes independently.

## 8. Risks and Mitigations

1. Risk: Developers assume full-repo strict quality is enforced.
   - Mitigation: explicitly label this as boundary-only stopgap in docs and CI job naming.
2. Risk: Boundary file list drifts from real usage over time.
   - Mitigation: treat boundary list update as mandatory when auth/scope/request code paths expand.
3. Risk: Governance checks increase CI duration.
   - Mitigation: keep gate narrow and command set minimal.

## 9. Exit Criteria

This stopgap is considered complete when:

1. Boundary strict typecheck and lint are both merge-blocking in CI.
2. No global strict migration is introduced in this change.
3. No business behavior changes are introduced.
