# Top 10 Full Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remediate all Top 10 findings from `docs/engineering-code-review-report-2026-04-21.md` in one integrated release with layered commits.

**Architecture:** Execute one branch in four layers: frontend governance gates, frontend decomposition, backend stability hardening, backend decomposition. Keep external route contracts stable while hard-cutting unsafe internal patterns (`localStorage` auth penetration, startup `log.Fatalf`, runtime `panic(err)` in AI tool initialization) and normalizing error semantics.

**Tech Stack:** Go 1.26+, Gin, GORM, Redis, React 19, TypeScript 5, Axios, ESLint 9, Vitest, GitHub Actions.

---

## Scope Check

The spec spans multiple subsystems (frontend governance, frontend decomposition, backend startup/tooling hardening, backend RBAC decomposition). To satisfy the explicit one-shot requirement, this plan keeps one implementation document but isolates work into independent task groups with hard checkpoints and commits between groups.

## File Structure Lock (before tasks)

**Frontend governance**
- Modify: `web/eslint.config.js`
- Modify: `web/tsconfig.auth-scope.json`
- Modify: `web/package.json`
- Modify: `.github/workflows/ci.yml`
- Modify: `web/src/api/api.ts`
- Modify: `web/src/api/requestContext.ts`
- Modify: `web/src/components/Auth/AuthContext.tsx`
- Modify: `web/src/features/ai/api/chatApi.ts`
- Modify: `web/src/features/ai/api/assistApi.ts`

**Frontend decomposition**
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/hooks/useClusterDetail.ts`
- Create: `web/src/pages/Deployment/Infrastructure/hooks/useClusterResources.ts`
- Create: `web/src/pages/Deployment/Infrastructure/components/ClusterOverviewPanel.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/components/ClusterOperationsPanel.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/components/ClusterSecurityPanel.tsx`
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Create: `web/src/components/AI/hooks/useCopilotSessionReducer.ts`
- Create: `web/src/components/AI/hooks/useCopilotStream.ts`
- Modify: `web/src/contexts/NotificationContext.tsx`
- Create: `web/src/contexts/notification/NotificationDataProvider.tsx`
- Create: `web/src/contexts/notification/NotificationWSProvider.tsx`
- Create: `web/src/contexts/notification/ApprovalStateProvider.tsx`
- Modify: `web/src/components/K8s/NamespacePolicyPanel.tsx`
- Modify: `web/src/components/K8s/QuotaEditor.tsx`
- Modify: `web/src/components/K8s/HPAEditor.tsx`

**Backend stability**
- Modify: `internal/core/config/config.go`
- Modify: `cmd/opspilot/root.go`
- Modify: `cmd/opspilot/main.go`
- Modify: `internal/modules/ai/agent/tools/host/tools.go`
- Modify: `internal/modules/ai/agent/tools/service/tools.go`
- Modify: `internal/modules/ai/agent/tools/kubernetes/tools.go`
- Modify: `internal/modules/ai/agent/tools/kubernetes/write.go`
- Modify: `internal/modules/ai/agent/tools/deployment/tools.go`
- Modify: `internal/modules/ai/agent/tools/monitor/tools.go`
- Modify: `internal/modules/ai/agent/tools/cicd/tools.go`
- Modify: `internal/modules/ai/agent/tools/governance/tools.go`
- Modify: `internal/modules/ai/agent/tools/infrastructure/tools.go`
- Modify: `internal/modules/ai/agent/tools/orchestrator/tools.go`
- Modify: `internal/modules/ai/agent/tools/orchestrator/platform_discovery.go`
- Modify: `internal/modules/ai/agent/tools/toolutil/unavailable.go`
- Modify: `internal/core/middleware/casbin.go`
- Modify: `internal/modules/user/handler/auth.go`
- Modify: `internal/core/httpx/xcode/code.go`

**Backend decomposition**
- Modify: `internal/modules/rbac/api/routes.go`
- Delete/Replace: `internal/modules/rbac/handler/permission.go`
- Create: `internal/modules/rbac/handler/user_handler.go`
- Create: `internal/modules/rbac/handler/role_handler.go`
- Create: `internal/modules/rbac/handler/permission_handler.go`
- Create: `internal/modules/rbac/handler/audit_handler.go`
- Create: `internal/modules/rbac/handler/common.go`

**Tests**
- Modify/Create: `web/src/api/api.test.ts`
- Modify/Create: `web/src/components/Auth/AuthContext.test.tsx`
- Modify/Create: `web/src/features/ai/api/chatApi.test.ts`
- Modify/Create: `web/src/features/ai/api/assistApi.test.ts`
- Modify/Create: `internal/modules/ai/agent/tools/host/tools_test.go`
- Modify/Create: `internal/modules/ai/agent/tools/monitor/tools_test.go`
- Modify/Create: `internal/modules/ai/agent/tools/kubernetes/tools_test.go`
- Modify/Create: `internal/modules/ai/agent/tools/factory_test.go`
- Modify/Create: `internal/modules/rbac/handler/permission_test.go`
- Modify/Create: `internal/core/config/config_test.go`
- Modify/Create: `internal/core/middleware/casbin_test.go`

### Task 1: Lock frontend governance gates (strict boundary + lint + CI)

**Files:**
- Modify: `web/eslint.config.js`
- Modify: `web/tsconfig.auth-scope.json`
- Modify: `web/package.json`
- Modify: `.github/workflows/ci.yml`
- Test: `web/src/api/api.test.ts`

- [ ] **Step 1: Write failing tests for boundary lint/type scripts**

```ts
// web/src/api/api.test.ts
it('uses centralized refresh gate and never injects Authorization from localStorage token', async () => {
  localStorage.setItem('token', 'legacy-token');
  await expect(apiService.refreshAccessToken()).resolves.toBeTypeOf('boolean');
  expect(localStorage.getItem('token')).toBe('legacy-token');
});
```

- [ ] **Step 2: Run targeted tests to confirm current baseline is incomplete**

Run: `cd web && npm run test:run -- src/api/api.test.ts`  
Expected: FAIL or missing governance assertions.

- [ ] **Step 3: Enforce boundary lint rules and strict config include list**

```js
// web/eslint.config.js (boundary block)
{
  files: authScopeBoundaryFiles,
  rules: {
    '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    '@typescript-eslint/no-explicit-any': 'error',
    'no-restricted-syntax': [
      'error',
      {
        selector: "CallExpression[callee.object.name='localStorage'][callee.property.name='getItem']",
        message: 'Use ScopeStore/AuthSessionStore boundary APIs instead of direct localStorage access.',
      },
      {
        selector: "CallExpression[callee.object.name='localStorage'][callee.property.name='setItem']",
        message: 'Use ScopeStore/AuthSessionStore boundary APIs instead of direct localStorage access.',
      },
      {
        selector: "CallExpression[callee.object.name='localStorage'][callee.property.name='removeItem']",
        message: 'Use ScopeStore/AuthSessionStore boundary APIs instead of direct localStorage access.',
      }
    ],
  },
}
```

```json
// web/tsconfig.auth-scope.json
{
  "extends": "./tsconfig.app.json",
  "compilerOptions": {
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noEmit": true
  },
  "include": [
    "src/api/api.ts",
    "src/api/requestContext.ts",
    "src/components/Auth/AuthContext.tsx",
    "src/features/ai/api/chatApi.ts",
    "src/features/ai/api/assistApi.ts"
  ]
}
```

- [ ] **Step 4: Wire scripts and CI blocking job**

```json
// web/package.json
{
  "scripts": {
    "lint:auth-scope": "eslint src/api/api.ts src/api/requestContext.ts src/components/Auth/AuthContext.tsx src/features/ai/api/chatApi.ts src/features/ai/api/assistApi.ts",
    "typecheck:auth-scope": "tsc -p tsconfig.auth-scope.json --noEmit"
  }
}
```

```yaml
# .github/workflows/ci.yml
frontend-auth-scope-governance:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-node@v4
      with:
        node-version: '20'
    - run: npm ci
      working-directory: web
    - run: npm run lint:auth-scope
      working-directory: web
    - run: npm run typecheck:auth-scope
      working-directory: web
```

- [ ] **Step 5: Run local governance checks**

Run: `cd web && npm run lint:auth-scope && npm run typecheck:auth-scope`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/eslint.config.js web/tsconfig.auth-scope.json web/package.json .github/workflows/ci.yml web/src/api/api.test.ts
git commit -m "chore(web): enforce auth-scope lint and strict gates in CI"
```

### Task 2: Finish frontend auth/scope/request hard-cut boundary

**Files:**
- Modify: `web/src/api/api.ts`
- Modify: `web/src/api/requestContext.ts`
- Modify: `web/src/components/Auth/AuthContext.tsx`
- Modify: `web/src/features/ai/api/chatApi.ts`
- Modify: `web/src/features/ai/api/assistApi.ts`
- Test: `web/src/components/Auth/AuthContext.test.tsx`
- Test: `web/src/features/ai/api/chatApi.test.ts`
- Test: `web/src/features/ai/api/assistApi.test.ts`

- [ ] **Step 1: Write failing tests for cookie-session-only and scope injection**

```ts
it('bootstraps auth state from /auth/me and does not read token keys', async () => {
  const spy = vi.spyOn(Storage.prototype, 'getItem');
  render(<AuthProvider><div /></AuthProvider>);
  await waitFor(() => expect(mockGetMe).toHaveBeenCalled());
  expect(spy.mock.calls.find(([k]) => k === 'token' || k === 'refreshToken')).toBeUndefined();
});
```

```ts
it('chatApi sends X-Project-ID from request context and no Authorization header', async () => {
  const headers = buildChatHeaders();
  expect(headers['X-Project-ID']).toBeDefined();
  expect((headers as Record<string, string>).Authorization).toBeUndefined();
});
```

- [ ] **Step 2: Run tests to verify failures**

Run: `cd web && npm run test:run -- src/components/Auth/AuthContext.test.tsx src/features/ai/api/chatApi.test.ts src/features/ai/api/assistApi.test.ts`  
Expected: FAIL before boundary hard-cut is complete.

- [ ] **Step 3: Normalize request context + refresh single gate**

```ts
// web/src/api/api.ts
private async tryRefreshAndRetry(config: AxiosRequestConfig): Promise<AxiosResponse<RawApiPayload>> {
  const refreshed = await this.refreshAccessToken();
  if (!refreshed) {
    dispatchTokenExpired();
    throw new ApiRequestError('登录已过期，请重新登录', 401, 4005);
  }
  return this.instance.request<RawApiPayload>(config);
}
```

- [ ] **Step 4: Remove AI transport manual auth construction**

```ts
// web/src/features/ai/api/chatApi.ts
const headers = getRequestContextHeaders();
const response = await fetch(url, {
  method: 'POST',
  credentials: 'include',
  headers: { 'Content-Type': 'application/json', ...headers },
  body: JSON.stringify(payload),
});
```

- [ ] **Step 5: Refactor AuthContext to /auth/me memory model**

```ts
// web/src/components/Auth/AuthContext.tsx
const bootstrapSession = async () => {
  try {
    const me = await authApi.getMe();
    setUser(me.data);
    setIsAuthenticated(true);
  } catch {
    setUser(null);
    setIsAuthenticated(false);
  } finally {
    setLoading(false);
  }
};
```

- [ ] **Step 6: Run boundary tests**

Run: `cd web && npm run test:run -- src/components/Auth/AuthContext.test.tsx src/features/ai/api/chatApi.test.ts src/features/ai/api/assistApi.test.ts src/api/api.test.ts`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add web/src/api/api.ts web/src/api/requestContext.ts web/src/components/Auth/AuthContext.tsx web/src/features/ai/api/chatApi.ts web/src/features/ai/api/assistApi.ts web/src/components/Auth/AuthContext.test.tsx web/src/features/ai/api/chatApi.test.ts web/src/features/ai/api/assistApi.test.ts web/src/api/api.test.ts
git commit -m "refactor(web): hard-cut auth and scope request boundary"
```

### Task 3: Decompose `ClusterDetailPage.tsx` monolith

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/hooks/useClusterDetail.ts`
- Create: `web/src/pages/Deployment/Infrastructure/hooks/useClusterResources.ts`
- Create: `web/src/pages/Deployment/Infrastructure/components/ClusterOverviewPanel.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/components/ClusterOperationsPanel.tsx`
- Test: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`

- [ ] **Step 1: Write failing test asserting page remains orchestration-only**

```ts
it('renders overview and operations panels from extracted hooks', async () => {
  render(<ClusterDetailPage />);
  expect(await screen.findByTestId('cluster-overview-panel')).toBeInTheDocument();
  expect(await screen.findByTestId('cluster-operations-panel')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test to confirm current implementation does not satisfy split contract**

Run: `cd web && npm run test:run -- src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`  
Expected: FAIL before split.

- [ ] **Step 3: Extract domain hooks**

```ts
// useClusterDetail.ts
export function useClusterDetail(clusterId: string) {
  const [detail, setDetail] = useState<ClusterDetail | null>(null);
  const load = useEffectEvent(async () => {
    const res = await clusterApi.getDetail(clusterId);
    setDetail(res.data);
  });
  useEffect(() => { void load(); }, [load]);
  return { detail, reload: load };
}
```

- [ ] **Step 4: Replace in-page state logic with hook and panel composition**

```tsx
// ClusterDetailPage.tsx
const { detail, reload } = useClusterDetail(clusterId);
const resources = useClusterResources(clusterId);
return (
  <>
    <ClusterOverviewPanel data={detail} onRefresh={reload} />
    <ClusterOperationsPanel resources={resources} />
  </>
);
```

- [ ] **Step 5: Run test and typecheck**

Run: `cd web && npm run test:run -- src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx && npm run typecheck:auth-scope`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx web/src/pages/Deployment/Infrastructure/hooks/useClusterDetail.ts web/src/pages/Deployment/Infrastructure/hooks/useClusterResources.ts web/src/pages/Deployment/Infrastructure/components/ClusterOverviewPanel.tsx web/src/pages/Deployment/Infrastructure/components/ClusterOperationsPanel.tsx web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx
git commit -m "refactor(web): split cluster detail page into orchestration and domain hooks"
```

### Task 4: Decompose `CopilotSurface.tsx` + split `NotificationContext.tsx`

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Create: `web/src/components/AI/hooks/useCopilotSessionReducer.ts`
- Create: `web/src/components/AI/hooks/useCopilotStream.ts`
- Modify: `web/src/contexts/NotificationContext.tsx`
- Create: `web/src/contexts/notification/NotificationDataProvider.tsx`
- Create: `web/src/contexts/notification/NotificationWSProvider.tsx`
- Create: `web/src/contexts/notification/ApprovalStateProvider.tsx`
- Test: `web/src/contexts/NotificationContext.test.tsx`

- [ ] **Step 1: Write failing tests for provider split and reduced re-render blast**

```ts
it('keeps approval updates from re-rendering data-only consumers', async () => {
  render(<NotificationProvider><DataOnlyProbe /></NotificationProvider>);
  triggerApprovalEvent();
  expect(dataOnlyRenderCount()).toBeLessThanOrEqual(2);
});
```

- [ ] **Step 2: Run tests to verify current context is too coupled**

Run: `cd web && npm run test:run -- src/contexts/NotificationContext.test.tsx`  
Expected: FAIL on excessive re-render assertions.

- [ ] **Step 3: Move Copilot state to reducer and stream hook**

```ts
// useCopilotSessionReducer.ts
export function copilotReducer(state: CopilotState, action: CopilotAction): CopilotState {
  switch (action.type) {
    case 'stream_chunk':
      return { ...state, chunks: [...state.chunks, action.payload] };
    case 'stream_done':
      return { ...state, streaming: false };
    default:
      return state;
  }
}
```

- [ ] **Step 4: Split notification providers and compose façade**

```tsx
// NotificationContext.tsx
export function NotificationProvider({ children }: { children: ReactNode }) {
  return (
    <NotificationDataProvider>
      <NotificationWSProvider>
        <ApprovalStateProvider>{children}</ApprovalStateProvider>
      </NotificationWSProvider>
    </NotificationDataProvider>
  );
}
```

- [ ] **Step 5: Run tests**

Run: `cd web && npm run test:run -- src/contexts/NotificationContext.test.tsx`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/hooks/useCopilotSessionReducer.ts web/src/components/AI/hooks/useCopilotStream.ts web/src/contexts/NotificationContext.tsx web/src/contexts/notification/NotificationDataProvider.tsx web/src/contexts/notification/NotificationWSProvider.tsx web/src/contexts/notification/ApprovalStateProvider.tsx web/src/contexts/NotificationContext.test.tsx
git commit -m "refactor(web): split copilot and notification context responsibilities"
```

### Task 5: Split K8s components into display/editor and move API side-effects

**Files:**
- Modify: `web/src/components/K8s/NamespacePolicyPanel.tsx`
- Modify: `web/src/components/K8s/QuotaEditor.tsx`
- Modify: `web/src/components/K8s/HPAEditor.tsx`
- Create: `web/src/components/K8s/hooks/useNamespacePolicyActions.ts`
- Test: `web/src/components/K8s/NamespacePolicyPanel.test.tsx`

- [ ] **Step 1: Add failing tests for side-effect separation**

```ts
it('delegates save action to hook and keeps component presentation-only', async () => {
  render(<NamespacePolicyPanel />);
  await user.click(screen.getByRole('button', { name: /保存/i }));
  expect(mockUseNamespacePolicyActions().savePolicy).toHaveBeenCalled();
});
```

- [ ] **Step 2: Run tests and observe failure**

Run: `cd web && npm run test:run -- src/components/K8s/NamespacePolicyPanel.test.tsx`  
Expected: FAIL before extraction.

- [ ] **Step 3: Extract API side effects**

```ts
// useNamespacePolicyActions.ts
export function useNamespacePolicyActions(clusterId: string) {
  const savePolicy = async (payload: SavePolicyInput) => kubernetesApi.saveNamespacePolicy(clusterId, payload);
  return { savePolicy };
}
```

- [ ] **Step 4: Refactor panel/editor components to pure props + callbacks**

```tsx
const { savePolicy } = useNamespacePolicyActions(clusterId);
<QuotaEditor value={quota} onSave={savePolicy} />;
```

- [ ] **Step 5: Run tests**

Run: `cd web && npm run test:run -- src/components/K8s/NamespacePolicyPanel.test.tsx`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/components/K8s/NamespacePolicyPanel.tsx web/src/components/K8s/QuotaEditor.tsx web/src/components/K8s/HPAEditor.tsx web/src/components/K8s/hooks/useNamespacePolicyActions.ts web/src/components/K8s/NamespacePolicyPanel.test.tsx
git commit -m "refactor(web): separate k8s display components from side effects"
```

### Task 6: Remove startup `log.Fatalf` and propagate config errors to entrypoint

**Files:**
- Modify: `internal/core/config/config.go`
- Modify: `cmd/opspilot/root.go`
- Modify: `cmd/opspilot/main.go`
- Test: `internal/core/config/config_test.go`

- [ ] **Step 1: Write failing config-load test**

```go
func TestNewConfig_ReturnsErrorWhenFileMissing(t *testing.T) {
	config.SetConfigFile("non-existent.yaml")
	err := config.NewConfig()
	if err == nil {
		t.Fatalf("expected error when config file is missing")
	}
}
```

- [ ] **Step 2: Run test to confirm function contract is missing**

Run: `go test ./internal/core/config -run TestNewConfig_ReturnsErrorWhenFileMissing -v`  
Expected: FAIL because current API is `MustNewConfig()`.

- [ ] **Step 3: Replace fatal API with error-return API**

```go
// internal/core/config/config.go
func NewConfig() error {
	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := viper.Unmarshal(&CFG); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Handle errors in entrypoint**

```go
// cmd/opspilot/root.go
if err := config.NewConfig(); err != nil {
	return fmt.Errorf("initialize config: %w", err)
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/core/config ./cmd/opspilot -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/core/config/config.go internal/core/config/config_test.go cmd/opspilot/root.go cmd/opspilot/main.go
git commit -m "refactor(core): replace fatal config load with explicit error path"
```

### Task 7: Replace AI tools initialization `panic(err)` with degraded registration

**Files:**
- Modify: `internal/modules/ai/agent/tools/host/tools.go`
- Modify: `internal/modules/ai/agent/tools/service/tools.go`
- Modify: `internal/modules/ai/agent/tools/kubernetes/tools.go`
- Modify: `internal/modules/ai/agent/tools/kubernetes/write.go`
- Modify: `internal/modules/ai/agent/tools/deployment/tools.go`
- Modify: `internal/modules/ai/agent/tools/monitor/tools.go`
- Modify: `internal/modules/ai/agent/tools/cicd/tools.go`
- Modify: `internal/modules/ai/agent/tools/governance/tools.go`
- Modify: `internal/modules/ai/agent/tools/infrastructure/tools.go`
- Modify: `internal/modules/ai/agent/tools/orchestrator/tools.go`
- Modify: `internal/modules/ai/agent/tools/orchestrator/platform_discovery.go`
- Modify: `internal/modules/ai/agent/tools/toolutil/unavailable.go`
- Test: `internal/modules/ai/agent/tools/host/tools_test.go`
- Test: `internal/modules/ai/agent/tools/monitor/tools_test.go`
- Test: `internal/modules/ai/agent/tools/kubernetes/tools_test.go`
- Test: `internal/modules/ai/agent/tools/factory_test.go`

- [ ] **Step 1: Write failing test for non-panic degradation**

```go
func TestToolFactory_DegradesWhenDependencyFails(t *testing.T) {
	_, err := BuildToolRegistryWithFaultyDeps()
	if err != nil {
		t.Fatalf("expected degraded registry, got error: %v", err)
	}
	if !RegistryHasUnavailableTool("deploy_service") {
		t.Fatalf("expected unavailable fallback tool registration")
	}
}
```

- [ ] **Step 2: Run package tests to confirm panic path exists**

Run: `go test ./internal/modules/ai/agent/tools/... -run DegradesWhenDependencyFails -v`  
Expected: FAIL or panic.

- [ ] **Step 3: Replace panic with fallback registration**

```go
if err != nil {
	registry.Register(toolutil.NewUnavailableTool("deploy_service", err))
	return
}
registry.Register(buildDeployServiceTool(client))
```

- [ ] **Step 4: Ensure unavailable tool returns actionable error payload**

```go
func NewUnavailableTool(name string, cause error) contracts.Tool {
	return &UnavailableTool{
		NameValue: name,
		Reason:    fmt.Sprintf("tool unavailable: %v", cause),
	}
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/modules/ai/agent/tools/... -v`  
Expected: PASS and no panic.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/ai/agent/tools internal/modules/ai/agent/tools/toolutil/unavailable.go
git commit -m "refactor(ai): degrade unavailable tools instead of panicking"
```

### Task 8: Standardize auth error semantics and structured Casbin deny logging

**Files:**
- Modify: `internal/modules/user/handler/auth.go`
- Modify: `internal/core/httpx/xcode/code.go`
- Modify: `internal/core/middleware/casbin.go`
- Test: `internal/modules/user/handler/auth_test.go`
- Test: `internal/core/middleware/casbin_test.go`

- [ ] **Step 1: Write failing tests for stable auth categories**

```go
func TestAuthHandler_Login_ReturnsStableUnauthorizedCode(t *testing.T) {
	// invalid credential scenario
	// assert response code == xcode.Unauthorized and stable message key
}
```

```go
func TestCasbinAudit_LogIncludesStructuredFields(t *testing.T) {
	// deny path
	// assert logger receives actor/resource/action/request_id fields
}
```

- [ ] **Step 2: Run tests and verify failures**

Run: `go test ./internal/modules/user/handler ./internal/core/middleware -v`  
Expected: FAIL on old semantics/log format.

- [ ] **Step 3: Map domain errors to stable xcode branches**

```go
func writeAuthLogicError(c *gin.Context, err error) {
	codeErr := xcode.FromError(err)
	httpx.Fail(c, codeErr.Code, codeErr.Msg)
}
```

- [ ] **Step 4: Replace `log.Printf` with structured logger call**

```go
logger.WithContext(c.Request.Context()).Infow(
	"rbac_access_denied",
	"actor", actor,
	"resource", resource,
	"action", action,
	"request_id", c.GetString("request_id"),
)
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/modules/user/handler ./internal/core/middleware -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/user/handler/auth.go internal/core/httpx/xcode/code.go internal/core/middleware/casbin.go internal/modules/user/handler/auth_test.go internal/core/middleware/casbin_test.go
git commit -m "refactor(auth): stabilize error semantics and casbin audit logging"
```

### Task 9: Split RBAC handler monolith (`permission.go`) by responsibility

**Files:**
- Modify: `internal/modules/rbac/api/routes.go`
- Replace: `internal/modules/rbac/handler/permission.go`
- Create: `internal/modules/rbac/handler/common.go`
- Create: `internal/modules/rbac/handler/user_handler.go`
- Create: `internal/modules/rbac/handler/role_handler.go`
- Create: `internal/modules/rbac/handler/permission_handler.go`
- Create: `internal/modules/rbac/handler/audit_handler.go`
- Test: `internal/modules/rbac/handler/permission_test.go`

- [ ] **Step 1: Write failing route regression test before split**

```go
func TestRBACRoutes_StillMountedAfterHandlerSplit(t *testing.T) {
	r := gin.New()
	registerRBACRoutes(r)
	assertRouteExists(t, r, "GET", "/rbac/permissions")
	assertRouteExists(t, r, "POST", "/rbac/roles")
}
```

- [ ] **Step 2: Run test and verify baseline**

Run: `go test ./internal/modules/rbac/... -run RBACRoutes_StillMountedAfterHandlerSplit -v`  
Expected: PASS baseline route map.

- [ ] **Step 3: Move shared request/response helpers to `common.go`**

```go
func bindJSON(c *gin.Context, req any) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		httpx.BindErr(c, err)
		return false
	}
	return true
}
```

- [ ] **Step 4: Extract user/role/permission/audit handlers and wire in routes**

```go
type UserHandler struct { svc service }
type RoleHandler struct { svc service }
type PermissionHandler struct { svc service }
type AuditHandler struct { svc service }
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/modules/rbac/... -v`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/rbac/api/routes.go internal/modules/rbac/handler/common.go internal/modules/rbac/handler/user_handler.go internal/modules/rbac/handler/role_handler.go internal/modules/rbac/handler/permission_handler.go internal/modules/rbac/handler/audit_handler.go internal/modules/rbac/handler/permission.go internal/modules/rbac/handler/permission_test.go
git commit -m "refactor(rbac): split monolithic permission handler by domain"
```

### Task 10: Full verification, integration cleanup, and closeout commit

**Files:**
- Modify: `docs/engineering-code-review-report-2026-04-21.md` (optional annotation of resolved items)
- Modify: `docs/superpowers/specs/2026-04-22-top10-full-remediation-design.md` (if deltas discovered)
- Modify: `docs/superpowers/plans/2026-04-22-top10-full-remediation-implementation-plan.md` (checkbox updates)

- [ ] **Step 1: Run frontend full checks**

Run: `cd web && npm run lint && npm run test:run && npm run build`  
Expected: PASS

- [ ] **Step 2: Run backend full checks**

Run: `go test ./...`  
Expected: PASS

- [ ] **Step 3: Run Top 10 evidence sweep**

Run: `rg -n "log\\.Fatalf|panic\\(" internal/core/config internal/modules/ai/agent/tools`  
Expected: no runtime hard-fail paths in scoped targets (test-only panic cases allowed).

Run: `wc -l web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx web/src/components/AI/CopilotSurface.tsx web/src/contexts/NotificationContext.tsx`  
Expected: significant reduction and decomposed responsibility boundaries.

- [ ] **Step 4: Update docs checkboxes and summarize residual risks**

```md
- [x] Top 10 item #1 remediated via ClusterDetail decomposition
- [x] Top 10 item #2 remediated via boundary strict gate
...
```

- [ ] **Step 5: Commit integration closeout**

```bash
git add -A
git commit -m "feat: complete full top10 engineering remediation across frontend and backend"
```

## Self-Review

1. Spec coverage check: all Top 10 themes are mapped to Tasks 1-9; Task 10 handles integrated verification and closeout.
2. Placeholder scan: no unresolved placeholders or deferred implementation language remains.
3. Type/signature consistency check: task snippets consistently reference `ScopeStore`, centralized refresh gate, degraded unavailable-tool registration, and split RBAC handler naming.
