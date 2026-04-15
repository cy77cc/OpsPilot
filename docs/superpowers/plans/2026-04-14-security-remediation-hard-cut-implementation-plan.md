# Security Remediation Hard-Cut Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close all findings in `docs/reviews/2026-04-14-full-architecture-security-review.md` in one hard-cut change without compatibility mode.

**Architecture:** This plan applies a single security baseline across backend and frontend: cookie-based auth, strict server-side identity, route-level authz for high-risk actions, encrypted credential writes, and frontend safe navigation. Work is sequenced by trust boundary: auth core, route/WS trust, host secrets/permissions, frontend behavior, then full regression verification.

**Tech Stack:** Go (Gin, Gorm, Redis, Casbin, gorilla/websocket), React + TypeScript + Axios + Vitest.

---

## Scope Check

The spec spans multiple subsystems (auth, websocket, host, frontend). Normally this can be split into multiple plans, but the requested delivery mode is one hard-cut change. This plan keeps one plan artifact while separating tasks by subsystem boundary to reduce merge and validation risk.

---

## File Structure Lock (before tasks)

**Backend auth/session**
- Modify: `internal/core/utils/jwt.go`
- Modify: `internal/core/middleware/jwt.go`
- Modify: `internal/modules/user/handler/auth.go`
- Modify: `internal/modules/user/logic/auth.go`
- Add/Modify tests: `internal/core/utils/jwt_test.go`, `internal/core/middleware/jwt_test.go`, `internal/modules/user/handler/auth_test.go`

**Backend route protection**
- Modify: `internal/modules/project/api/routes.go`
- Modify: `internal/modules/notification/api/routes.go`
- Add tests: `internal/modules/project/api/routes_test.go`, `internal/modules/notification/api/routes_test.go`

**Notification websocket trust**
- Modify: `internal/bootstrap/modules.go`
- Modify: `internal/websocket/handler.go`
- Modify: `internal/websocket/hub.go`
- Modify: `configs/config.yaml`, `internal/core/config/config.go` (origin allowlist)
- Add tests: `internal/websocket/handler_test.go`, `internal/websocket/hub_test.go`

**Host credential + high-risk authorization**
- Modify: `internal/modules/host/logic/probe.go`
- Modify: `internal/modules/host/logic/onboarding.go`
- Modify: `internal/modules/host/logic/host_service.go`
- Modify: `internal/modules/host/model/node.go`
- Modify: `internal/modules/host/handler/files_handler.go`
- Modify: `internal/modules/host/handler/terminal_session.go`
- Modify: `internal/modules/host/handler/credentials_handler.go`
- Add tests: `internal/modules/host/logic/probe_test.go`, `internal/modules/host/logic/host_service_test.go`, `internal/modules/host/handler/security_test.go`

**SSH transport security**
- Modify: `internal/client/ssh/ssh.go`
- Add helper: `internal/client/ssh/known_hosts.go`
- Add tests: `internal/client/ssh/ssh_test.go`

**Monitoring webhook authenticity**
- Modify: `internal/modules/monitoring/api/routes.go`
- Modify: `internal/modules/monitoring/handler/handler.go`
- Modify: `internal/core/config/config.go`
- Modify: `configs/config.yaml`
- Add tests: `internal/modules/monitoring/handler/webhook_auth_test.go`

**Frontend auth + websocket + safe navigation**
- Modify: `web/src/api/api.ts`
- Modify: `web/src/components/Auth/AuthContext.tsx`
- Modify: `web/src/hooks/useNotificationWebSocket.ts`
- Modify: `web/src/pages/Hosts/HostTerminalPage.tsx`
- Modify: `web/src/components/Notification/NotificationPanel.tsx`
- Modify: `web/src/contexts/NotificationContext.tsx`
- Add: `web/src/utils/safeNavigate.ts`
- Add tests: `web/src/utils/safeNavigate.test.ts`, `web/src/hooks/useNotificationWebSocket.test.tsx`, `web/src/__tests__/Notification/NotificationPanel.test.tsx`

**Frontend menu/route consistency + RBAC cleanup**
- Modify: `web/src/app/layout/navigation.config.tsx`
- Modify: `web/src/app/routes/platform.routes.tsx`
- Modify: `web/src/components/RBAC/Authorized.tsx`
- Add tests: `web/src/app/routes/platform.routes.test.tsx`, `web/src/components/RBAC/Authorized.test.tsx`

---

### Task 1: Fix JWT secret lifecycle and disable query-token auth

**Files:**
- Modify: `internal/core/utils/jwt.go`
- Modify: `internal/core/middleware/jwt.go`
- Test: `internal/core/utils/jwt_test.go`
- Test: `internal/core/middleware/jwt_test.go`

- [ ] **Step 1: Write failing JWT secret lifecycle tests**
```go
func TestGenToken_RejectsEmptySecret(t *testing.T) {
    config.CFG.JWT.Secret = ""
    _, err := GenToken(1, false)
    if err == nil {
        t.Fatalf("expected error when jwt secret is empty")
    }
}
```

- [ ] **Step 2: Run lifecycle tests and verify failure**
Run: `go test ./internal/core/utils -run TestGenToken_RejectsEmptySecret -v`  
Expected: FAIL because current code signs even when secret is empty.

- [ ] **Step 3: Implement runtime secret getter and validation**
```go
func currentJWTSecret() ([]byte, error) {
    s := strings.TrimSpace(config.CFG.JWT.Secret)
    if s == "" {
        return nil, errors.New("jwt secret is empty")
    }
    return []byte(s), nil
}
```

- [ ] **Step 4: Add failing middleware test for query token rejection**
```go
func TestJWTAuth_RejectsQueryToken(t *testing.T) {
    r := gin.New()
    r.GET("/x", JWTAuth(), func(c *gin.Context) { c.Status(http.StatusOK) })
    req := httptest.NewRequest(http.MethodGet, "/x?token=abc", nil)
    rec := httptest.NewRecorder()
    r.ServeHTTP(rec, req)
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d", rec.Code)
    }
}
```

- [ ] **Step 5: Update middleware to header-only bearer parsing**
```go
accessTokenH := strings.TrimSpace(c.Request.Header.Get("Authorization"))
if accessTokenH == "" {
    c.AbortWithStatusJSON(http.StatusUnauthorized, xcode.NewErrCode(xcode.Unauthorized))
    return
}
```

- [ ] **Step 6: Run auth-core tests**
Run: `go test ./internal/core/utils ./internal/core/middleware -v`  
Expected: PASS

- [ ] **Step 7: Commit**
```bash
git add internal/core/utils/jwt.go internal/core/middleware/jwt.go internal/core/utils/jwt_test.go internal/core/middleware/jwt_test.go
git commit -m "fix(auth): enforce runtime jwt secret and remove query-token auth path"
```

### Task 2: Move auth transport to secure cookies (backend)

**Files:**
- Modify: `internal/modules/user/handler/auth.go`
- Modify: `internal/modules/user/logic/auth.go`
- Test: `internal/modules/user/handler/auth_test.go`

- [ ] **Step 1: Write failing login cookie test**
```go
func TestLogin_SetsAuthCookies(t *testing.T) {
    // call /auth/login
    // assert Set-Cookie contains HttpOnly and SameSite
}
```

- [ ] **Step 2: Run cookie test and verify failure**
Run: `go test ./internal/modules/user/handler -run TestLogin_SetsAuthCookies -v`  
Expected: FAIL because cookies are not set today.

- [ ] **Step 3: Set access/refresh cookies in Login and Refresh handlers**
```go
c.SetCookie("opspilot_at", accessToken, maxAge, "/", "", true, true)
c.SetCookie("opspilot_rt", refreshToken, maxAge, "/", "", true, true)
```

- [ ] **Step 4: Clear cookies in Logout handler**
```go
c.SetCookie("opspilot_at", "", -1, "/", "", true, true)
c.SetCookie("opspilot_rt", "", -1, "/", "", true, true)
```

- [ ] **Step 5: Keep response body minimal and non-sensitive**
```go
httpx.OK(c, gin.H{"user": resp.User, "roles": resp.Roles, "permissions": resp.Permissions})
```

- [ ] **Step 6: Run auth handler tests**
Run: `go test ./internal/modules/user/handler -v`  
Expected: PASS

- [ ] **Step 7: Commit**
```bash
git add internal/modules/user/handler/auth.go internal/modules/user/logic/auth.go internal/modules/user/handler/auth_test.go
git commit -m "feat(auth): switch login-refresh-logout transport to secure HttpOnly cookies"
```

### Task 3: Enforce JWT on project and notification routes

**Files:**
- Modify: `internal/modules/project/api/routes.go`
- Modify: `internal/modules/notification/api/routes.go`
- Test: `internal/modules/project/api/routes_test.go`
- Test: `internal/modules/notification/api/routes_test.go`

- [ ] **Step 1: Write failing unauthenticated route tests**
```go
func TestProjectsRoute_Unauthenticated_Returns401(t *testing.T) {
    r := gin.New()
    RegisterProjectHandlers(r.Group("/api/v1"), svcCtxForTest(t))
    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
    r.ServeHTTP(rec, req)
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d", rec.Code)
    }
}
func TestNotificationsRoute_Unauthenticated_Returns401(t *testing.T) {
    r := gin.New()
    RegisterNotificationHandlers(r.Group("/api/v1"), svcCtxForTest(t))
    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
    r.ServeHTTP(rec, req)
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d", rec.Code)
    }
}
```

- [ ] **Step 2: Run route tests and verify failure**
Run: `go test ./internal/modules/project/api ./internal/modules/notification/api -v`  
Expected: FAIL because routes are not behind JWT today.

- [ ] **Step 3: Patch project route group with JWT middleware**
```go
projects := g.Group("/projects", middleware.JWTAuth())
```

- [ ] **Step 4: Patch notification route group with JWT middleware**
```go
notifications := r.Group("/notifications", middleware.JWTAuth())
```

- [ ] **Step 5: Run route tests**
Run: `go test ./internal/modules/project/api ./internal/modules/notification/api -v`  
Expected: PASS

- [ ] **Step 6: Commit**
```bash
git add internal/modules/project/api/routes.go internal/modules/notification/api/routes.go internal/modules/project/api/routes_test.go internal/modules/notification/api/routes_test.go
git commit -m "fix(routes): protect project and notification APIs with JWT auth"
```

### Task 4: Secure notification websocket identity and origin checks

**Files:**
- Modify: `internal/bootstrap/modules.go`
- Modify: `internal/websocket/handler.go`
- Modify: `internal/core/config/config.go`
- Modify: `configs/config.yaml`
- Test: `internal/websocket/handler_test.go`

- [ ] **Step 1: Write failing websocket impersonation test**
```go
func TestNotificationWS_RejectsUserIDQueryImpersonation(t *testing.T) {
    r := gin.New()
    r.GET("/ws/notifications", HandleWebSocket)
    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/ws/notifications?user_id=999", nil)
    r.ServeHTTP(rec, req)
    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401 when uid context missing, got %d", rec.Code)
    }
}
```

- [ ] **Step 2: Run websocket handler tests and verify failure**
Run: `go test ./internal/websocket -run TestNotificationWS_RejectsUserIDQueryImpersonation -v`  
Expected: FAIL because handler accepts `user_id` query today.

- [ ] **Step 3: Register websocket endpoint under auth middleware**
```go
ws := engine.Group("/ws", middleware.JWTAuth())
ws.GET("/notifications", websocket.HandleWebSocket)
```

- [ ] **Step 4: Remove `user_id` query trust, use context `uid` only**
```go
uid := httpx.UIDFromCtx(c)
if uid == 0 { c.JSON(http.StatusUnauthorized, ...); return }
```

- [ ] **Step 5: Add configurable websocket origin allowlist**
```go
CheckOrigin: func(r *http.Request) bool { return originAllowed(r.Header.Get("Origin")) }
```

- [ ] **Step 6: Run websocket tests**
Run: `go test ./internal/websocket -v`  
Expected: PASS

- [ ] **Step 7: Commit**
```bash
git add internal/bootstrap/modules.go internal/websocket/handler.go internal/core/config/config.go configs/config.yaml internal/websocket/handler_test.go
git commit -m "fix(ws): bind notification websocket identity to authenticated context and origin allowlist"
```

### Task 5: Fix websocket update message ID serialization bug

**Files:**
- Modify: `internal/websocket/hub.go`
- Test: `internal/websocket/hub_test.go`

- [ ] **Step 1: Write failing serialization test**
```go
func TestPushUpdate_UsesNumericStringID(t *testing.T) {
    // expect "123", not rune conversion
}
```

- [ ] **Step 2: Run serialization test and verify failure**
Run: `go test ./internal/websocket -run TestPushUpdate_UsesNumericStringID -v`  
Expected: FAIL with current `string(rune(...))` behavior.

- [ ] **Step 3: Replace rune conversion with numeric string conversion**
```go
msg := WSMessage{Type: "update", ID: strconv.FormatUint(uint64(notifID), 10)}
```

- [ ] **Step 4: Run websocket tests**
Run: `go test ./internal/websocket -v`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/websocket/hub.go internal/websocket/hub_test.go
git commit -m "fix(ws): serialize notification update ids as numeric strings"
```

### Task 6: Encrypt new SSH password writes and hide sensitive response fields

**Files:**
- Modify: `internal/modules/host/logic/probe.go`
- Modify: `internal/modules/host/logic/onboarding.go`
- Modify: `internal/modules/host/logic/host_service.go`
- Modify: `internal/modules/host/model/node.go`
- Test: `internal/modules/host/logic/probe_test.go`
- Test: `internal/modules/host/handler/host_query_test.go`

- [ ] **Step 1: Write failing encryption-on-write test**
```go
func TestProbe_PersistsEncryptedPassword(t *testing.T) {
    // assert db value != plaintext input
}
```

- [ ] **Step 2: Write failing response masking test**
```go
func TestHostList_DoesNotExposeSSHPassword(t *testing.T) {
    // seed node with ssh_password
    // call GET /api/v1/hosts
    // assert response body does not contain "ssh_password"
}
```

- [ ] **Step 3: Run host tests and verify failures**
Run: `go test ./internal/modules/host/... -run 'TestProbe_PersistsEncryptedPassword|TestHostList_DoesNotExposeSSHPassword' -v`  
Expected: FAIL

- [ ] **Step 4: Encrypt credentials in probe/create/update flows**
```go
cipher, err := utils.EncryptText(strings.TrimSpace(req.Password), config.CFG.Security.EncryptionKey)
if err != nil { return nil, err }
node.SSHPassword = cipher
```

- [ ] **Step 5: Hide SSHPassword in API JSON**
```go
SSHPassword string `gorm:"column:ssh_password;type:varchar(256)" json:"-"`
```

- [ ] **Step 6: Run host module tests**
Run: `go test ./internal/modules/host/... -v`  
Expected: PASS

- [ ] **Step 7: Commit**
```bash
git add internal/modules/host/logic/probe.go internal/modules/host/logic/onboarding.go internal/modules/host/logic/host_service.go internal/modules/host/model/node.go internal/modules/host/logic/probe_test.go internal/modules/host/handler/host_query_test.go
git commit -m "fix(host): encrypt new ssh password writes and mask sensitive fields in responses"
```

### Task 7: Enforce fine-grained authz on high-risk host handlers

**Files:**
- Modify: `internal/modules/host/handler/files_handler.go`
- Modify: `internal/modules/host/handler/terminal_session.go`
- Modify: `internal/modules/host/handler/credentials_handler.go`
- Test: `internal/modules/host/handler/security_test.go`

- [ ] **Step 1: Write failing permission-gate tests**
```go
func TestFilesHandler_RequiresPermission(t *testing.T) {
    // authenticated user without host:file:* permission
    // GET /api/v1/hosts/1/files
    // expect 403
}
func TestTerminalHandler_RequiresPermission(t *testing.T) {
    // authenticated user without host:terminal:* permission
    // POST /api/v1/hosts/1/terminal/sessions
    // expect 403
}
func TestCredentialsHandler_RequiresPermission(t *testing.T) {
    // authenticated user without host:credential:* permission
    // GET /api/v1/credentials/ssh_keys
    // expect 403
}
```

- [ ] **Step 2: Run permission tests and verify failure**
Run: `go test ./internal/modules/host/handler -run 'TestFilesHandler_RequiresPermission|TestTerminalHandler_RequiresPermission|TestCredentialsHandler_RequiresPermission' -v`  
Expected: FAIL

- [ ] **Step 3: Add `httpx.Authorize` checks at handler entry**
```go
if !httpx.Authorize(c, h.svcCtx.DB, "host:file:read", "host:file:*", "host:*") { return }
```

- [ ] **Step 4: Add equivalent checks for terminal and credential endpoints**
```go
if !httpx.Authorize(c, h.svcCtx.DB, "host:terminal:write", "host:terminal:*", "host:*") { return }
```

- [ ] **Step 5: Run host handler tests**
Run: `go test ./internal/modules/host/handler -v`  
Expected: PASS

- [ ] **Step 6: Commit**
```bash
git add internal/modules/host/handler/files_handler.go internal/modules/host/handler/terminal_session.go internal/modules/host/handler/credentials_handler.go internal/modules/host/handler/security_test.go
git commit -m "fix(host): enforce explicit authorization on terminal file and credential endpoints"
```

### Task 8: Make probe token consume atomic under concurrency

**Files:**
- Modify: `internal/modules/host/logic/host_service.go`
- Test: `internal/modules/host/logic/host_service_test.go`

- [ ] **Step 1: Write failing concurrent consume test**
```go
func TestConsumeProbe_ConcurrentOnlyOneSucceeds(t *testing.T) {
    // two goroutines consume same token
    // expect exactly one success
}
```

- [ ] **Step 2: Run race-sensitive logic test and verify failure**
Run: `go test ./internal/modules/host/logic -run TestConsumeProbe_ConcurrentOnlyOneSucceeds -v`  
Expected: FAIL intermittently or deterministic failure.

- [ ] **Step 3: Add atomic guard (transaction + rows affected check)**
```go
res := tx.Model(&model.HostProbeSession{}).
    Where("id = ? AND consumed_at IS NULL", probe.ID).
    Update("consumed_at", &now)
if res.RowsAffected != 1 { return errors.New("probe_not_found") }
```

- [ ] **Step 4: Run host logic tests**
Run: `go test ./internal/modules/host/logic -v`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/modules/host/logic/host_service.go internal/modules/host/logic/host_service_test.go
git commit -m "fix(host): make probe token consumption atomic and concurrency-safe"
```

### Task 9: Enforce SSH host key verification

**Files:**
- Modify: `internal/client/ssh/ssh.go`
- Create: `internal/client/ssh/known_hosts.go`
- Test: `internal/client/ssh/ssh_test.go`

- [ ] **Step 1: Write failing host-key verification test**
```go
func TestNewSSHClient_RejectsUnknownHostKey(t *testing.T) {
    _, err := NewSSHClient("root", "x", "127.0.0.1", 22, "", "")
    if err == nil {
        t.Fatalf("expected host key verification error")
    }
}
```

- [ ] **Step 2: Run ssh client tests and verify failure**
Run: `go test ./internal/client/ssh -run TestNewSSHClient_RejectsUnknownHostKey -v`  
Expected: FAIL because current code uses `ssh.InsecureIgnoreHostKey()`.

- [ ] **Step 3: Replace insecure callback with known_hosts verifier**
```go
hostKeyCb, err := BuildHostKeyCallback()
if err != nil { return nil, err }
sshConf := &ssh.ClientConfig{
    User: username,
    Auth: authMethods,
    HostKeyCallback: hostKeyCb,
    Timeout: 8 * time.Second,
}
```

- [ ] **Step 4: Implement known_hosts loader helper**
```go
func BuildHostKeyCallback() (ssh.HostKeyCallback, error) {
    khPath := os.Getenv("OPS_KNOWN_HOSTS_PATH")
    if strings.TrimSpace(khPath) == "" {
        khPath = filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
    }
    return knownhosts.New(khPath)
}
```

- [ ] **Step 5: Run ssh client tests**
Run: `go test ./internal/client/ssh -v`  
Expected: PASS

- [ ] **Step 6: Commit**
```bash
git add internal/client/ssh/ssh.go internal/client/ssh/known_hosts.go internal/client/ssh/ssh_test.go
git commit -m "fix(ssh): enforce host key verification with known_hosts callback"
```

### Task 10: Add webhook signature verification for `/alerts/receiver`

**Files:**
- Modify: `internal/modules/monitoring/api/routes.go`
- Modify: `internal/modules/monitoring/handler/handler.go`
- Modify: `internal/core/config/config.go`
- Modify: `configs/config.yaml`
- Test: `internal/modules/monitoring/handler/webhook_auth_test.go`

- [ ] **Step 1: Write failing webhook signature tests**
```go
func TestReceiveWebhook_RejectsMissingSignature(t *testing.T) {
    // no X-OpsPilot-Signature header -> 401
}
func TestReceiveWebhook_RejectsInvalidSignature(t *testing.T) {
    // wrong signature header -> 401
}
func TestReceiveWebhook_AcceptsValidSignature(t *testing.T) {
    // correct HMAC over request body -> 200
}
```

- [ ] **Step 2: Run monitoring handler tests and verify failure**
Run: `go test ./internal/modules/monitoring/handler -run 'TestReceiveWebhook_RejectsMissingSignature|TestReceiveWebhook_RejectsInvalidSignature|TestReceiveWebhook_AcceptsValidSignature' -v`  
Expected: FAIL because webhook currently accepts unsigned requests.

- [ ] **Step 3: Add webhook secret config**
```go
type Monitoring struct {
    WebhookSecret string `mapstructure:"webhook_secret"`
}
```

- [ ] **Step 4: Verify signature before JSON binding**
```go
sig := strings.TrimSpace(c.GetHeader("X-OpsPilot-Signature"))
if !verifyWebhookHMAC(body, sig, h.svcCtx.Config.Monitoring.WebhookSecret) {
    httpx.Fail(c, xcode.Unauthorized, "invalid webhook signature")
    return
}
```

- [ ] **Step 5: Run monitoring tests**
Run: `go test ./internal/modules/monitoring/handler -v`  
Expected: PASS

- [ ] **Step 6: Commit**
```bash
git add internal/modules/monitoring/api/routes.go internal/modules/monitoring/handler/handler.go internal/core/config/config.go configs/config.yaml internal/modules/monitoring/handler/webhook_auth_test.go
git commit -m "fix(monitoring): require HMAC signature verification for alert receiver webhook"
```

### Task 11: Frontend auth migration to cookie session and no token persistence

**Files:**
- Modify: `web/src/api/api.ts`
- Modify: `web/src/components/Auth/AuthContext.tsx`
- Test: `web/src/__tests__/auth/tokenRefresh.test.ts`
- Test: `web/src/components/Auth/AuthContext.test.tsx`

- [ ] **Step 1: Write failing frontend test asserting no localStorage token dependency**
```ts
it('does not require localStorage token for authenticated bootstrap', async () => {
  localStorage.removeItem('token');
  // expect AuthContext to rely on /auth/me response
});
```

- [ ] **Step 2: Run auth frontend tests and verify failure**
Run: `npm test -- web/src/components/Auth/AuthContext.test.tsx web/src/__tests__/auth/tokenRefresh.test.ts --runInBand`  
Expected: FAIL

- [ ] **Step 3: Update API client to rely on cookie transport**
```ts
this.instance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api/v1',
  timeout: 30000,
  withCredentials: true,
});
```

- [ ] **Step 4: Remove token localStorage read/write in AuthContext**
```ts
const [token] = useState<string | null>(null); // no persisted bearer
await refreshUser(); // truth from /auth/me
```

- [ ] **Step 5: Run frontend auth tests**
Run: `npm test -- web/src/components/Auth/AuthContext.test.tsx web/src/__tests__/auth/tokenRefresh.test.ts --runInBand`  
Expected: PASS

- [ ] **Step 6: Commit**
```bash
git add web/src/api/api.ts web/src/components/Auth/AuthContext.tsx web/src/components/Auth/AuthContext.test.tsx web/src/__tests__/auth/tokenRefresh.test.ts
git commit -m "refactor(web-auth): migrate to cookie session and remove token persistence"
```

### Task 12: Remove token query usage from websocket clients

**Files:**
- Modify: `web/src/hooks/useNotificationWebSocket.ts`
- Modify: `web/src/pages/Hosts/HostTerminalPage.tsx`
- Test: `web/src/hooks/useNotificationWebSocket.test.tsx`

- [ ] **Step 1: Write failing websocket url test without token query**
```ts
it('builds notification websocket URL without token or user_id query', () => {
  // expect /ws/notifications without sensitive query params
});
```

- [ ] **Step 2: Run websocket hook tests and verify failure**
Run: `npm test -- web/src/hooks/useNotificationWebSocket.test.tsx --runInBand`  
Expected: FAIL

- [ ] **Step 3: Remove query token/user_id assembly and sensitive logs**
```ts
wsUrl = `${wsProtocol}//${window.location.host}/ws/notifications`;
```

- [ ] **Step 4: Remove host terminal ws token appending**
```ts
return `${protocol}://${window.location.host}${wsPath}`;
```

- [ ] **Step 5: Run websocket related tests**
Run: `npm test -- web/src/hooks/useNotificationWebSocket.test.tsx web/src/pages/Hosts/HostTerminalPage.test.tsx --runInBand`  
Expected: PASS

- [ ] **Step 6: Commit**
```bash
git add web/src/hooks/useNotificationWebSocket.ts web/src/pages/Hosts/HostTerminalPage.tsx web/src/hooks/useNotificationWebSocket.test.tsx
git commit -m "fix(web-ws): remove token query params and sensitive websocket url logging"
```

### Task 13: Add safe URL navigation and sanitize auth error exposure

**Files:**
- Create: `web/src/utils/safeNavigate.ts`
- Modify: `web/src/components/Notification/NotificationPanel.tsx`
- Modify: `web/src/contexts/NotificationContext.tsx`
- Modify: `web/src/pages/Auth/LoginPage.tsx`
- Modify: `web/src/pages/Auth/RegisterPage.tsx`
- Test: `web/src/utils/safeNavigate.test.ts`
- Test: `web/src/__tests__/Notification/NotificationPanel.test.tsx`

- [ ] **Step 1: Write failing safeNavigate tests**
```ts
it('rejects javascript protocol urls', () => {
  expect(isSafeActionUrl('javascript:alert(1)')).toBe(false);
});
```

- [ ] **Step 2: Run safety tests and verify failure**
Run: `npm test -- web/src/utils/safeNavigate.test.ts web/src/__tests__/Notification/NotificationPanel.test.tsx --runInBand`  
Expected: FAIL

- [ ] **Step 3: Implement safe navigate utility**
```ts
export function isSafeActionUrl(raw: string): boolean {
  const v = (raw || '').trim();
  if (v.startsWith('/')) return true;
  const u = new URL(v, window.location.origin);
  if (!['http:', 'https:'].includes(u.protocol)) return false;
  return u.origin === window.location.origin;
}
```

- [ ] **Step 4: Replace direct `window.location.href` calls with safe helper**
```ts
const handled = safeNavigate(notification.notification.action_url || '');
if (!handled) message.warning('不安全或无效的跳转地址');
```

- [ ] **Step 5: Normalize auth error display text**
```ts
setError('登录失败，请检查账号或稍后重试');
```

- [ ] **Step 6: Run notification/auth ui tests**
Run: `npm test -- web/src/utils/safeNavigate.test.ts web/src/__tests__/Notification/NotificationPanel.test.tsx web/src/components/Auth/AuthContext.test.tsx --runInBand`  
Expected: PASS

- [ ] **Step 7: Commit**
```bash
git add web/src/utils/safeNavigate.ts web/src/components/Notification/NotificationPanel.tsx web/src/contexts/NotificationContext.tsx web/src/pages/Auth/LoginPage.tsx web/src/pages/Auth/RegisterPage.tsx web/src/utils/safeNavigate.test.ts web/src/__tests__/Notification/NotificationPanel.test.tsx
git commit -m "fix(web-security): enforce safe notification navigation and sanitize auth error messaging"
```

### Task 14: Align governance menu-route behavior and remove hard-coded permission helper

**Files:**
- Modify: `web/src/app/layout/navigation.config.tsx`
- Modify: `web/src/app/routes/platform.routes.tsx`
- Modify: `web/src/components/RBAC/Authorized.tsx`
- Test: `web/src/app/routes/platform.routes.test.tsx`
- Test: `web/src/components/RBAC/Authorized.test.tsx`

- [ ] **Step 1: Write failing menu-route consistency test**
```ts
it('does not expose legacy settings governance menu when route redirects are disabled', () => {});
```

- [ ] **Step 2: Run route/rbac tests and verify failure**
Run: `npm test -- web/src/app/routes/platform.routes.test.tsx web/src/components/RBAC/Authorized.test.tsx --runInBand`  
Expected: FAIL

- [ ] **Step 3: Align menu and route behavior under `governanceMenuEnabled`**
```ts
...(governanceMenuEnabled ? [] : [
  { key: '/settings/users', icon: <UserOutlined />, label: '用户管理' },
  { key: '/settings/roles', icon: <UserOutlined />, label: '角色管理' },
  { key: '/settings/permissions', icon: <UserOutlined />, label: '权限列表' },
])
```

- [ ] **Step 4: Remove unused hard-coded `checkPermission` export**
```ts
// remove `checkPermission` export and keep only component default export
export default Authorized;
```

- [ ] **Step 5: Run route/rbac tests**
Run: `npm test -- web/src/app/routes/platform.routes.test.tsx web/src/components/RBAC/Authorized.test.tsx --runInBand`  
Expected: PASS

- [ ] **Step 6: Commit**
```bash
git add web/src/app/layout/navigation.config.tsx web/src/app/routes/platform.routes.tsx web/src/components/RBAC/Authorized.tsx web/src/app/routes/platform.routes.test.tsx web/src/components/RBAC/Authorized.test.tsx
git commit -m "fix(web-consistency): align governance menu routes and remove hard-coded permission helper"
```

### Task 15: End-to-end verification and closeout

**Files:**
- Modify: `docs/reviews/2026-04-14-full-architecture-security-review.md` (mark resolved evidence)
- Modify: `README.md` (auth/session transport note)

- [x] **Step 1: Run backend full test pass**
Run: `go test ./...`  
Expected: PASS

- [x] **Step 2: Run frontend full test pass**
Run: `cd web && npm test`  
Expected: PASS

- [x] **Step 3: Run lint and build checks**
Run: `make build && make web-build`  
Expected: successful build artifacts.

- [x] **Step 4: Validate reviewed findings closure**
Run: manual checklist against `docs/reviews/2026-04-14-full-architecture-security-review.md`  
Expected: each R-001~R-017 mapped to code/test evidence.

- [x] **Step 5: Commit closeout updates**
```bash
git add docs/reviews/2026-04-14-full-architecture-security-review.md README.md
git commit -m "docs: record remediation evidence for full hard-cut security review"
```
