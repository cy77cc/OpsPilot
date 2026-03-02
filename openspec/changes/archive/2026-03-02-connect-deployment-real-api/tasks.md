# Tasks: 对接部署管理真实后端 API

## Phase 1: 基础 API

### Task 1: 审计日志 API

**后端任务:**

- [x] 创建 `internal/model/audit_log.go` 数据模型
- [x] 创建 `internal/service/deployment/audit.go` 处理器
- [x] 添加路由 `GET /deploy/audit-logs`
- [x] 数据库迁移

**前端任务:**

- [x] 更新 `web/src/api/modules/deployment.ts` 添加 `getAuditLogs` 方法
- [x] 修改 `web/src/pages/Deployment/Observability/AuditLogsPage.tsx` 调用真实 API

---

### Task 2: 指标统计 API

**后端任务:**

- [x] 创建 `internal/service/deployment/metrics.go` 处理器
- [x] 添加路由 `GET /deploy/metrics/summary`
- [x] 添加路由 `GET /deploy/metrics/trends`
- [x] 实现聚合查询逻辑

**前端任务:**

- [x] 更新 `web/src/api/modules/deployment.ts` 添加指标相关方法
- [x] 修改 `web/src/pages/Deployment/Observability/MetricsDashboardPage.tsx` 调用真实 API

---

## Phase 2: 拓扑和策略

### Task 3: 部署拓扑 API

**后端任务:**

- [x] 创建 `internal/service/deployment/topology.go` 处理器
- [x] 添加路由 `GET /deploy/topology`
- [x] 实现拓扑数据聚合逻辑（从 targets 和 releases 聚合）

**前端任务:**

- [x] 更新 `web/src/api/modules/deployment.ts` 添加 `getTopology` 方法
- [x] 修改 `web/src/pages/Deployment/Observability/DeploymentTopologyPage.tsx` 调用真实 API

---

### Task 4: 策略管理 API

**后端任务:**

- [x] 创建 `internal/model/policy.go` 数据模型
- [x] 创建 `internal/service/deployment/policy.go` 处理器
- [x] 添加 CRUD 路由 `/deploy/policies`
- [x] 数据库迁移

**前端任务:**

- [x] 更新 `web/src/api/modules/deployment.ts` 添加策略 CRUD 方法
- [x] 修改 `web/src/pages/Deployment/Observability/PolicyManagementPage.tsx` 调用真实 API
- [x] 添加创建/编辑策略的表单

---

## Phase 3: AIOps

### Task 5: AIOps API

**后端任务:**

- [x] 创建 `internal/model/aiops.go` 数据模型（RiskFinding, Anomaly, Suggestion）
- [x] 创建 `internal/service/aiops/` 服务模块
- [x] 添加路由 `/aiops/risk-findings`, `/aiops/anomalies`, `/aiops/suggestions`
- [x] 数据库迁移
- [x] 实现基础规则引擎（可选：接入 ML 模型）

**前端任务:**

- [x] 创建 `web/src/api/modules/aiops.ts` API 模块
- [x] 修改 `web/src/pages/Deployment/Observability/AIOpsInsightsPage.tsx` 调用真实 API

---

## 执行顺序

```
Phase 1 (基础)
  Task 1 (审计日志) ──────► Task 2 (指标统计)
                                │
                                ▼
Phase 2 (扩展)
  Task 3 (拓扑) ─────────────► Task 4 (策略)
                                │
                                ▼
Phase 3 (高级)
  Task 5 (AIOps)
```

## 验证步骤

每个 Task 完成后：

1. 后端单元测试通过
2. API 返回正确数据格式
3. 前端页面正确显示数据
4. 无 console 错误

## 文件清单

### 新增文件

```
internal/model/audit_log.go
internal/model/policy.go
internal/model/aiops.go
internal/service/deployment/audit.go
internal/service/deployment/metrics.go
internal/service/deployment/topology.go
internal/service/deployment/policy.go
internal/service/aiops/routes.go
internal/service/aiops/handler.go
internal/service/aiops/logic.go
web/src/api/modules/aiops.ts
```

### 修改文件

```
internal/service/deployment/routes.go
internal/service/service.go
internal/testutil/integration.go
web/src/api/modules/deployment.ts
web/src/pages/Deployment/Observability/AuditLogsPage.tsx
web/src/pages/Deployment/Observability/MetricsDashboardPage.tsx
web/src/pages/Deployment/Observability/DeploymentTopologyPage.tsx
web/src/pages/Deployment/Observability/PolicyManagementPage.tsx
web/src/pages/Deployment/Observability/AIOpsInsightsPage.tsx
```
