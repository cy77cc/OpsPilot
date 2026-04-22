# Frontend Auth and Scope Governance Baseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hard-cut frontend auth to cookie-session semantics, unify `projectId/teamId` behind one scope source, and add lint/type/test guardrails so localStorage-based auth paths cannot return.

**Architecture:** Implement two explicit frontend boundaries: `AuthSessionStore` for cookie-backed session state and `ScopeStore` for non-sensitive project/team preferences. Route all Axios/fetch/WebSocket context through one request-context helper, migrate bypass modules and pages to the new stores, then enforce the new boundary with targeted ESLint and strict typechecking rather than a whole-repo strict flip.

**Tech Stack:** React 19, TypeScript 5, Axios, fetch streaming, Ant Design, Vite, Vitest, ESLint 9, `typescript-eslint`.

---

## Scope Check

This plan only covers the first remaining review sub-project: frontend auth/session boundary, scope-state unification, refresh normalization, and quality gates. It does **not** include `ClusterDetailPage` decomposition, `NotificationContext` slimming, `CopilotSurface` decomposition, RBAC splitting, or AI tools panic removal.

## File Structure Lock (before tasks)

**Session boundary**
- Create: `web/src/app/session/sessionStore.ts`
- Create: `web/src/app/session/sessionStore.test.ts`
- Modify: `web/src/components/Auth/AuthContext.tsx`
- Modify: `web/src/components/Auth/AuthContext.test.tsx`
- Modify: `web/src/api/modules/auth.ts`

**Scope boundary**
- Create: `web/src/app/scope/scopeStore.ts`
- Create: `web/src/app/scope/useScope.ts`
- Create: `web/src/app/scope/scopeStore.test.ts`
- Modify: `web/src/components/Project/ProjectSwitcher.tsx`
- Create: `web/src/components/Project/ProjectSwitcher.test.tsx`

**Request context + refresh**
- Create: `web/src/api/requestContext.ts`
- Create: `web/src/api/requestContext.test.ts`
- Modify: `web/src/api/api.ts`
- Modify: `web/src/api/api.test.ts`
- Modify: `web/src/utils/tokenManager.ts`
- Modify: `web/src/utils/tokenManager.test.ts`
- Modify: `web/src/__tests__/auth/tokenRefresh.test.ts`

**Transport and API bypass removal**
- Modify: `web/src/features/ai/api/chatApi.ts`
- Modify: `web/src/features/ai/api/assistApi.ts`
- Create: `web/src/features/ai/api/chatApi.test.ts`
- Create: `web/src/features/ai/api/assistApi.test.ts`
- Modify: `web/src/hooks/useNotificationWebSocket.ts`
- Modify: `web/src/hooks/useNotificationWebSocket.test.tsx`
- Modify: `web/src/api/modules/hosts.ts`
- Create: `web/src/api/modules/hosts.test.ts`
- Modify: `web/src/api/modules/services.ts`
- Create: `web/src/api/modules/services.test.ts`

**Scope consumer migration**
- Modify: `web/src/pages/Monitor/RulesConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RulesConfigPage.test.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.test.tsx`
- Modify: `web/src/pages/Deployment/DeploymentPage.tsx`
- Modify: `web/src/pages/Deployment/DeploymentPage.test.tsx`
- Modify: `web/src/pages/Services/ServiceProvisionPage.tsx`
- Create: `web/src/pages/Services/ServiceProvisionPage.test.tsx`
- Modify: `web/src/components/K8s/NamespacePolicyPanel.tsx`
- Create: `web/src/components/K8s/NamespacePolicyPanel.test.tsx`

**Quality gates**
- Modify: `web/eslint.config.js`
- Modify: `web/package.json`
- Create: `web/tsconfig.auth-scope.json`

## Task 1: Introduce `ScopeStore` and remove direct scope persistence from the switcher

**Files:**
- Create: `web/src/app/scope/scopeStore.ts`
- Create: `web/src/app/scope/useScope.ts`
- Create: `web/src/app/scope/scopeStore.test.ts`
- Modify: `web/src/components/Project/ProjectSwitcher.tsx`
- Create: `web/src/components/Project/ProjectSwitcher.test.tsx`

- [ ] **Step 1: Write the failing scope-store test**

```ts
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { createScopeStore } from './scopeStore';

describe('createScopeStore', () => {
  beforeEach(() => localStorage.clear());

  it('hydrates persisted project/team scope and notifies subscribers on update', () => {
    localStorage.setItem('opspilot.scope', JSON.stringify({ projectId: '42', teamId: '7' }));

    const store = createScopeStore('opspilot.scope');
    const listener = vi.fn();
    const unsubscribe = store.subscribe(listener);

    expect(store.getSnapshot()).toEqual({ projectId: '42', teamId: '7' });

    store.setScope({ projectId: '84' });

    expect(store.getSnapshot()).toEqual({ projectId: '84', teamId: '7' });
    expect(JSON.parse(localStorage.getItem('opspilot.scope') || '{}')).toEqual({ projectId: '84', teamId: '7' });
    expect(listener).toHaveBeenCalledTimes(1);

    unsubscribe();
  });
});
```

- [ ] **Step 2: Run the scope-store test and verify it fails**

Run: `cd web && npm run test:run -- src/app/scope/scopeStore.test.ts`

Expected: FAIL because `scopeStore.ts` does not exist.

- [ ] **Step 3: Implement `ScopeStore` and `useScope`**

```ts
export type ScopeState = {
  projectId?: string;
  teamId?: string;
};

export function createScopeStore(storageKey = 'opspilot.scope') {
  let state: ScopeState = readPersistedScope(storageKey);
  const listeners = new Set<() => void>();

  return {
    getSnapshot: () => state,
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
    setScope(next: Partial<ScopeState>) {
      state = { ...state, ...next };
      writePersistedScope(storageKey, state);
      listeners.forEach((listener) => listener());
    },
    clearScope() {
      state = {};
      localStorage.removeItem(storageKey);
      listeners.forEach((listener) => listener());
    },
  };
}
```

- [ ] **Step 4: Refactor `ProjectSwitcher` to use the store**

```ts
const { projectId, setProjectId } = useScope();

onChange={(next) => {
  setProjectId(String(next));
  window.dispatchEvent(new CustomEvent('project:changed', { detail: { projectId: next } }));
}}
```

- [ ] **Step 5: Add the switcher integration test**

```ts
it('persists project selection through scopeStore instead of direct localStorage access', async () => {
  render(<ProjectSwitcher />);
  fireEvent.mouseDown(screen.getByRole('combobox'));
  fireEvent.click(await screen.findByText('Platform'));
  expect(JSON.parse(localStorage.getItem('opspilot.scope') || '{}')).toEqual({ projectId: 'platform-id' });
});
```

- [ ] **Step 6: Run the scope tests and verify they pass**

Run: `cd web && npm run test:run -- src/app/scope/scopeStore.test.ts src/components/Project/ProjectSwitcher.test.tsx`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add \
  web/src/app/scope/scopeStore.ts \
  web/src/app/scope/useScope.ts \
  web/src/app/scope/scopeStore.test.ts \
  web/src/components/Project/ProjectSwitcher.tsx \
  web/src/components/Project/ProjectSwitcher.test.tsx
git commit -m "feat(web): add shared scope store for project and team context"
```

## Task 2: Introduce `AuthSessionStore` and hard-cut `AuthContext` away from token persistence

**Files:**
- Create: `web/src/app/session/sessionStore.ts`
- Create: `web/src/app/session/sessionStore.test.ts`
- Modify: `web/src/components/Auth/AuthContext.tsx`
- Modify: `web/src/components/Auth/AuthContext.test.tsx`
- Modify: `web/src/api/modules/auth.ts`

- [ ] **Step 1: Write the failing session-store/AuthContext tests**

```ts
it('bootstraps authenticated state from /auth/me without touching token storage keys', async () => {
  const getItemSpy = vi.spyOn(Storage.prototype, 'getItem');
  render(<AuthProvider><AuthStatusDisplay /></AuthProvider>);
  await waitFor(() => expect(mockGetMe).toHaveBeenCalledTimes(1));
  expect(getItemSpy.mock.calls.filter(([key]) => key === 'token' || key === 'refreshToken')).toEqual([]);
});

it('login finalizes session by refetching /auth/me instead of storing returned tokens', async () => {
  mockLogin.mockResolvedValue({ data: {} });
  mockGetMe.mockResolvedValue({ data: { id: 1, username: 'alice', name: 'Alice', email: 'a@example.com', status: 'active', roles: ['user'], permissions: ['svc:view'] } });
  const { result } = renderHook(() => useAuth(), { wrapper: AuthProvider });
  await act(async () => { await result.current.login({ username: 'alice', password: 'secret' }); });
  expect(mockGetMe).toHaveBeenCalledTimes(2);
  expect(localStorage.getItem('token')).toBeNull();
  expect(localStorage.getItem('refreshToken')).toBeNull();
});
```

- [ ] **Step 2: Run the auth tests and verify they fail**

Run: `cd web && npm run test:run -- src/components/Auth/AuthContext.test.tsx`

Expected: FAIL because `AuthContext` still reads/writes `token`, `refreshToken`, `user`, and `permissions`.

- [ ] **Step 3: Refactor `authApi` to no-token handshake responses**

```ts
export const authApi = {
  async login(data: LoginParams): Promise<ApiResponse<void>> {
    await apiService.post('/auth/login', data);
    return { success: true, data: undefined };
  },
  async register(data: RegisterParams): Promise<ApiResponse<void>> {
    await apiService.post('/auth/register', data);
    return { success: true, data: undefined };
  },
  async getMe(): Promise<ApiResponse<AuthUser>> {
    const res = await apiService.get<any>('/auth/me');
    return {
      ...res,
      data: {
        id: Number(res.data?.id || 0),
        username: res.data?.username || '',
        name: res.data?.name || res.data?.username || '',
        email: res.data?.email || '',
        status: res.data?.status || 'active',
        roles: res.data?.roles || [],
        permissions: res.data?.permissions || [],
      },
    };
  },
  async logout(): Promise<ApiResponse<void>> {
    return apiService.post('/auth/logout', {});
  },
};
```

- [ ] **Step 4: Implement `AuthSessionStore` and refactor `AuthContext` to consume it**

```ts
export type SessionState = {
  user: AuthUser | null;
  permissions: string[];
  loading: boolean;
  isAuthenticated: boolean;
};

export const sessionStore = createSessionStore(authApi);

const login = async (payload: LoginParams) => {
  await sessionStore.login(payload);
};

const refreshUser = async () => {
  await sessionStore.bootstrap();
};
```

- [ ] **Step 5: Remove proactive JWT expiry parsing from `AuthContext`**

```ts
// delete parseJwtExpiresAt import and expiresAt timeout bookkeeping
window.addEventListener(TOKEN_EVENTS.REFRESHED, handleSessionRefreshed);
window.addEventListener(TOKEN_EVENTS.EXPIRED, handleSessionExpired);
```

- [ ] **Step 6: Run the session/auth tests and verify they pass**

Run: `cd web && npm run test:run -- src/app/session/sessionStore.test.ts src/components/Auth/AuthContext.test.tsx`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add \
  web/src/app/session/sessionStore.ts \
  web/src/app/session/sessionStore.test.ts \
  web/src/components/Auth/AuthContext.tsx \
  web/src/components/Auth/AuthContext.test.tsx \
  web/src/api/modules/auth.ts
git commit -m "refactor(web): hard-cut auth context to cookie session store"
```

## Task 3: Centralize request-context injection and convert refresh to response-driven session events

**Files:**
- Create: `web/src/api/requestContext.ts`
- Create: `web/src/api/requestContext.test.ts`
- Modify: `web/src/api/api.ts`
- Modify: `web/src/api/api.test.ts`
- Modify: `web/src/utils/tokenManager.ts`
- Modify: `web/src/utils/tokenManager.test.ts`
- Modify: `web/src/__tests__/auth/tokenRefresh.test.ts`

- [ ] **Step 1: Write the failing request-context and token-manager tests**

```ts
it('builds request headers from scopeStore and not from projectId/teamId localStorage keys', () => {
  scopeStore.setScope({ projectId: '42', teamId: '7' });
  expect(getRequestContextHeaders()).toEqual({
    'X-Project-ID': '42',
    'X-Team-ID': '7',
  });
});

it('dispatches refresh lifecycle events without token payload parsing', () => {
  const refreshed = vi.fn();
  window.addEventListener(TOKEN_EVENTS.REFRESHED, refreshed);
  dispatchTokenRefreshed();
  expect(refreshed).toHaveBeenCalledWith(expect.objectContaining({ detail: undefined }));
});
```

- [ ] **Step 2: Run the API/token-manager tests and verify they fail**

Run: `cd web && npm run test:run -- src/api/requestContext.test.ts src/api/api.test.ts src/utils/tokenManager.test.ts src/__tests__/auth/tokenRefresh.test.ts`

Expected: FAIL because `requestContext.ts` does not exist and `tokenManager.ts` still parses stored JWTs.

- [ ] **Step 3: Implement request-context helpers**

```ts
export function getRequestContextHeaders(): Record<string, string> {
  const { projectId, teamId } = scopeStore.getSnapshot();
  return {
    ...(projectId ? { 'X-Project-ID': projectId } : {}),
    ...(teamId ? { 'X-Team-ID': teamId } : {}),
  };
}

export function buildContextualFetchInit(init: RequestInit = {}): RequestInit {
  return {
    ...init,
    credentials: 'include',
    headers: {
      ...getRequestContextHeaders(),
      ...(init.headers || {}),
    },
  };
}
```

- [ ] **Step 4: Simplify `tokenManager.ts` to event-only semantics**

```ts
export const TOKEN_EVENTS = {
  REFRESHED: 'tokenRefreshed',
  EXPIRED: 'tokenExpired',
  NEEDS_REFRESH: 'tokenNeedsRefresh',
} as const;

export function dispatchTokenRefreshed(): void {
  window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.REFRESHED));
}

export function dispatchTokenExpired(): void {
  window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.EXPIRED));
}

export function dispatchTokenNeedsRefresh(source: 'response' | 'manual' = 'response'): void {
  window.dispatchEvent(new CustomEvent(TOKEN_EVENTS.NEEDS_REFRESH, { detail: { source } }));
}
```

- [ ] **Step 5: Refactor `api.ts` to use request-context headers and response-driven refresh**

```ts
this.instance.interceptors.request.use((config) => {
  Object.assign(config.headers, getRequestContextHeaders());
  return config;
});

if (isAuthBusinessCode(payload.code) && !requestURL.includes('/auth/refresh') && !originalConfig._retry) {
  dispatchTokenNeedsRefresh('response');
  originalConfig._retry = true;
  return this.tryRefreshAndRetry(originalConfig);
}
```

- [ ] **Step 6: Run the request-boundary tests**

Run: `cd web && npm run test:run -- src/api/requestContext.test.ts src/api/api.test.ts src/utils/tokenManager.test.ts src/__tests__/auth/tokenRefresh.test.ts src/components/Auth/AuthContext.test.tsx`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add \
  web/src/api/requestContext.ts \
  web/src/api/requestContext.test.ts \
  web/src/api/api.ts \
  web/src/api/api.test.ts \
  web/src/utils/tokenManager.ts \
  web/src/utils/tokenManager.test.ts \
  web/src/__tests__/auth/tokenRefresh.test.ts
git commit -m "refactor(web): centralize scope headers and response-driven refresh events"
```

## Task 4: Remove remaining auth bypasses from fetch transports and API modules

**Files:**
- Modify: `web/src/features/ai/api/chatApi.ts`
- Modify: `web/src/features/ai/api/assistApi.ts`
- Create: `web/src/features/ai/api/chatApi.test.ts`
- Create: `web/src/features/ai/api/assistApi.test.ts`
- Modify: `web/src/hooks/useNotificationWebSocket.ts`
- Modify: `web/src/hooks/useNotificationWebSocket.test.tsx`
- Modify: `web/src/api/modules/hosts.ts`
- Create: `web/src/api/modules/hosts.test.ts`
- Modify: `web/src/api/modules/services.ts`
- Create: `web/src/api/modules/services.test.ts`

- [ ] **Step 1: Write the failing transport and API bypass tests**

```ts
it('chatStream sends cookie-based request context without localStorage bearer auth', async () => {
  scopeStore.setScope({ projectId: '42' });
  await chatStream({ message: 'hello' }, handlers);
  expect(fetch).toHaveBeenCalledWith(
    expect.stringContaining('/ai/chat'),
    expect.objectContaining({
      credentials: 'include',
      headers: expect.objectContaining({ 'X-Project-ID': '42' }),
    }),
  );
});

it('downloadFile uses contextual fetch without Authorization header', async () => {
  scopeStore.setScope({ projectId: '42' });
  await hostsApi.downloadFile('7', '/tmp/app.yaml');
  expect(fetch).toHaveBeenCalledWith(
    expect.stringContaining('/hosts/7/files/download'),
    expect.objectContaining({
      credentials: 'include',
      headers: expect.not.objectContaining({ Authorization: expect.anything() }),
    }),
  );
});
```

- [ ] **Step 2: Run the transport/API tests and verify they fail**

Run: `cd web && npm run test:run -- src/features/ai/api/chatApi.test.ts src/features/ai/api/assistApi.test.ts src/hooks/useNotificationWebSocket.test.tsx src/api/modules/hosts.test.ts src/api/modules/services.test.ts`

Expected: FAIL because the new tests do not exist and the current modules still read localStorage directly.

- [ ] **Step 3: Refactor AI transports and host download to use contextual fetch**

```ts
const response = await fetch(`${base}/ai/chat`, buildContextualFetchInit({
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    ...(params.lastEventId ? { 'Last-Event-ID': String(params.lastEventId) } : {}),
  },
  body: JSON.stringify(payload),
  signal: controller.signal,
}));
```

```ts
const resp = await fetch(
  `${base}/hosts/${id}/files/download?path=${encodeURIComponent(filePath)}`,
  buildContextualFetchInit(),
);
```

- [ ] **Step 4: Remove service/host module localStorage fallbacks**

```ts
async create(data: ServiceCreateParams): Promise<ApiResponse<ServiceItem>> {
  const response = await apiService.post<any>('/services', data);
  return { ...response, data: mapService(response.data) };
}
```

- [ ] **Step 5: Keep websocket client cookie-only and scope-agnostic**

```ts
expect(MockWebSocket.instances[0].url).not.toContain('token=');
expect(MockWebSocket.instances[0].url).not.toContain('user_id=');
```

- [ ] **Step 6: Run the transport/API tests**

Run: `cd web && npm run test:run -- src/features/ai/api/chatApi.test.ts src/features/ai/api/assistApi.test.ts src/hooks/useNotificationWebSocket.test.tsx src/api/modules/hosts.test.ts src/api/modules/services.test.ts src/pages/Hosts/HostTerminalPage.test.tsx`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add \
  web/src/features/ai/api/chatApi.ts \
  web/src/features/ai/api/assistApi.ts \
  web/src/features/ai/api/chatApi.test.ts \
  web/src/features/ai/api/assistApi.test.ts \
  web/src/hooks/useNotificationWebSocket.ts \
  web/src/hooks/useNotificationWebSocket.test.tsx \
  web/src/api/modules/hosts.ts \
  web/src/api/modules/hosts.test.ts \
  web/src/api/modules/services.ts \
  web/src/api/modules/services.test.ts
git commit -m "refactor(web): remove localStorage auth bypasses from transports and APIs"
```

## Task 5: Migrate page-level scope consumers to `ScopeStore`

**Files:**
- Modify: `web/src/pages/Monitor/RulesConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RulesConfigPage.test.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.tsx`
- Modify: `web/src/pages/Monitor/ChannelsConfigPage.test.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.tsx`
- Modify: `web/src/pages/Monitor/RoutingConfigPage.test.tsx`
- Modify: `web/src/pages/Deployment/DeploymentPage.tsx`
- Modify: `web/src/pages/Deployment/DeploymentPage.test.tsx`
- Modify: `web/src/pages/Services/ServiceProvisionPage.tsx`
- Create: `web/src/pages/Services/ServiceProvisionPage.test.tsx`
- Modify: `web/src/components/K8s/NamespacePolicyPanel.tsx`
- Create: `web/src/components/K8s/NamespacePolicyPanel.test.tsx`

- [ ] **Step 1: Write the failing scope-consumer tests**

```ts
it('reads monitor project scope from ScopeStore instead of raw localStorage', async () => {
  scopeStore.setScope({ projectId: '42' });
  render(<RulesConfigPage />);
  await waitFor(() => {
    expect(mockApi.monitoring.getEffectiveRules).toHaveBeenCalledWith(expect.objectContaining({ projectId: '42' }));
  });
});

it('submits deployment target payload with team/project from ScopeStore', async () => {
  scopeStore.setScope({ projectId: '42', teamId: '7' });
  render(<DeploymentPage />);
  fireEvent.change(screen.getByLabelText('名称'), { target: { value: 'staging-target' } });
  fireEvent.mouseDown(screen.getByLabelText('目标类型'));
  fireEvent.click(await screen.findByText('k8s'));
  fireEvent.change(screen.getByLabelText('Cluster ID'), { target: { value: '11' } });
  fireEvent.click(screen.getByRole('button', { name: '创建部署目标' }));
  expect(mockApi.deployment.createTarget).toHaveBeenCalledWith(expect.objectContaining({ project_id: 42, team_id: 7 }));
});
```

- [ ] **Step 2: Run the page tests and verify failure**

Run: `cd web && npm run test:run -- src/pages/Monitor/RulesConfigPage.test.tsx src/pages/Monitor/ChannelsConfigPage.test.tsx src/pages/Monitor/RoutingConfigPage.test.tsx src/pages/Deployment/DeploymentPage.test.tsx src/pages/Services/ServiceProvisionPage.test.tsx src/components/K8s/NamespacePolicyPanel.test.tsx`

Expected: FAIL because these pages still read `window.localStorage` directly or the new tests do not exist.

- [ ] **Step 3: Replace monitor pages with `useScope`**

```ts
const { projectId, setProjectId } = useScope();
const currentProjectId = scope.scope === 'project' ? projectId : undefined;

useEffect(() => {
  setProjectId(normalizeProjectId(scope.projectId));
}, [scope.projectId, setProjectId]);
```

- [ ] **Step 4: Replace deployment/service/Kubernetes page reads**

```ts
const { projectId, teamId } = useScope();

project_id: Number(projectId || 0) || undefined,
team_id: Number(teamId || 0) || undefined,
```

- [ ] **Step 5: Add the missing page tests**

```ts
it('uses ScopeStore project selection when creating a service', async () => {
  scopeStore.setScope({ projectId: '88' });
  render(<ServiceProvisionPage />);
  fireEvent.change(screen.getByLabelText('服务名'), { target: { value: 'payments' } });
  fireEvent.change(screen.getByLabelText('负责人'), { target: { value: 'platform' } });
  fireEvent.change(screen.getByLabelText('镜像'), { target: { value: 'ghcr.io/example/payments:1.0.0' } });
  fireEvent.click(screen.getByRole('button', { name: '保存并创建' }));
  expect(mockApi.services.create).toHaveBeenCalledWith(expect.objectContaining({ project_id: 88 }));
});
```

- [ ] **Step 6: Run the page migration tests**

Run: `cd web && npm run test:run -- src/pages/Monitor/RulesConfigPage.test.tsx src/pages/Monitor/ChannelsConfigPage.test.tsx src/pages/Monitor/RoutingConfigPage.test.tsx src/pages/Deployment/DeploymentPage.test.tsx src/pages/Services/ServiceProvisionPage.test.tsx src/components/K8s/NamespacePolicyPanel.test.tsx`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add \
  web/src/pages/Monitor/RulesConfigPage.tsx \
  web/src/pages/Monitor/RulesConfigPage.test.tsx \
  web/src/pages/Monitor/ChannelsConfigPage.tsx \
  web/src/pages/Monitor/ChannelsConfigPage.test.tsx \
  web/src/pages/Monitor/RoutingConfigPage.tsx \
  web/src/pages/Monitor/RoutingConfigPage.test.tsx \
  web/src/pages/Deployment/DeploymentPage.tsx \
  web/src/pages/Deployment/DeploymentPage.test.tsx \
  web/src/pages/Services/ServiceProvisionPage.tsx \
  web/src/pages/Services/ServiceProvisionPage.test.tsx \
  web/src/components/K8s/NamespacePolicyPanel.tsx \
  web/src/components/K8s/NamespacePolicyPanel.test.tsx
git commit -m "refactor(web): move project and team consumers onto shared scope store"
```

## Task 6: Enforce the boundary with ESLint and targeted strict typechecking

**Files:**
- Modify: `web/eslint.config.js`
- Modify: `web/package.json`
- Create: `web/tsconfig.auth-scope.json`

- [ ] **Step 1: Write the failing governance checks**

```bash
cd web
npx eslint src/api/api.ts src/components/Auth/AuthContext.tsx src/features/ai/api/chatApi.ts src/features/ai/api/assistApi.ts src/api/modules/hosts.ts src/api/modules/services.ts
npm run typecheck:auth-scope
```

Expected: FAIL before the new config exists and before the touched boundary is strict-clean.

- [ ] **Step 2: Add targeted strict config**

```json
{
  "extends": "./tsconfig.app.json",
  "compilerOptions": {
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true
  },
  "include": [
    "src/api/api.ts",
    "src/api/requestContext.ts",
    "src/api/modules/auth.ts",
    "src/api/modules/hosts.ts",
    "src/api/modules/services.ts",
    "src/components/Auth/**/*",
    "src/app/session/**/*",
    "src/app/scope/**/*",
    "src/features/ai/api/**/*",
    "src/hooks/useNotificationWebSocket.ts"
  ]
}
```

- [ ] **Step 3: Add boundary ESLint rules**

```ts
{
  files: [
    'src/api/api.ts',
    'src/api/requestContext.ts',
    'src/api/modules/auth.ts',
    'src/api/modules/hosts.ts',
    'src/api/modules/services.ts',
    'src/components/Auth/**/*.{ts,tsx}',
    'src/app/session/**/*.{ts,tsx}',
    'src/features/ai/api/**/*.{ts,tsx}',
    'src/hooks/useNotificationWebSocket.ts',
  ],
  rules: {
    '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
    '@typescript-eslint/no-explicit-any': 'error',
    'no-restricted-properties': [
      'error',
      { object: 'localStorage', property: 'getItem', message: 'Use AuthSessionStore/ScopeStore instead of direct localStorage access in auth/request boundary files.' },
      { object: 'localStorage', property: 'setItem', message: 'Use AuthSessionStore/ScopeStore instead of direct localStorage access in auth/request boundary files.' },
      { object: 'localStorage', property: 'removeItem', message: 'Use AuthSessionStore/ScopeStore instead of direct localStorage access in auth/request boundary files.' }
    ]
  }
}
```

- [ ] **Step 4: Add the typecheck script**

```json
"scripts": {
  "typecheck:auth-scope": "tsc -p tsconfig.auth-scope.json --noEmit"
}
```

- [ ] **Step 5: Run the governance checks**

Run: `cd web && npx eslint src/api/api.ts src/api/requestContext.ts src/api/modules/auth.ts src/api/modules/hosts.ts src/api/modules/services.ts src/components/Auth/AuthContext.tsx src/features/ai/api/chatApi.ts src/features/ai/api/assistApi.ts src/hooks/useNotificationWebSocket.ts && npm run typecheck:auth-scope`

Expected: PASS

- [ ] **Step 6: Run the final frontend regression suite**

Run: `cd web && npm run test:run -- src/app/scope/scopeStore.test.ts src/app/session/sessionStore.test.ts src/api/requestContext.test.ts src/api/api.test.ts src/utils/tokenManager.test.ts src/components/Auth/AuthContext.test.tsx src/features/ai/api/chatApi.test.ts src/features/ai/api/assistApi.test.ts src/hooks/useNotificationWebSocket.test.tsx src/api/modules/hosts.test.ts src/api/modules/services.test.ts src/components/Project/ProjectSwitcher.test.tsx src/pages/Monitor/RulesConfigPage.test.tsx src/pages/Monitor/ChannelsConfigPage.test.tsx src/pages/Monitor/RoutingConfigPage.test.tsx src/pages/Deployment/DeploymentPage.test.tsx src/pages/Services/ServiceProvisionPage.test.tsx src/components/K8s/NamespacePolicyPanel.test.tsx`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add web/eslint.config.js web/package.json web/tsconfig.auth-scope.json
git commit -m "chore(web): enforce auth and scope governance boundary checks"
```

## Self-Review

### Spec coverage

This plan maps directly to the spec:

1. `AuthSessionStore` and `/auth/me` bootstrap are covered in Task 2.
2. `ScopeStore` and unified scope persistence are covered in Tasks 1 and 5.
3. Request-context injection and refresh normalization are covered in Task 3.
4. AI/WebSocket/API bypass cleanup is covered in Task 4.
5. Lint/type/test guardrails are covered in Task 6.

No spec section is left without a task.

### Placeholder scan

This plan contains no `TBD`, `TODO`, or “similar to above” placeholders. Every task lists concrete files, commands, and example code.

### Type consistency

The plan uses one consistent naming set:

1. `AuthSessionStore`
2. `ScopeStore`
3. `requestContext`
4. `TOKEN_EVENTS` retained as the frontend event bus name, but with session-only semantics

The plan intentionally removes browser JWT parsing instead of introducing a second refresh model.
