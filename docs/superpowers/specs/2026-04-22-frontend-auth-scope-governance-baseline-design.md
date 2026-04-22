# OpsPilot Frontend Auth and Scope Governance Baseline Design

- Date: 2026-04-22
- Scope: frontend governance baseline for the remaining authentication, scope-state, refresh, and quality-gate findings from `docs/engineering-code-review-report-2026-04-21.md`
- Goal: hard-cut frontend auth to cookie-session semantics, unify `projectId/teamId` into one scope source, and add enforceable lint/type/test guardrails so old localStorage-based auth paths cannot return

## 1. Background

The remaining review findings in the frontend are not isolated bugs. They share one root cause: identity state, scope state, and request behavior are spread across pages, API modules, and ad hoc browser storage reads.

Current examples in the codebase:

1. `web/src/components/Auth/AuthContext.tsx` still persists `token`, `refreshToken`, `user`, and `permissions`.
2. `web/src/utils/tokenManager.ts` still depends on a locally stored JWT to determine refresh timing.
3. `web/src/api/api.ts` reads `projectId/teamId` from `localStorage` on every request.
4. `web/src/features/ai/api/chatApi.ts` and `web/src/features/ai/api/assistApi.ts` still hand-build `Authorization` and `X-Project-ID`.
5. Pages and feature modules such as deployment, services, monitoring, project switcher, and Kubernetes panels each read `projectId/teamId` directly.

This design intentionally narrows the next remediation stage to one coherent sub-project:

1. hard-cut cookie-session auth in the frontend,
2. unify project/team scope state behind a single store,
3. normalize refresh and request-context injection,
4. introduce minimal lint/type/test guardrails for the new boundary.

This design does not cover the other remaining review themes such as `ClusterDetailPage` decomposition, `NotificationContext` slimming, `CopilotSurface` decomposition, RBAC handler splitting, or AI tool panic removal.

## 2. Constraints and Non-Goals

### 2.1 Constraints

1. Backend remains stateless. No new server-side in-memory session state is allowed.
2. Authentication is a hard-cut. No compatibility bridge is allowed for `token` or `refreshToken` localStorage usage.
3. Existing React + Axios + fetch transport patterns may be reorganized, but the user-facing auth API contract remains `/auth/login`, `/auth/refresh`, `/auth/logout`, and `/auth/me`.
4. `projectId/teamId` may remain a frontend preference across page reloads, but only one module may read or write that persisted value.
5. This sub-project must be implementable without requiring an immediate full-repo `strict: true` conversion.

### 2.2 Non-Goals

1. No backward-compatible dual auth mode (`cookie + localStorage bearer`) is allowed.
2. No attempt to solve all remaining frontend architecture issues in this one change.
3. No redesign of backend session mechanics.
4. No elimination of all localStorage usage in the frontend. Non-auth preferences such as language, theme, UI toggles, and view mode remain allowed.

## 3. Target Outcomes

After rollout:

1. Frontend authentication state is derived only from `HttpOnly` cookies, `/auth/me`, and browser-memory state.
2. `token`, `refreshToken`, `user`, and `permissions` are no longer persisted browser auth sources.
3. `projectId/teamId` have exactly one frontend source of truth: a dedicated `ScopeStore`.
4. API modules, AI transports, and WebSocket clients no longer hand-build auth headers or read auth tokens directly.
5. Refresh behavior uses one centralized trigger and one in-flight refresh promise.
6. New or refactored boundary modules are protected by lint/type/test checks that prevent reintroducing localStorage auth logic.

## 4. Architecture Changes

### 4.1 Session boundary

Introduce a dedicated frontend session boundary, conceptually named `AuthSessionStore`.

Responsibilities:

1. bootstrap session state via `/auth/me`,
2. store `user`, `permissions`, `loading`, and `isAuthenticated` in browser memory only,
3. coordinate login, refresh, and logout state transitions,
4. emit session lifecycle events without exposing token strings as application state.

Rules:

1. `AuthContext` becomes a consumer/facade over the session store, not a persistence layer.
2. Login and register success are finalized by refetching `/auth/me`, not by persisting returned token fields.
3. Logout clears browser-memory session state even if the logout request fails.
4. Session expiry handling clears session memory state and redirects through the existing login flow.

### 4.2 Scope boundary

Introduce a dedicated frontend scope boundary, conceptually named `ScopeStore`.

Responsibilities:

1. own `projectId` and `teamId`,
2. expose read/update APIs for pages and transports,
3. persist non-sensitive scope preference across reloads,
4. broadcast scope changes to interested consumers.

Rules:

1. `ScopeStore` is the only module allowed to read/write persisted scope storage.
2. Pages, components, and API modules may subscribe to the scope store, but may not directly touch the old keys.
3. Scope persistence is allowed because it is a work-context preference, not an authentication credential.

Default resolution order:

1. explicit page or route input,
2. remembered scope preference from `ScopeStore`,
3. derived default from available project/team data when the page provides one.

### 4.3 Request-context injection

Normalize request context injection behind one place in `web/src/api/api.ts`, conceptually named `ApiContextInjector`.

Responsibilities:

1. inject `X-Project-ID` and `X-Team-ID` from `ScopeStore`,
2. centralize auth refresh retry behavior,
3. normalize API error extraction.

Rules:

1. No module may manually set bearer auth from browser storage.
2. No module may manually reimplement refresh retry logic.
3. `api.ts` remains the only Axios-based retry gate for auth refresh.

### 4.4 AI and WebSocket transport normalization

`chatApi.ts`, `assistApi.ts`, and notification websocket bootstrapping must align with the new boundary.

Rules:

1. AI transport uses browser cookies for auth and `ScopeStore` for project/team context.
2. Notification websocket uses cookie-backed identity only.
3. No transport may read `token`, `refreshToken`, or user identity from localStorage or query parameters.

### 4.5 Quality-gate boundary

This sub-project also establishes the minimum enforceable governance baseline.

Rules:

1. The session/scope boundary modules and the transport modules touched by this change must be strict-clean.
2. ESLint must reject direct auth localStorage usage in the boundary layer.
3. Regression tests must explicitly assert absence of token-storage dependencies.

This is boundary-first strictness, not a full-repo strictness flip.

## 5. Component-Level Design

### 5.1 Session components

1. `web/src/components/Auth/AuthContext.tsx`
   - stop restoring session from localStorage token/user blobs,
   - bootstrap from `/auth/me`,
   - delegate refresh/login/logout state transitions to the new session boundary,
   - keep redirect-after-login behavior.
2. `web/src/utils/tokenManager.ts`
   - stop parsing locally stored JWTs,
   - become an event/state coordinator for refresh lifecycle instead of a token persistence helper,
   - expose only session lifecycle events needed by the store and API boundary.
3. `web/src/api/modules/auth.ts`
   - stay as the transport wrapper for auth endpoints,
   - stop serving as an implicit token persistence boundary.

### 5.2 Scope components

1. `web/src/components/Project/ProjectSwitcher.tsx`
   - read and write through `ScopeStore` only.
2. Monitoring scope pages and selectors
   - stop reading project scope directly from `window.localStorage`.
3. Deployment, services, Kubernetes, and other project/team-aware screens
   - replace direct `localStorage` reads with store selectors or explicit props derived from the store.

### 5.3 Transport components

1. `web/src/api/api.ts`
   - inject scope headers from `ScopeStore`,
   - keep one `refreshPromise` gate,
   - publish generic session lifecycle events, not token payload as app state.
2. `web/src/features/ai/api/chatApi.ts`
   - remove manual `Authorization` header construction,
   - source project/team context from `ScopeStore`.
3. `web/src/features/ai/api/assistApi.ts`
   - same normalization as chat transport.
4. `web/src/hooks/useNotificationWebSocket.ts`
   - keep cookie-only websocket identity,
   - remain free of user/token query construction.

### 5.4 Files directly affected in this sub-project

At minimum, this design expects changes in:

1. `web/src/components/Auth/AuthContext.tsx`
2. `web/src/api/api.ts`
3. `web/src/utils/tokenManager.ts`
4. `web/src/features/ai/api/chatApi.ts`
5. `web/src/features/ai/api/assistApi.ts`
6. `web/src/components/Project/ProjectSwitcher.tsx`
7. Monitoring scope pages that currently read project ID directly
8. Any API module that still hand-builds `Authorization` from `localStorage` such as host/service-related modules

## 6. Data Flow

### 6.1 App bootstrap

1. App starts.
2. `AuthSessionStore` calls `/auth/me`.
3. On success, browser-memory `user/permissions/isAuthenticated` is populated.
4. On failure, browser-memory session is cleared and the app is treated as logged out.
5. Once session state is known, `ScopeStore` initializes project/team scope.

No auth state is restored from localStorage during bootstrap.

### 6.2 Login flow

1. Client posts credentials to `/auth/login`.
2. Backend sets secure cookies.
3. Frontend does not persist token fields from the response.
4. Frontend immediately calls `/auth/me`.
5. Session store writes browser-memory user and permission state from `/auth/me`.

### 6.3 Refresh flow

1. An API response or session event indicates refresh is required.
2. `api.ts` enters the single refresh gate (`refreshPromise`).
3. `POST /auth/refresh` runs once for all concurrent callers.
4. On success, frontend emits a session-refreshed event and refetches `/auth/me`.
5. On failure, frontend emits a session-expired event and clears browser-memory session state.

No caller may open a second refresh mechanism outside this path.

### 6.4 Logout flow

1. Client calls `/auth/logout`.
2. Browser-memory session state is cleared regardless of request result.
3. Redirect-after-login state is preserved the same way as today.
4. Scope preference may remain persisted, but any auth-derived session state must be gone.

### 6.5 Scope flow

1. User selects project or team through UI.
2. `ScopeStore` updates in-memory state and persists the preference.
3. API/header injection consumers observe the new value.
4. Subsequent requests automatically include scope headers through the centralized injector.

### 6.6 API and transport flow

1. Browser sends auth cookies automatically.
2. `api.ts` injects scope headers from `ScopeStore`.
3. AI transports and websocket clients do not manually attach bearer tokens.
4. Errors propagate through one normalized API error shape.

## 7. Quality Gates

### 7.1 TypeScript strictness

This change does not flip the whole repo to `strict: true` in one step.

Instead:

1. boundary modules introduced or heavily refactored by this sub-project must be strict-clean,
2. the implementation may use a dedicated strict config or targeted `tsc --noEmit` entry that checks the boundary slice,
3. any new boundary code must avoid `any` fallbacks that recreate the old ambiguity.

### 7.2 ESLint minimum rules

The minimum lint baseline for this sub-project must prevent boundary regressions.

At minimum, lint must catch:

1. direct auth-key localStorage usage in the auth/request boundary,
2. unused variables and parameters in newly refactored boundary modules,
3. hook misuse in React boundary code,
4. clearly avoidable `any` leaks in the new session/scope interfaces.

### 7.3 Test policy

The new boundary is not accepted without regression tests that prove the old path is gone.

Required regression coverage:

1. bootstrap authenticated state from `/auth/me` without token-storage dependency,
2. login/register/logout without persisting token keys,
3. refresh flow with one in-flight refresh gate,
4. AI transports do not build bearer auth from localStorage,
5. scope headers come from the scope store, not ad hoc storage reads,
6. websocket auth no longer depends on token/user query state.

## 8. Delivery Plan

### Phase A: New boundary primitives

1. Introduce `AuthSessionStore`.
2. Introduce `ScopeStore`.
3. Add targeted tests that fail while old localStorage auth logic still exists.

### Phase B: Replace auth/session mainline

1. Refactor `AuthContext`.
2. Refactor `tokenManager`.
3. Refactor `api.ts` refresh and scope injection behavior.

### Phase C: Replace high-risk bypasses

1. Refactor AI transport modules.
2. Refactor websocket client bootstrapping.
3. Remove manual bearer header construction in remaining API modules.

### Phase D: Replace page-level scope reads

1. Project switcher.
2. Monitoring scope pages.
3. Deployment/services/Kubernetes pages that still read old keys directly.

### Phase E: Enforce quality gates

1. Enable the minimum ESLint rules.
2. Enable targeted strict/type validation for the new boundary modules.
3. Delete or rewrite tests that still encode old token-storage assumptions.

## 9. Acceptance Criteria

1. `token`, `refreshToken`, `user`, and `permissions` are no longer persisted as browser auth sources.
2. Frontend authenticated state is restored only through `/auth/me`.
3. There is exactly one refresh entrypoint and exactly one in-flight refresh promise.
4. AI chat, form assist, websocket, and API modules no longer manually build bearer auth from browser storage.
5. `projectId/teamId` are only read or written through one scope boundary.
6. The new boundary modules pass lint, targeted type checks, and regression tests.
7. No new code path introduced by this change depends on the removed localStorage auth model.

## 10. Risks and Mitigations

1. Risk: login and refresh behavior diverge during the hard-cut.
   - Mitigation: make `/auth/me` the only post-login source of truth and test login/refresh/logout as one flow.
2. Risk: project/team context disappears on reload for pages that previously read raw localStorage keys.
   - Mitigation: preserve scope as a store-managed preference and migrate page readers in a defined phase.
3. Risk: existing tests hide old assumptions.
   - Mitigation: explicitly replace token-storage tests with cookie-session boundary assertions.
4. Risk: enabling lint/type gates across too much surface blocks delivery.
   - Mitigation: enforce strictness on the touched boundary slice first, then expand in later projects.

