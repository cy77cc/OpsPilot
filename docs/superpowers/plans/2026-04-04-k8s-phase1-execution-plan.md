# K8s Phase 1 (Action-First Cluster Console) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 6-8 周内交付“集群详情页可操作闭环”，完成 70% 用户可见能力 + 30% 最小生产底座能力（审批、审计、任务状态、权限）。

**Architecture:** 复用现有 cluster 模块，统一后端操作响应与审批门禁，在前端详情页与操作中心形成“触发操作 -> 审批/执行 -> 审计追踪”的单链路闭环。先覆盖节点/工作负载/Service-Ingress 基础操作，严格限制非目标能力进入 Phase 1。

**Tech Stack:** Go (Gin/GORM/client-go), React + TypeScript + Ant Design, Vitest, existing RBAC + governance services.

---

## File Structure

### Backend (Phase 1)
- Modify: `internal/service/cluster/routes.go`
- Modify: `internal/service/cluster/handler_operations.go`
- Modify: `internal/service/cluster/logic_nodes.go`
- Modify: `internal/service/cluster/logic_services.go`
- Modify: `internal/service/cluster/logic_resources.go`
- Modify: `internal/service/cluster/logic_advanced.go`
- Modify: `internal/service/cluster/operation_response.go`
- Modify: `internal/service/cluster/approval_policy.go`
- Modify: `internal/service/cluster/repository.go`
- Modify/Create tests:
  - `internal/service/cluster/**/*_test.go`
  - `internal/service/governance/approval/service_test.go`
  - `internal/service/governance/audit/service_test.go`

### Frontend (Phase 1)
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`
- Modify: `web/src/api/modules/cluster.ts`
- Modify: `web/src/api/modules/cluster.operations.test.ts`
- Modify: `web/src/ProtectedApp.tsx` (if route/permission wiring changes)

### Docs / Rollout
- Modify/Create: `docs/superpowers/specs/2026-04-04-k8s-platform-roadmap-design.md`
- Modify/Create: `docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md`
- Create: `docs/runbooks/cluster-high-risk-operations.md`

---

## Chunk 1: Contract Baseline (Week 1-2)

### Task 1: 固化统一操作响应与审批语义

**Files:**
- Modify: `internal/service/cluster/operation_response.go`
- Modify: `web/src/api/modules/cluster.ts`
- Test: `web/src/api/modules/cluster.operations.test.ts`

- [ ] **Step 1: 明确后端响应枚举与字段契约**
Run: `rg -n "ClusterOperationResponse|approval_required|audit_id" internal/service/cluster web/src/api/modules/cluster.ts`
Expected: 找到统一状态字段与调用点。

- [ ] **Step 2: 为前端规范化解码补充失败分支测试**
Run: `cd web && npx vitest run src/api/modules/cluster.operations.test.ts`
Expected: 新增用例先失败（缺字段映射或状态归一化问题）。

- [ ] **Step 3: 修正并统一 state/code/approval/audit_id 解析**
Run: `cd web && npx vitest run src/api/modules/cluster.operations.test.ts`
Expected: 通过。

- [ ] **Step 4: 以文档方式冻结契约**
Run: `rg -n "state|approval|audit_id" docs/superpowers/specs/2026-04-04-k8s-platform-roadmap-design.md`
Expected: 规格文档与实现一致。

Acceptance:
- 前后端对 `completed|approval_required|rejected|failed` 的语义一致。

### Task 2: 权限与审批最小闭环回归

**Files:**
- Modify: `internal/service/cluster/approval_policy.go`
- Modify: `internal/service/cluster/handler_operations.go`
- Test: `internal/service/cluster/**/*_test.go`

- [ ] **Step 1: 增加审批 token 缺失/过期/重放测试**
Run: `go test ./internal/service/cluster/... -run "Approval|Token|HighRisk"`
Expected: 新增 case 初始失败。

- [ ] **Step 2: 统一高风险操作门禁调用路径**
Run: `go test ./internal/service/cluster/... -run "Approval|Token|HighRisk"`
Expected: 通过并输出稳定错误码。

- [ ] **Step 3: 权限兼容性回归**
Run: `go test ./internal/service/rbac/...`
Expected: 现有 RBAC 行为不回归。

Acceptance:
- 高风险操作必须可授权、可拒绝、可追踪。

---

## Chunk 2: Node + Workload Actions (Week 2-4)

### Task 3: 节点操作闭环（cordon/uncordon/drain/remove + 标签/污点）

**Files:**
- Modify: `internal/service/cluster/logic_nodes.go`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Test: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`

- [ ] **Step 1: 补全后端节点操作测试矩阵**
Run: `go test ./internal/service/cluster/... -run "Node|Cordon|Drain|Remove|Taint|Label"`
Expected: 缺口 case 失败。

- [ ] **Step 2: 后端补齐失败路径与审计记录**
Run: `go test ./internal/service/cluster/... -run "Node|Cordon|Drain|Remove|Taint|Label"`
Expected: 通过。

- [ ] **Step 3: 前端节点行操作 + 审批弹窗 + 审计链接联动验证**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`
Expected: 通过。

Acceptance:
- 节点操作在详情页可执行，审批和审计链路可视。

### Task 4: 工作负载基础操作（Deployment/StatefulSet/Pod）

**Files:**
- Modify: `internal/service/cluster/routes.go`
- Modify: `internal/service/cluster/logic_resources.go`
- Modify: `web/src/api/modules/cluster.ts`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`

- [ ] **Step 1: 明确 Phase 1 操作最小集接口（重启/扩缩容/删除）**
Run: `rg -n "deployments|statefulsets|pods" internal/service/cluster/routes.go internal/service/cluster/logic_resources.go`
Expected: 列出当前只读接口与缺失写接口。

- [ ] **Step 2: 为新增写接口添加 handler + 审批/审计接入**
Run: `go test ./internal/service/cluster/... -run "Workload|Deployment|Stateful|Pod"`
Expected: 通过。

- [ ] **Step 3: 前端在 workload tab 增加操作入口与状态反馈**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`
Expected: 通过。

Acceptance:
- 详情页内完成工作负载日常操作，无需跳转外部系统。

---

## Chunk 3: Service/Ingress + Operation Center (Week 4-6)

### Task 5: Service/Ingress 基础治理

**Files:**
- Modify: `internal/service/cluster/logic_services.go`
- Modify: `internal/service/cluster/routes.go`
- Modify: `web/src/api/modules/cluster.ts`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`

- [ ] **Step 1: 定义 Phase 1 基础写操作（create/update/delete）与参数校验**
Run: `rg -n "GetServices|GetIngresses" internal/service/cluster`
Expected: 识别新增接口位置。

- [ ] **Step 2: 接入审批策略（仅高风险配置变更）与审计记录**
Run: `go test ./internal/service/cluster/... -run "Service|Ingress|Audit|Approval"`
Expected: 通过。

- [ ] **Step 3: 前端补齐 Service/Ingress 操作 UI 与错误提示**
Run: `cd web && npm run test -- --runInBand`
Expected: 相关测试通过。

Acceptance:
- Service/Ingress 支持基础运维动作并可追踪。

### Task 6: 操作中心增强（筛选、详情、链路回跳）

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`
- Modify: `internal/service/cluster/handler_operations.go`

- [ ] **Step 1: 补齐筛选条件与分页边界测试**
Run: `go test ./internal/service/cluster/... -run "OperationHistory|Audit"`
Expected: 通过。

- [ ] **Step 2: 增强详情面板（审批信息、诊断信息、请求响应摘要）**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`
Expected: 通过（含跳转链路断言）。

- [ ] **Step 3: 从详情页所有高风险动作统一回链到操作中心**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`
Expected: 通过。

Acceptance:
- 用户可按资源/状态/操作者/时间范围快速追溯任何关键操作。

---

## Chunk 4: Production-Ready Hardening (Week 6-8)

### Task 7: 失败恢复与运行手册

**Files:**
- Create: `docs/runbooks/cluster-high-risk-operations.md`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`

- [ ] **Step 1: 为 drain/remove/upgrade/renew 定义失败恢复步骤**
Run: `rg -n "drain|remove|upgrade|renew" web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
Expected: 覆盖核心高风险动作。

- [ ] **Step 2: 在错误提示中注入 runbook 链接或处置建议**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterDetailPage.test.tsx`
Expected: 通过。

Acceptance:
- 高风险失败有明确处置路径，不出现“仅报错无指导”。

### Task 8: 回归、验收与发布控制

**Files:**
- Modify: `e2e/**`（按现有套件补充）
- Modify: `docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md`（状态更新）

- [ ] **Step 1: 执行后端完整回归**
Run: `go test ./...`
Expected: 全量通过。

- [ ] **Step 2: 执行前端回归**
Run: `cd web && npm run test`
Expected: 全量通过。

- [ ] **Step 3: 执行构建验证**
Run: `cd web && npm run build`
Expected: 构建成功。

- [ ] **Step 4: 发布前核对“四可”验收清单**
Run: 手动验收（授权/执行/追踪/审计）
Expected: 所有高风险动作满足四可。

Acceptance:
- Phase 1 达到上线门槛并具备回滚/应急预案。

---

## Execution Sequencing (parallel lanes)

- Lane A (Backend): Task 1/2/3/4/5
- Lane B (Frontend): Task 1/3/4/5/6/7
- Lane C (Quality): Task 2/6/8
- Lane D (Docs/Release): Task 7/8

并行规则：
- 依赖链：Task 1 -> Task 3/4/5 -> Task 6/7 -> Task 8
- Task 2 与 Task 1 可并行，但必须在 Task 4/5 前完成高风险门禁回归。

## Definition of Done

- [ ] 节点、工作负载、Service/Ingress 的 Phase 1 操作能力可用。
- [ ] 高风险动作审批与审计链路完整。
- [ ] 操作中心可筛选、可追踪、可定位详情。
- [ ] 失败恢复策略与 runbook 可执行。
- [ ] 后端测试、前端测试、构建验证全部通过。
- [ ] 非目标能力（Mesh/CNI 深度/GitOps 高级）未被引入 Phase 1。

## Scope Guardrails

- 不新增 Service Mesh 深度治理能力。
- 不引入 CNI/IPAM 深层可视化。
- 不在本阶段交付全量 GitOps/应用市场高级功能。
- 不做 VPA 全面生产化。
