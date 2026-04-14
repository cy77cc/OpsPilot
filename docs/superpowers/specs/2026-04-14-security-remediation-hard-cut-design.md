# OpsPilot Security Remediation Hard-Cut Design

- Date: 2026-04-14
- Scope: full-stack hard-cut remediation for findings in `docs/reviews/2026-04-14-full-architecture-security-review.md`
- Goal: close all reviewed issues in one change, without compatibility mode

## 1. Background

The current codebase has cross-cutting security and correctness gaps across authentication, route protection, WebSocket identity binding, host credential handling, and frontend navigation/URL safety.

The user decision for this project is explicit:

1. Fix all reviewed issues in one large change.
2. No backward compatibility mode.
3. Move token/session strategy to HttpOnly cookie-based auth.
4. Do not migrate existing plaintext SSH passwords in database history.

This design describes a single cohesive remediation that makes the security baseline consistent across backend and frontend.

## 2. Constraints and Non-Goals

### 2.1 Constraints

1. One change, not multiple independent changes.
2. Breaking changes are allowed.
3. New/updated credential writes must be protected.
4. Existing project patterns (Gin + middleware + module routes + React API modules) should be preserved.

### 2.2 Non-goals

1. No migration of already persisted plaintext SSH passwords.
2. No temporary dual-mode auth (localStorage + cookie) bridge.
3. No broad unrelated refactor outside reviewed findings.

## 3. Target Outcomes

After rollout:

1. Authentication relies on secure cookie transport, not localStorage persistence.
2. HTTP/WS identity always comes from trusted server context, never user-controllable query identity.
3. High-risk host operations require both authentication and explicit authorization.
4. Sensitive credential fields are not exposed by API responses.
5. Newly written host credentials are encrypted at rest.
6. Concurrency and message-encoding correctness defects from review are fixed.
7. Navigation and URL handling in frontend are consistent and safe.

## 4. Architecture Changes

### 4.1 Auth and token lifecycle

1. Remove static JWT secret capture at package init.
2. Enforce runtime JWT secret validation (fail-fast on empty secret).
3. Shift login/refresh/logout flow to HttpOnly cookie transport.
4. Frontend auth state comes from `/auth/me` and in-memory state, not persisted bearer token.

### 4.2 Route and middleware normalization

1. Project routes (`/projects`) must be behind `JWTAuth`.
2. Notification routes (`/notifications`) must be behind `JWTAuth`.
3. Host high-risk endpoints (terminal/files/credentials) must enforce `Authorize(...)`.
4. JWT middleware should stop accepting `token` query parameter.

### 4.3 WebSocket trust boundary

1. Notification websocket endpoint must be protected by auth middleware.
2. Remove `user_id` and token query dependency from notification websocket.
3. Bind websocket user identity from trusted context only.
4. Restrict websocket origin checks to configured allowlist.

### 4.4 Host credential protection

1. Encrypt newly written SSH password values in probe/create/update flows.
2. Keep read path decrypt support where needed for SSH connection.
3. Ensure `Node` API serialization excludes `ssh_password`.

### 4.5 Frontend safety and consistency

1. Remove token localStorage dependency from API/websocket flows.
2. Add safe navigation guard for notification `action_url`.
3. Align settings/governance menu visibility with actual route accessibility.

## 5. Component-Level Design

### 5.1 Backend components

1. `internal/core/utils/jwt.go`
   - replace global secret variable with runtime getter + validation.
2. `internal/core/middleware/jwt.go`
   - accept only `Authorization: Bearer ...`, reject query token.
3. `internal/modules/*/api/routes.go`
   - patch `project` and `notification` groups to require auth.
4. `internal/bootstrap/modules.go` + `internal/websocket/handler.go`
   - secure websocket registration and identity extraction.
5. `internal/modules/host/handler/*`
   - enforce authorization on high-risk host handlers.
6. `internal/modules/host/logic/*`
   - encrypt writes for SSH passwords in probe/onboarding/update paths.
7. `internal/modules/host/model/node.go`
   - hide SSH password from JSON output.
8. `internal/modules/host/logic/host_service.go`
   - harden probe token consumption atomicity with rows-affected/locking checks.
9. `internal/websocket/hub.go`
   - fix update message ID serialization.

### 5.2 Frontend components

1. `web/src/api/api.ts` and auth module
   - remove localStorage bearer dependency and adapt to cookie auth.
2. `web/src/components/Auth/AuthContext.tsx`
   - use `/auth/me` bootstrap as source of truth without token persistence.
3. `web/src/hooks/useNotificationWebSocket.ts`
   - remove token/user_id query construction and sensitive URL logging.
4. `web/src/pages/Hosts/HostTerminalPage.tsx`
   - remove query token websocket construction.
5. `web/src/components/Notification/NotificationPanel.tsx`
   - replace direct `window.location.href` with validated navigation helper.
6. `web/src/contexts/NotificationContext.tsx`
   - apply same safe-navigation behavior for browser notification click.
7. `web/src/app/layout/navigation.config.tsx` + `web/src/app/routes/platform.routes.tsx`
   - make menu and route behavior consistent for governance toggle.

## 6. Data Flow

### 6.1 Login flow

1. Client posts credentials to `/auth/login`.
2. Server validates user and sets secure cookies.
3. Client initializes session with `/auth/me`.
4. Client stores user/profile/permissions in memory state only.

### 6.2 API call flow

1. Browser automatically sends auth cookie.
2. Backend middleware resolves user and injects `uid`.
3. Handler applies `Authorize` where required.
4. Response DTO excludes sensitive fields.

### 6.3 Notification websocket flow

1. Client opens websocket without identity query parameters.
2. Server authenticates from trusted request context.
3. Hub registers client under authenticated user ID.
4. Update messages include stringified numeric IDs.

### 6.4 Host credential write/read flow

1. Incoming password value is encrypted before persistence.
2. Stored ciphertext is never serialized to public API response.
3. Operational SSH path decrypts only in service/handler boundary where needed.

## 7. Error Handling and Security Policy

1. Missing/invalid JWT secret is startup-blocking.
2. Unauthorized access returns consistent `401`; forbidden returns `403`.
3. Websocket handshake failures are explicit and non-leaky.
4. Unsafe URLs (`javascript:`, `data:`, unapproved origins) are rejected client-side.
5. Authentication error messages shown in UI are generic to avoid sensitive detail leakage.

## 8. Testing Strategy

### 8.1 Security regression

1. Query token authentication is rejected.
2. Unauthenticated access to project/notification routes returns `401`.
3. Unauthorized access to host high-risk endpoints returns `403`.
4. Websocket cannot impersonate user via query `user_id`.
5. Node list/get payload does not include `ssh_password`.
6. New credential writes persist encrypted values.

### 8.2 Auth/session regression

1. Login sets secure cookies.
2. Refresh rotates session cookies correctly.
3. Logout clears cookies and invalidates refresh state.
4. Frontend bootstrap via `/auth/me` restores authenticated state.

### 8.3 Correctness regression

1. Concurrent probe token consume succeeds only once.
2. Notification websocket update ID parsing works with numeric string.
3. Governance menu and route behavior remain consistent under both feature-flag states.

## 9. Delivery Plan (single change, phased execution)

Within one change, execute in this order:

1. Phase A (P0 auth + route baseline):
   - JWT secret/runtime fix, query token removal, route auth patching, websocket trust fix.
2. Phase B (P0 host data + host authorization):
   - host credential encryption-on-write, response masking, high-risk authorize gates.
3. Phase C (P1/P2/P3 correctness + frontend):
   - probe race fix, hub ID fix, safe URL navigation, cookie-auth frontend transition, menu-route alignment, cleanup of hard-coded permission helper.

No compatibility fallback layer is added between phases.

## 10. Acceptance Criteria

1. All findings in `docs/reviews/2026-04-14-full-architecture-security-review.md` are resolved with code/test evidence.
2. Core auth/session flows pass backend and frontend tests.
3. High-risk host actions are blocked for unauthorized users.
4. Notification websocket enforces server-side identity only.
5. No new plaintext SSH password writes occur after deployment.

## 11. Risks and Mitigations

1. Risk: breaking login flow during cookie migration.
   - Mitigation: add end-to-end login/refresh/logout test coverage before merge.
2. Risk: permission gating may block previously accessible operations.
   - Mitigation: explicit route/permission matrix tests and clear operator-facing release note.
3. Risk: frontend stale assumptions about token storage.
   - Mitigation: remove storage dependencies centrally (`api.ts`, auth context, websocket hooks) and verify with integration tests.

