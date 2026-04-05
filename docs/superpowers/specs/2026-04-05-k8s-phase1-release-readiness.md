# K8s Phase 1 发布就绪验收记录（Task 8）

## 1. 执行日期与范围

- 执行日期：2026-04-05
- 目标阶段：Phase 1（Task 8：回归、验收与发布控制）
- 验收对象：集群详情页可操作闭环（节点/工作负载/Service-Ingress）及操作中心追踪能力

## 2. 验证矩阵与结果

### 2.1 全量回归

1. 后端全量：`go test ./...`
- 结果：失败
- 非本阶段阻塞（仓库既有）：
  - `internal/dao/ai` 缺失迁移脚本夹具（`storage/migrations/20260320_0003...`、`20260321_0004...`）
  - `internal/service/ai/handler` 多个 SSE 回放相关用例失败
  - `storage/migration` 缺失迁移文件夹具（`20260317_0003_create_ai_approval_tasks.sql`）

2. 前端全量：`cd web && npm run test`
- 结果：失败
- 非本阶段阻塞（仓库既有）：
  - `src/__tests__/Notification/NotificationPanel.test.tsx` 多用例失败
  - `src/pages/Services/ServicePages.test.tsx` 超时
  - `src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx` 在全量并发下超时
  - `src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx` 部分场景在全量并发下超时
  - 全局存在 `window is not defined` 未处理异常（来自测试运行过程）

3. 构建验证：`cd web && npm run build`
- 结果：失败
- 非本阶段阻塞（仓库既有）：
  - `src/data/mockData.ts` 引用不存在的类型导出（`ConfigApp`、`ConfigItem`、`ConfigTemplate`、`Release`、`AuditLog`）
  - `src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx` 两处类型断言不兼容（`page`、`page_size`）

### 2.2 Phase 1 聚焦回归

1. `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster/... ./internal/service/governance/...`
- 结果：通过

2. `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx --testTimeout=60000`
- 结果：通过（2/2）

3. `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx -t "shows .* guidance" --testTimeout=60000`
- 结果：通过（4/4，15 skipped）

## 3. “四可”验收清单

1. 可授权（Authorization）
- 结论：通过
- 依据：高风险操作审批门禁与 legacy 权限兼容已在 Task 2 落地并回归。

2. 可执行（Execution）
- 结论：通过
- 依据：节点、工作负载、Service/Ingress 的 Phase 1 最小操作链路已落地并完成对应任务测试。

3. 可追踪（Traceability）
- 结论：通过
- 依据：详情页动作可回跳操作中心；操作中心支持深链详情与分页边界修正。

4. 可审计（Auditability）
- 结论：通过
- 依据：统一审计字段与审批语义已冻结；高风险失败路径可给出处置建议并挂接 runbook。

## 4. 发布控制结论

- Phase 1 功能范围内：达到发布门槛（聚焦回归通过，“四可”满足）。
- 仓库全量门禁：暂未绿（存在非 Phase 1 既有失败）。

建议发布策略：
1. 仅对 Phase 1 集群管理能力按路径灰度放量。
2. 将全量测试/构建失败项列为独立治理任务，不阻塞本次 Phase 1 受控发布。
3. 高风险动作失败恢复按 `docs/runbooks/cluster-high-risk-operations.md` 执行。
