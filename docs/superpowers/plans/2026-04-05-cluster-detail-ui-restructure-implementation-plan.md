# Cluster Detail UI Restructure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild cluster detail UX into an action-first overview + domain pages, demote low-priority base info, enforce front/back capability alignment, and start `service/cluster` directory modularization without behavior regressions.

**Architecture:** Keep external routes stable while decomposing the monolithic `ClusterDetailPage` into focused pages (`overview`, `nodes`, `workloads`, `network`, `config-storage`) and wiring shared operation-envelope interaction patterns. Add a capability-matrix artifact as a release gate and perform only phase-1 backend reorganization (move/adapter, no behavior changes). Prefer additive changes first, then controlled migration.

**Tech Stack:** React + TypeScript + Ant Design + React Router + Vitest; Go + Gin + GORM; Markdown docs under `docs/superpowers`.

---

## Scope Check

The spec contains two partially independent streams:
1. UI decomposition and interaction tightening.
2. `service/cluster` backend directory modularization.

To avoid blocking UI delivery on backend migration risk, this plan executes them in two tracks within one plan:
- Track A (Tasks 1-8): UX + capability alignment + interface-level backend deltas.
- Track B (Tasks 9-11): backend directory modularization phase-1 with compatibility adapters.

## File Structure (Lock Before Tasking)

- Create: `docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md`
- Modify: `web/src/ProtectedApp.tsx`
- Modify: `web/src/components/Layout/AppLayout.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterNodesPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterWorkloadsPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterNetworkTrafficPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterConfigStoragePage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterNodesPage.test.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterWorkloadsPage.test.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterNetworkTrafficPage.test.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterConfigStoragePage.test.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx`
- Modify: `internal/service/cluster/handler_phase3_runtime.go`
- Modify: `internal/service/cluster/handler_phase3_runtime_test.go`
- Modify: `internal/service/cluster/routes.go`
- Create: `internal/service/cluster/contracts/cluster_types.go`
- Create: `internal/service/cluster/contracts/operation_envelope.go`
- Create: `internal/service/cluster/contracts/phase3_types.go`
- Modify: `internal/service/cluster/types.go`
- Modify: `internal/service/cluster/operation_response.go`
- Modify: `internal/service/cluster/phase3_types.go`
- Create: `internal/service/cluster/logic/node_ops_logic.go`
- Create: `internal/service/cluster/logic/workload_ops_logic.go`
- Create: `internal/service/cluster/logic/service_logic.go`
- Create: `internal/service/cluster/logic/advanced_logic.go`
- Modify: `internal/service/cluster/logic_nodes.go`
- Modify: `internal/service/cluster/logic_resources.go`
- Modify: `internal/service/cluster/logic_services.go`
- Modify: `internal/service/cluster/logic_advanced.go`

## Task 1: Add Capability Matrix Artifact and Release Gate Baseline

**Files:**
- Create: `docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md`

- [ ] **Step 1: Write failing doc-structure check command**

Run: `rg -n "\| 页面能力 \| 前端入口 \| API 端点 \| 后端 handler \| 测试文件 \| 状态 \|" docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md`
Expected: FAIL (file missing).

- [ ] **Step 2: Create matrix skeleton with status enums and governance rule**

```md
# Cluster Detail Capability Matrix

## Status Enum
- `ready`
- `partial`
- `missing`

## Release Gate
- Any `missing` capability MUST NOT expose clickable entry in UI.

| 页面能力 | 前端入口 | API 端点 | 后端 handler | 测试文件 | 状态 | 备注 |
|---------|---------|---------|------------|--------|------|------|
| 集群健康摘要 | /clusters/:id | GET /clusters/:id | GetClusterDetail | ClusterDetailPage.test.tsx | ready |  |
```

- [ ] **Step 3: Populate at least all capabilities listed in the design spec §5**

```md
| 最近失败操作 | /clusters/:id | GET /clusters/:id/operations/history?status=failed&page_size=5 | ListOperationHistory | ClusterOperationCenterPage.test.tsx | ready | 复用现有 |
| 安全告警摘要 | /clusters/:id | GET /clusters/:id/security/alerts?severity=high&page_size=5 | ListRuntimeAlerts | handler_phase3_runtime_test.go | partial | 需补过滤参数 |
```

- [ ] **Step 4: Re-run structure check**

Run: `rg -n "\| 页面能力 \| 前端入口 \| API 端点 \| 后端 handler \| 测试文件 \| 状态 \|" docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md`
Expected: PASS with one table header match.

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md
git commit -m "docs: add cluster detail capability matrix as release gate"
```

## Task 2: Refactor ClusterDetailPage into Action-First Overview Shell

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`

- [ ] **Step 1: Add failing overview-focused test**

```tsx
it('renders action-first overview and keeps base info in collapsed section', async () => {
  renderClusterDetail();
  expect(await screen.findByText('集群作战面板')).toBeInTheDocument();
  expect(screen.getByText('关键操作台')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: '展开基础信息' })).toBeInTheDocument();
});
```

- [ ] **Step 2: Run the targeted test to verify failure**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx -t "action-first overview" --testTimeout=60000`
Expected: FAIL with missing section labels/buttons.

- [ ] **Step 3: Implement overview shell and collapse base info by default**

```tsx
<Card title="集群作战面板">
  <Row gutter={16}>
    <Col span={16}><OverviewSignalsCard clusterId={clusterId} /></Col>
    <Col span={8}><ActionConsoleCard clusterId={clusterId} /></Col>
  </Row>
</Card>
<Button onClick={() => setInfoExpanded((v) => !v)}>{infoExpanded ? '收起基础信息' : '展开基础信息'}</Button>
{infoExpanded ? (
  <Descriptions size="small" column={2} items={[
    { key: 'name', label: '名称', children: cluster?.name || '-' },
    { key: 'source', label: '来源', children: cluster?.source || '-' },
    { key: 'status', label: '状态', children: cluster?.status || '-' },
    { key: 'endpoint', label: '地址', children: cluster?.endpoint || '-' },
  ]} />
) : null}
```

- [ ] **Step 4: Run full ClusterDetailPage tests**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx --testTimeout=60000`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx
git commit -m "feat(ui): convert cluster detail to action-first overview shell"
```

## Task 3: Add Nodes & Capacity Page and Route

**Files:**
- Create: `web/src/pages/Deployment/Infrastructure/ClusterNodesPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterNodesPage.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`

- [ ] **Step 1: Write failing route + render tests**

```tsx
it('registers /deployment/infrastructure/clusters/:id/nodes route', () => {
  renderProtectedRoutes();
  expect(routeExists('/deployment/infrastructure/clusters/:id/nodes')).toBe(true);
});

it('loads node list with compact table and namespace-independent toolbar', async () => {
  renderNodesPage();
  expect(await screen.findByText('节点与容量')).toBeInTheDocument();
  expect(screen.getByRole('table')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests and verify failure**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterNodesPage.test.tsx --testTimeout=60000`
Expected: FAIL with missing page/module.

- [ ] **Step 3: Implement page with compact nodes table and operation entry links**

```tsx
<Table size="small" rowKey="name" columns={nodeColumns} dataSource={nodes} />
<Link to={`/deployment/infrastructure/clusters/${clusterId}/operations`}>进入操作中心</Link>
```

- [ ] **Step 4: Run tests**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterNodesPage.test.tsx src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx --testTimeout=60000`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/ClusterNodesPage.tsx web/src/pages/Deployment/Infrastructure/ClusterNodesPage.test.tsx web/src/ProtectedApp.tsx
git commit -m "feat(ui): add dedicated nodes and capacity page"
```

## Task 4: Add Workloads Page and Route

**Files:**
- Create: `web/src/pages/Deployment/Infrastructure/ClusterWorkloadsPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterWorkloadsPage.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`

- [ ] **Step 1: Add failing tests for namespace toolbar + compact workload tables**

```tsx
it('renders namespace filter toolbar and workload tabs', async () => {
  renderWorkloadsPage();
  expect(await screen.findByLabelText('Namespace')).toBeInTheDocument();
  expect(screen.getByText('Deployments')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run test (expect fail)**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterWorkloadsPage.test.tsx --testTimeout=60000`
Expected: FAIL.

- [ ] **Step 3: Implement workloads page extracting existing ClusterDetail workloads logic**

```tsx
<Select value={namespace} onChange={setNamespace} options={namespaceOptions} style={{ width: 220 }} />
<Tabs items={[
  { key: 'deployments', label: 'Deployments', children: <Table size="small" rowKey="name" columns={deploymentColumns} dataSource={deployments} /> },
  { key: 'statefulsets', label: 'StatefulSets', children: <Table size="small" rowKey="name" columns={statefulSetColumns} dataSource={statefulsets} /> },
  { key: 'pods', label: 'Pods', children: <Table size="small" rowKey="name" columns={podColumns} dataSource={pods} /> },
]} />
```

- [ ] **Step 4: Run tests**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterWorkloadsPage.test.tsx --testTimeout=60000`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/ClusterWorkloadsPage.tsx web/src/pages/Deployment/Infrastructure/ClusterWorkloadsPage.test.tsx web/src/ProtectedApp.tsx
git commit -m "feat(ui): split workloads into dedicated cluster page"
```

## Task 5: Add Network & Traffic Page and Route

**Files:**
- Create: `web/src/pages/Deployment/Infrastructure/ClusterNetworkTrafficPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterNetworkTrafficPage.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`
- Modify: `web/src/components/Layout/AppLayout.tsx`

- [ ] **Step 1: Write failing tests for network route visibility and service/ingress sections**

```tsx
it('renders service and ingress cards in network traffic page', async () => {
  renderNetworkPage();
  expect(await screen.findByText('网络与流量')).toBeInTheDocument();
  expect(screen.getByText('Services')).toBeInTheDocument();
  expect(screen.getByText('Ingresses')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests (expect fail)**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterNetworkTrafficPage.test.tsx --testTimeout=60000`
Expected: FAIL.

- [ ] **Step 3: Implement page and menu link under infrastructure cluster context**

```tsx
<Route path="/deployment/infrastructure/clusters/:id/network" element={withAuth('cluster', 'read', <ClusterNetworkTrafficPage />)} />
```

```tsx
{ key: '/deployment/infrastructure/clusters/:id/network', label: '网络与流量' }
```

- [ ] **Step 4: Run tests**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterNetworkTrafficPage.test.tsx src/components/Layout/AppLayout.test.tsx --testTimeout=60000`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/ClusterNetworkTrafficPage.tsx web/src/pages/Deployment/Infrastructure/ClusterNetworkTrafficPage.test.tsx web/src/ProtectedApp.tsx web/src/components/Layout/AppLayout.tsx
git commit -m "feat(ui): add network and traffic cluster page and navigation entry"
```

## Task 6: Add Config & Storage Page and Route

**Files:**
- Create: `web/src/pages/Deployment/Infrastructure/ClusterConfigStoragePage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterConfigStoragePage.test.tsx`
- Modify: `web/src/ProtectedApp.tsx`

- [ ] **Step 1: Add failing tests for merged config+storage page**

```tsx
it('shows config and storage sections in one compact page', async () => {
  renderConfigStoragePage();
  expect(await screen.findByText('配置与存储')).toBeInTheDocument();
  expect(screen.getByText('ConfigMaps')).toBeInTheDocument();
  expect(screen.getByText('Persistent Volumes')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests (expect fail)**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterConfigStoragePage.test.tsx --testTimeout=60000`
Expected: FAIL.

- [ ] **Step 3: Implement merged page**

```tsx
<Row gutter={12}>
  <Col span={12}><Card title="ConfigMaps"><Table size="small" rowKey="name" columns={configMapColumns} dataSource={configMaps} /></Card></Col>
  <Col span={12}><Card title="Secrets"><Table size="small" rowKey="name" columns={secretColumns} dataSource={secrets} /></Card></Col>
</Row>
<Card title="Persistent Volumes"><Table size="small" rowKey="name" columns={pvColumns} dataSource={pvs} /></Card>
```

- [ ] **Step 4: Run tests**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterConfigStoragePage.test.tsx --testTimeout=60000`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/ClusterConfigStoragePage.tsx web/src/pages/Deployment/Infrastructure/ClusterConfigStoragePage.test.tsx web/src/ProtectedApp.tsx
git commit -m "feat(ui): merge config and storage into dedicated compact page"
```

## Task 7: Wire Overview Cross-Links to Existing Security/Policy/Operations Pages

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`

- [ ] **Step 1: Add failing tests for quick-entry links**

```tsx
it('renders quick links to security, policy, and operation centers', async () => {
  renderClusterDetail();
  expect(await screen.findByRole('link', { name: '进入安全中心' })).toHaveAttribute('href', '/deployment/infrastructure/clusters/42/security');
  expect(screen.getByRole('link', { name: '进入策略中心' })).toHaveAttribute('href', '/deployment/infrastructure/clusters/42/policies');
  expect(screen.getByRole('link', { name: '查看全部操作' })).toHaveAttribute('href', '/deployment/infrastructure/clusters/42/operations');
});
```

- [ ] **Step 2: Run tests (expect fail)**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx -t "quick links" --testTimeout=60000`
Expected: FAIL.

- [ ] **Step 3: Implement links and compact action panel cards**

```tsx
<Link to={`/deployment/infrastructure/clusters/${clusterId}/security`}>进入安全中心</Link>
<Link to={`/deployment/infrastructure/clusters/${clusterId}/policies`}>进入策略中心</Link>
<Link to={`/deployment/infrastructure/clusters/${clusterId}/operations`}>查看全部操作</Link>
```

- [ ] **Step 4: Run tests**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx --testTimeout=60000`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx
git commit -m "feat(ui): add overview quick links to security policy and operation centers"
```

## Task 8: Fill Capability-Matrix Gaps by Extending Runtime Alert Query Filters

**Files:**
- Modify: `internal/service/cluster/handler_phase3_runtime.go`
- Modify: `internal/service/cluster/handler_phase3_runtime_test.go`
- Modify: `docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md`

- [ ] **Step 1: Add failing backend tests for `severity` + `page_size` filters**

```go
func TestHandlerPhase3Runtime_ListAlertsSupportsSeverityAndPageSize(t *testing.T) {
    // seed critical/high/low events
    // GET /clusters/42/security/alerts?severity=high&page_size=1
    // assert only high and len==1
}
```

- [ ] **Step 2: Run tests (expect fail)**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Runtime_.*Severity.*PageSize"`
Expected: FAIL.

- [ ] **Step 3: Implement query handling in `ListRuntimeAlerts`**

```go
severity := strings.TrimSpace(strings.ToLower(c.Query("severity")))
pageSize := parsePositiveInt(c.DefaultQuery("page_size", "100"), 100)
q := h.svcCtx.DB.WithContext(c.Request.Context()).Where("cluster_id = ?", clusterID)
if severity != "" {
    q = q.Where("severity = ?", severity)
}
q = q.Order("id DESC").Limit(pageSize)
```

- [ ] **Step 4: Mark matrix row from `partial` to `ready`**

```md
| 安全告警摘要 | /clusters/:id | GET /clusters/:id/security/alerts?severity=high&page_size=5 | ListRuntimeAlerts | handler_phase3_runtime_test.go | ready | 支持过滤 |
```

- [ ] **Step 5: Run tests and commit**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "HandlerPhase3Runtime_|Route.*security"`
Expected: PASS.

```bash
git add internal/service/cluster/handler_phase3_runtime.go internal/service/cluster/handler_phase3_runtime_test.go docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md
git commit -m "feat(cluster-runtime): support alert severity and page-size filters for overview signals"
```

## Task 9: Create `contracts/` Package and Compatibility Re-exports (Phase-1 Modularization)

**Files:**
- Create: `internal/service/cluster/contracts/cluster_types.go`
- Create: `internal/service/cluster/contracts/operation_envelope.go`
- Create: `internal/service/cluster/contracts/phase3_types.go`
- Modify: `internal/service/cluster/types.go`
- Modify: `internal/service/cluster/operation_response.go`
- Modify: `internal/service/cluster/phase3_types.go`

- [ ] **Step 1: Add failing compile-time compatibility test**

```go
func TestClusterContracts_TypeAliasCompatibility(t *testing.T) {
    var _ clustercontracts.ClusterDetail = ClusterDetail{}
    var _ = OperationStateCompleted
    var _ = ClusterModePlatformManaged
}
```

- [ ] **Step 2: Run tests (expect fail)**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "ClusterContracts_TypeAliasCompatibility"`
Expected: FAIL (package/types missing).

- [ ] **Step 3: Create contracts package and add alias bridge in old files**

```go
// internal/service/cluster/types.go
package cluster

import clustercontracts "github.com/cy77cc/OpsPilot/internal/service/cluster/contracts"

type ClusterDetail = clustercontracts.ClusterDetail
```

```go
// internal/service/cluster/operation_response.go
const OperationStateCompleted = clustercontracts.OperationStateCompleted
```

- [ ] **Step 4: Run tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "ClusterContracts_TypeAliasCompatibility|OperationResponseConstructors"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/contracts/cluster_types.go internal/service/cluster/contracts/operation_envelope.go internal/service/cluster/contracts/phase3_types.go internal/service/cluster/types.go internal/service/cluster/operation_response.go internal/service/cluster/phase3_types.go
git commit -m "refactor(cluster): introduce contracts package with compatibility aliases"
```

## Task 10: Introduce `logic/` Domain Files with Adapter Preservation

**Files:**
- Create: `internal/service/cluster/logic/node_ops_logic.go`
- Create: `internal/service/cluster/logic/workload_ops_logic.go`
- Create: `internal/service/cluster/logic/service_logic.go`
- Create: `internal/service/cluster/logic/advanced_logic.go`
- Modify: `internal/service/cluster/logic_nodes.go`
- Modify: `internal/service/cluster/logic_resources.go`
- Modify: `internal/service/cluster/logic_services.go`
- Modify: `internal/service/cluster/logic_advanced.go`

- [ ] **Step 1: Add failing behavior-preservation tests around one representative operation per domain**

```go
func TestNodeOps_CompatibilityDrainFlow(t *testing.T) {}
func TestWorkloadOps_CompatibilityScaleDeploymentFlow(t *testing.T) {}
func TestServiceOps_CompatibilityCreateServiceFlow(t *testing.T) {}
func TestAdvancedOps_CompatibilityUpgradeFlow(t *testing.T) {}
```

- [ ] **Step 2: Run tests (expect fail for missing logic package indirection)**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Compatibility.*Flow"`
Expected: FAIL.

- [ ] **Step 3: Move core functions into `logic/` files and keep old files as pass-through wrappers**

```go
// logic_nodes.go
func (h *Handler) executeHighRiskNodeOperation(c *gin.Context, clusterID uint, nodeName, action, approvalToken string, fn func(context.Context, *model.Cluster, *model.ClusterNode, *kubernetes.Clientset) (map[string]any, error)) (ClusterOperationResponse, error) {
    return h.executeHighRiskNodeOperationImpl(c, clusterID, nodeName, action, approvalToken, fn)
}
```

```go
// logic/node_ops_logic.go
func (h *Handler) executeHighRiskNodeOperationImpl(c *gin.Context, clusterID uint, nodeName, action, approvalToken string, fn func(context.Context, *model.Cluster, *model.ClusterNode, *kubernetes.Clientset) (map[string]any, error)) (ClusterOperationResponse, error) {
    // move original executeHighRiskNodeOperation body here without behavior changes
}
```

- [ ] **Step 4: Run focused and package tests**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Compatibility.*Flow|RequireHighRiskApproval|HandlerPhase3"`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/cluster/logic/node_ops_logic.go internal/service/cluster/logic/workload_ops_logic.go internal/service/cluster/logic/service_logic.go internal/service/cluster/logic/advanced_logic.go internal/service/cluster/logic_nodes.go internal/service/cluster/logic_resources.go internal/service/cluster/logic_services.go internal/service/cluster/logic_advanced.go
git commit -m "refactor(cluster): move domain logic into logic package with compatibility wrappers"
```

## Task 11: Final Regression and Documentation Sync

**Files:**
- Modify: `docs/superpowers/specs/2026-04-05-cluster-detail-ui-restructure-design.md`
- Modify: `docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md`

- [ ] **Step 1: Run backend regression suite for touched cluster domains**

Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster -run "Phase3|Policy|Operation|Compatibility|Route"`
Expected: PASS.

- [ ] **Step 2: Run frontend targeted regressions for old+new pages**

Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx src/pages/Deployment/Infrastructure/ClusterNodesPage.test.tsx src/pages/Deployment/Infrastructure/ClusterWorkloadsPage.test.tsx src/pages/Deployment/Infrastructure/ClusterNetworkTrafficPage.test.tsx src/pages/Deployment/Infrastructure/ClusterConfigStoragePage.test.tsx src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx src/pages/Deployment/Infrastructure/ClusterSecurityCenterPage.test.tsx --testTimeout=60000`
Expected: PASS.

- [ ] **Step 3: Sync spec status markers to implemented reality**

```md
> 状态：v3 — UI 路由拆分完成，Capability Matrix ready 项全部落地
```

- [ ] **Step 4: Validate matrix has zero `missing` for exposed entries**

Run: `rg -n "\| .*\| .*\| .*\| .*\| .*\| missing \|" docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md`
Expected: No matches for any UI-exposed entry.

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/specs/2026-04-05-cluster-detail-ui-restructure-design.md docs/superpowers/specs/2026-04-05-cluster-detail-capability-matrix.md
git commit -m "docs(cluster-ui): sync implementation status and capability matrix after restructure"
```
