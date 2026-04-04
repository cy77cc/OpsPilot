# Loading Skeleton Unification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep the app shell visible during route transitions and standardize frontend content-loading states on shared skeleton components instead of full-page fallbacks and scattered spinners.

**Architecture:** Split the current route suspense boundary so lazy route loading only affects the content region inside `AppLayout`, then expand the existing loading-skeleton module into semantic page/list/detail/form primitives. Migrate high-visibility pages and permission guards from `Spin`, `Card loading`, and `Table loading` to those primitives while preserving `Button loading` for action feedback and keeping refresh states lighter than first-load states.

**Tech Stack:** React 19, react-router-dom 6, Ant Design 6, Tailwind utility classes, Vitest, Testing Library

---

### Task 1: Route-Level Content Skeleton Boundary

**Files:**
- Modify: `web/src/ProtectedApp.tsx`
- Modify: `web/src/components/Layout/AppLayout.tsx`
- Create: `web/src/components/LoadingSkeleton/PageSkeleton.tsx`
- Modify: `web/src/components/LoadingSkeleton/index.ts`
- Test: `web/src/ProtectedApp.loading.test.tsx`

- [ ] **Step 1: Write the failing route-boundary test**

```tsx
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { render, screen } from '@testing-library/react';

vi.mock('./components/Layout/AppLayout', () => ({
  default: ({ children }: { children: React.ReactNode }) => (
    <div>
      <div data-testid="app-shell">shell</div>
      <main>{children}</main>
    </div>
  ),
}));

vi.mock('./pages/Dashboard/Dashboard', async () => {
  await new Promise((resolve) => setTimeout(resolve, 50));
  return { default: () => <div>dashboard-content</div> };
});

describe('ProtectedApp loading boundary', () => {
  it('keeps the app shell mounted while route content is suspended', async () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <ProtectedApp />
      </MemoryRouter>
    );

    expect(screen.getByTestId('app-shell')).toBeInTheDocument();
    expect(screen.getByTestId('page-skeleton')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the test to confirm the current app still blanks to the top-level fallback**

Run: `cd web && npm run test:run -- src/ProtectedApp.loading.test.tsx`

Expected: FAIL because `app-shell` is absent while the top-level `Suspense fallback` is rendered.

- [ ] **Step 3: Introduce a reusable page-level route fallback**

```tsx
import React from 'react';
import { Skeleton } from 'antd';

const PageSkeleton: React.FC = () => (
  <div
    data-testid="page-skeleton"
    aria-busy="true"
    className="space-y-6"
  >
    <Skeleton.Input active block className="!h-8 !w-56" />
    <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
      <Skeleton active paragraph={{ rows: 4 }} />
      <Skeleton active paragraph={{ rows: 4 }} />
      <Skeleton active paragraph={{ rows: 4 }} />
    </div>
    <Skeleton active paragraph={{ rows: 10 }} />
  </div>
);

export default PageSkeleton;
```

- [ ] **Step 4: Move suspense fallback ownership into the layout content region**

```tsx
const RouteFallback = <PageSkeleton />;

return (
  <PermissionProvider>
    <NotificationProvider userId={user?.id}>
      <Suspense fallback={<div className="min-h-screen bg-white" />}>
        <AppLayout>
          <PageTransition>
            <Suspense fallback={RouteFallback}>
              <Routes>{/* existing Route tree */}</Routes>
            </Suspense>
          </PageTransition>
        </AppLayout>
      </Suspense>
    </NotificationProvider>
  </PermissionProvider>
);
```

- [ ] **Step 5: Ensure `AppLayout` content area can host local fallback content cleanly**

```tsx
<Content className="min-h-[calc(100vh-64px)] bg-gray-50">
  <div className="mx-auto w-full max-w-[1600px] p-4 md:p-6">
    {children}
  </div>
</Content>
```

- [ ] **Step 6: Re-export the new page skeleton**

```ts
export { default } from './LoadingSkeleton';
export { default as PageSkeleton } from './PageSkeleton';
```

- [ ] **Step 7: Run the targeted test until it passes**

Run: `cd web && npm run test:run -- src/ProtectedApp.loading.test.tsx`

Expected: PASS with the shell visible and `page-skeleton` rendered during suspense.

- [ ] **Step 8: Commit**

```bash
git add web/src/ProtectedApp.tsx \
  web/src/components/Layout/AppLayout.tsx \
  web/src/components/LoadingSkeleton/PageSkeleton.tsx \
  web/src/components/LoadingSkeleton/index.ts \
  web/src/ProtectedApp.loading.test.tsx
git commit -m "feat: keep app shell visible during route loading"
```

### Task 2: Shared Semantic Skeleton Kit

**Files:**
- Modify: `web/src/components/LoadingSkeleton/LoadingSkeleton.tsx`
- Modify: `web/src/components/LoadingSkeleton/LoadingSkeleton.css`
- Modify: `web/src/components/LoadingSkeleton/index.ts`
- Create: `web/src/components/LoadingSkeleton/TableSkeleton.tsx`
- Create: `web/src/components/LoadingSkeleton/DetailSkeleton.tsx`
- Create: `web/src/components/LoadingSkeleton/FormSkeleton.tsx`
- Create: `web/src/components/LoadingSkeleton/CardGridSkeleton.tsx`
- Create: `web/src/components/LoadingSkeleton/LoadingSkeleton.test.tsx`

- [ ] **Step 1: Write failing component tests for semantic skeleton exports**

```tsx
import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  PageSkeleton,
  TableSkeleton,
  DetailSkeleton,
  FormSkeleton,
  CardGridSkeleton,
} from './index';

describe('loading skeleton kit', () => {
  it('renders a table skeleton with toolbar and rows', () => {
    render(<TableSkeleton toolbar rows={6} columns={5} />);
    expect(screen.getByTestId('table-skeleton')).toBeInTheDocument();
    expect(screen.getAllByTestId('table-skeleton-row')).toHaveLength(6);
  });

  it('renders a detail skeleton with summary cards', () => {
    render(<DetailSkeleton summaryCards={3} sections={2} />);
    expect(screen.getAllByTestId('detail-skeleton-card')).toHaveLength(3);
  });
});
```

- [ ] **Step 2: Run the test to capture missing component failures**

Run: `cd web && npm run test:run -- src/components/LoadingSkeleton/LoadingSkeleton.test.tsx`

Expected: FAIL because semantic components and test ids are not implemented yet.

- [ ] **Step 3: Keep `LoadingSkeleton.tsx` as the base adapter and add explicit variants**

```tsx
export type LoadingSkeletonType =
  | 'card'
  | 'list'
  | 'table'
  | 'detail';

export interface LoadingSkeletonProps {
  type?: LoadingSkeletonType;
  count?: number;
  'data-testid'?: string;
}
```

- [ ] **Step 4: Add semantic wrappers with stable, small APIs**

```tsx
const TableSkeleton: React.FC<{ toolbar?: boolean; rows?: number; columns?: number }> = ({
  toolbar = true,
  rows = 8,
}) => (
  <section data-testid="table-skeleton" aria-busy="true" className="space-y-4">
    {toolbar ? <Skeleton.Input active block className="!h-10 !w-72" /> : null}
    <div className="rounded-xl border border-gray-200 bg-white p-4">
      {Array.from({ length: rows }).map((_, index) => (
        <div key={index} data-testid="table-skeleton-row" className="py-3">
          <Skeleton active title={false} paragraph={{ rows: 1, width: ['100%'] }} />
        </div>
      ))}
    </div>
  </section>
);
```

- [ ] **Step 5: Add detail, form, and card-grid skeletons with shared accessibility wrapper**

```tsx
const DetailSkeleton: React.FC<{ summaryCards?: number; sections?: number }> = ({
  summaryCards = 3,
  sections = 3,
}) => (
  <section data-testid="detail-skeleton" aria-busy="true" className="space-y-6">
    <Skeleton.Input active block className="!h-8 !w-72" />
    <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
      {Array.from({ length: summaryCards }).map((_, index) => (
        <div key={index} data-testid="detail-skeleton-card" className="rounded-xl border border-gray-200 bg-white p-5">
          <Skeleton active paragraph={{ rows: 2 }} />
        </div>
      ))}
    </div>
    {Array.from({ length: sections }).map((_, index) => (
      <div key={index} className="rounded-xl border border-gray-200 bg-white p-5">
        <Skeleton active paragraph={{ rows: 5 }} />
      </div>
    ))}
  </section>
);
```

- [ ] **Step 6: Export the semantic kit from a single barrel**

```ts
export { default as LoadingSkeleton } from './LoadingSkeleton';
export { default as PageSkeleton } from './PageSkeleton';
export { default as TableSkeleton } from './TableSkeleton';
export { default as DetailSkeleton } from './DetailSkeleton';
export { default as FormSkeleton } from './FormSkeleton';
export { default as CardGridSkeleton } from './CardGridSkeleton';
```

- [ ] **Step 7: Run the skeleton component tests**

Run: `cd web && npm run test:run -- src/components/LoadingSkeleton/LoadingSkeleton.test.tsx`

Expected: PASS with semantic skeleton test ids present.

- [ ] **Step 8: Commit**

```bash
git add web/src/components/LoadingSkeleton
git commit -m "feat: add shared semantic loading skeletons"
```

### Task 3: Permission Guard and Representative Page Conversion

**Files:**
- Modify: `web/src/components/RBAC/Authorized.tsx`
- Modify: `web/src/pages/Dashboard/Dashboard.tsx`
- Modify: `web/src/pages/Deployment/DeploymentListPage.tsx`
- Modify: `web/src/pages/Services/ServiceListPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Test: `web/src/components/RBAC/Authorized.test.tsx`
- Test: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`

- [ ] **Step 1: Write the failing permission-guard test**

```tsx
import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import Authorized from './Authorized';

vi.mock('./PermissionContext', () => ({
  usePermission: () => ({
    loading: true,
    hasPermission: () => false,
  }),
}));

describe('Authorized', () => {
  it('shows a content skeleton instead of a spinner while permissions load', () => {
    render(<Authorized resource="host" action="read"><div>ok</div></Authorized>);
    expect(screen.getByTestId('page-skeleton')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the guard test and confirm current `Spin` behavior fails it**

Run: `cd web && npm run test:run -- src/components/RBAC/Authorized.test.tsx`

Expected: FAIL because `Authorized` currently renders `Spin`.

- [ ] **Step 3: Replace the permission spinner with a local page skeleton**

```tsx
import { PageSkeleton } from '../LoadingSkeleton';

if (loading) {
  return <PageSkeleton />;
}
```

- [ ] **Step 4: Convert representative list/detail pages from first-load spinner patterns to semantic skeletons**

```tsx
if (loading && !cluster) {
  return <DetailSkeleton summaryCards={3} sections={4} />;
}
```

```tsx
return loading && data.length === 0 ? (
  <TableSkeleton toolbar rows={8} columns={6} />
) : (
  <Table dataSource={rows} /* no first-load Table loading overlay */ />
);
```

- [ ] **Step 5: Preserve lighter refresh semantics on pages that already have visible stale content**

```tsx
const isInitialLoading = loading && deployments.length === 0;

return (
  <>
    <Button icon={<ReloadOutlined />} loading={loading && !isInitialLoading}>刷新</Button>
    {isInitialLoading ? <TableSkeleton toolbar rows={8} columns={6} /> : <Table dataSource={deployments} />}
  </>
);
```

- [ ] **Step 6: Add one detail-page regression test for first-load skeleton behavior**

```tsx
it('renders detail skeleton before cluster data is available', async () => {
  vi.spyOn(Api.cluster, 'getClusterDetail').mockImplementation(
    () => new Promise(() => {})
  );
  render(<MemoryRouter initialEntries={['/deployment/infrastructure/clusters/1']}><ClusterDetailPage /></MemoryRouter>);
  expect(screen.getByTestId('detail-skeleton')).toBeInTheDocument();
});
```

- [ ] **Step 7: Run the focused tests**

Run: `cd web && npm run test:run -- src/components/RBAC/Authorized.test.tsx src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`

Expected: PASS with `PageSkeleton` and `DetailSkeleton` replacing large spinners.

- [ ] **Step 8: Commit**

```bash
git add web/src/components/RBAC/Authorized.tsx \
  web/src/pages/Dashboard/Dashboard.tsx \
  web/src/pages/Deployment/DeploymentListPage.tsx \
  web/src/pages/Services/ServiceListPage.tsx \
  web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx \
  web/src/components/RBAC/Authorized.test.tsx \
  web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx
git commit -m "feat: migrate representative screens to skeleton loading"
```

### Task 4: Bulk Replace Spinner and AntD Loading Patterns Across Major Screens

**Files:**
- Modify: `web/src/pages/Monitor/MonitorPage.tsx`
- Modify: `web/src/pages/Settings/UsersPage.tsx`
- Modify: `web/src/pages/Settings/RolesPage.tsx`
- Modify: `web/src/pages/Settings/PermissionsPage.tsx`
- Modify: `web/src/pages/Settings/AIModelSettingsPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterListPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/CredentialListPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`
- Modify: `web/src/pages/Deployment/Observability/AuditLogsPage.tsx`
- Modify: `web/src/pages/Deployment/Observability/MetricsDashboardPage.tsx`
- Modify: `web/src/pages/Deployment/Observability/DeploymentTopologyPage.tsx`
- Modify: `web/src/pages/Deployment/Observability/AIOpsInsightsPage.tsx`
- Modify: `web/src/pages/Deployment/Observability/PolicyManagementPage.tsx`
- Modify: `web/src/pages/Tools/ToolsPage.tsx`
- Modify: `web/src/pages/Tasks/TasksPage.tsx`
- Modify: `web/src/pages/CMDB/CMDBPage.tsx`
- Modify: `web/src/pages/Hosts/HostListPage.tsx`
- Modify: `web/src/pages/Hosts/HostDetailPage.tsx`
- Modify: `web/src/pages/K8s/K8sPage.tsx`

- [ ] **Step 1: Inventory the remaining spinner and AntD loading call sites before editing**

Run: `cd web && rg -n "Spin|loading=\\{loading\\}|if \\(loading\\) return" src/pages src/components -S`

Expected: a concrete list of remaining files still using blocking or first-load loading states.

- [ ] **Step 2: Convert first-load table pages to `TableSkeleton` and keep refresh feedback local**

```tsx
const isInitialLoading = loading && records.length === 0;

return isInitialLoading ? (
  <TableSkeleton toolbar rows={10} columns={7} />
) : (
  <Table dataSource={records} loading={false} />
);
```

- [ ] **Step 3: Convert dashboard- and monitor-style pages to page/card-grid skeletons**

```tsx
if (loading && !summary) {
  return <PageSkeleton />;
}
```

```tsx
{loading && cards.length === 0 ? (
  <CardGridSkeleton count={4} />
) : (
  <ActualCards />
)}
```

- [ ] **Step 4: Convert detail pages and side panels away from blocking `Spin` wrappers**

```tsx
<Drawer open={open} onClose={close}>
  {nodeDrawerVisible && selectedNode == null ? (
    <DetailSkeleton summaryCards={1} sections={2} />
  ) : (
    <NodeDetails node={selectedNode} />
  )}
</Drawer>
```

- [ ] **Step 5: Remove overlapping `Card loading` and `Table loading` props where custom skeletons now own first-load state**

```tsx
<Card title="集群管理" loading={false}>
  {isInitialLoading ? <TableSkeleton toolbar rows={8} columns={6} /> : <Table dataSource={clusters} />}
</Card>
```

- [ ] **Step 6: Run a grep again to confirm remaining spinner usage is intentional**

Run: `cd web && rg -n "Spin|loading=\\{loading\\}|if \\(loading\\) return" src/pages src/components -S`

Expected: only intentional cases remain, primarily `Button loading`, tiny inline pending indicators, or explicit non-content cases.

- [ ] **Step 7: Commit**

```bash
git add web/src/pages/Monitor/MonitorPage.tsx \
  web/src/pages/Settings/UsersPage.tsx \
  web/src/pages/Settings/RolesPage.tsx \
  web/src/pages/Settings/PermissionsPage.tsx \
  web/src/pages/Settings/AIModelSettingsPage.tsx \
  web/src/pages/Deployment/Infrastructure/ClusterListPage.tsx \
  web/src/pages/Deployment/Infrastructure/CredentialListPage.tsx \
  web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx \
  web/src/pages/Deployment/Observability/AuditLogsPage.tsx \
  web/src/pages/Deployment/Observability/MetricsDashboardPage.tsx \
  web/src/pages/Deployment/Observability/DeploymentTopologyPage.tsx \
  web/src/pages/Deployment/Observability/AIOpsInsightsPage.tsx \
  web/src/pages/Deployment/Observability/PolicyManagementPage.tsx \
  web/src/pages/Tools/ToolsPage.tsx \
  web/src/pages/Tasks/TasksPage.tsx \
  web/src/pages/CMDB/CMDBPage.tsx \
  web/src/pages/Hosts/HostListPage.tsx \
  web/src/pages/Hosts/HostDetailPage.tsx \
  web/src/pages/K8s/K8sPage.tsx
git commit -m "refactor: unify skeleton loading across major pages"
```

### Task 5: Verification and Cleanup

**Files:**
- Modify: `web/src/ProtectedApp.loading.test.tsx`
- Modify: `web/src/components/LoadingSkeleton/LoadingSkeleton.test.tsx`
- Modify: `web/src/components/RBAC/Authorized.test.tsx`
- Modify: any touched page tests that need loading-state updates

- [ ] **Step 1: Add or update tests for refresh-vs-initial semantics on one list page**

```tsx
it('keeps stale table content visible during refresh', async () => {
  render(<DeploymentListPage />);
  expect(await screen.findByText('发布中心')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '刷新' }));
  expect(screen.getByRole('table')).toBeInTheDocument();
  expect(screen.queryByTestId('table-skeleton')).not.toBeInTheDocument();
});
```

- [ ] **Step 2: Run the targeted frontend tests**

Run: `cd web && npm run test:run -- src/ProtectedApp.loading.test.tsx src/components/LoadingSkeleton/LoadingSkeleton.test.tsx src/components/RBAC/Authorized.test.tsx src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`

Expected: PASS

- [ ] **Step 3: Run TypeScript**

Run: `cd web && npx tsc -p tsconfig.json --noEmit`

Expected: PASS

- [ ] **Step 4: Run the full frontend test suite if feasible**

Run: `cd web && npm run test:run`

Expected: PASS, or a short list of unrelated existing failures documented before merge.

- [ ] **Step 5: Manual browser verification**

Run:

```bash
cd web
npm run dev
```

Expected manual checks:
- navigating between sidebar items keeps header and sidebar visible
- route suspense shows page skeletons inside the content area
- representative list pages show table skeletons on first load
- representative detail pages show detail skeletons on first load
- refresh buttons keep visible content on screen where stale data exists
- submit buttons still use `Button loading`

- [ ] **Step 6: Commit final cleanup**

```bash
git add web/src
git commit -m "test: cover unified loading skeleton behavior"
```

## Self-Review

- Spec coverage:
  - route boundary changes are covered in Task 1
  - shared semantic skeleton kit is covered in Task 2
  - permission loading and representative page conversion are covered in Task 3
  - bulk spinner and AntD loading replacement is covered in Task 4
  - automated verification and refresh semantics are covered in Task 5
- Placeholder scan:
  - no `TODO`, `TBD`, or deferred "handle later" language remains
- Type consistency:
  - the plan consistently uses `PageSkeleton`, `TableSkeleton`, `DetailSkeleton`, `FormSkeleton`, and `CardGridSkeleton`
  - first-load semantics are consistently named `isInitialLoading`

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-04-loading-skeleton-unification.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
