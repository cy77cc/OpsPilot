# K8s Phase 2 NetworkPolicy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 交付“多 CNI NetworkPolicy 可视化编排”Phase 2 能力，达到“读写 + 仿真校验 + 审批审计 + 回滚追踪”闭环。

**Architecture:** 基于统一 DSL + CNI 适配层（Cilium/Calico/Flannel）+ 中心化仿真引擎。发布链路复用现有治理能力（RBAC、审批、审计、操作中心），并保持后续下沉集群侧执行器的接口兼容。Flannel 在无策略引擎时强制阻断发布，避免“可下发但不生效”。

**Tech Stack:** Go (Gin/GORM/client-go), React + TypeScript + Ant Design, Vitest, existing governance (approval/audit/policy), Prometheus metrics.

---

## File Structure

### Backend
- Create: `internal/service/cluster/policy_definition.go`
- Create: `internal/service/cluster/policy_simulation.go`
- Create: `internal/service/cluster/policy_adapter_cilium.go`
- Create: `internal/service/cluster/policy_adapter_calico.go`
- Create: `internal/service/cluster/policy_adapter_flannel.go`
- Create: `internal/service/cluster/policy_release.go`
- Create: `internal/service/cluster/policy_metrics.go`
- Create: `internal/service/cluster/handler_policy.go`
- Modify: `internal/service/cluster/routes.go`
- Modify: `internal/service/cluster/approval_policy.go`
- Modify: `internal/service/cluster/handler_operations.go`
- Modify: `internal/service/cluster/repository.go`
- Test: `internal/service/cluster/policy_*_test.go`
- Test: `internal/service/governance/approval/service_test.go`
- Test: `internal/service/governance/audit/service_test.go`

### Frontend
- Create: `web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx`
- Create: `web/src/components/ClusterPolicy/PolicyTopologyPanel.tsx`
- Create: `web/src/components/ClusterPolicy/PolicySimulationDiffPanel.tsx`
- Create: `web/src/components/ClusterPolicy/PolicyReleasePanel.tsx`
- Modify: `web/src/api/modules/cluster.ts`
- Create: `web/src/api/modules/cluster.policy.test.ts`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`
- Modify: `web/src/ProtectedApp.tsx`

### Docs / Rollout
- Create: `docs/runbooks/cluster-policy-release-and-rollback.md`
- Modify: `docs/superpowers/specs/2026-04-05-k8s-phase2-networkpolicy-design.md`
- Modify: `docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md`

---

## Chunk 1: Contract & Simulation Baseline

### Task 1: 固化统一 DSL、错误码与状态机契约

**Files:**
- Create: `internal/service/cluster/policy_definition.go`
- Modify: `internal/service/cluster/types.go`
- Test: `internal/service/cluster/policy_definition_test.go`

- [ ] **Step 1: 定义 DSL 结构体、状态枚举、错误码常量**
Run: `rg -n "PolicyRelease|simulation|approval_required|FLANNEL_" internal/service/cluster -S`
Expected: 新增类型与常量可被检索。

- [ ] **Step 2: 写失败测试锁定必须字段和默认行为**
Run: `go test ./internal/service/cluster/... -run "PolicyDefinition|PolicyState|PolicyErrorCode"`
Expected: 初始失败（字段或默认值未满足）。

- [ ] **Step 3: 实现最小通过逻辑并补齐序列化兼容**
Run: `go test ./internal/service/cluster/... -run "PolicyDefinition|PolicyState|PolicyErrorCode"`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/policy_definition.go internal/service/cluster/types.go internal/service/cluster/policy_definition_test.go && git commit -m "Freeze policy DSL and release-state contracts for phase-2 rollout"`
Expected: commit 成功。

Acceptance:
- DSL 字段、状态机和错误码具备稳定后续开发基线。

### Task 2: 搭建仿真引擎（冲突检测 + 风险评分）

**Files:**
- Create: `internal/service/cluster/policy_simulation.go`
- Test: `internal/service/cluster/policy_simulation_test.go`

- [ ] **Step 1: 为冲突检测和风险评分写失败测试（含关键命名空间阻断）**
Run: `go test ./internal/service/cluster/... -run "Simulation|RiskScore|BlockingConflict"`
Expected: 初始失败。

- [ ] **Step 2: 实现冲突检测、影响面计算、风险分级**
Run: `go test ./internal/service/cluster/... -run "Simulation|RiskScore|BlockingConflict"`
Expected: 通过。

- [ ] **Step 3: 覆盖 CRITICAL 阈值阻断与 warning 路径**
Run: `go test ./internal/service/cluster/... -run "Critical|Warning|ImpactSummary"`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/policy_simulation.go internal/service/cluster/policy_simulation_test.go && git commit -m "Add simulation engine with blocking conflict rules and risk scoring"`
Expected: commit 成功。

Acceptance:
- 仿真可输出 `blocking_issues/warnings/impact_summary/risk_score`。

---

## Chunk 2: Multi-CNI Adapter Lane

### Task 3: Cilium 适配器（主验收线）

**Files:**
- Create: `internal/service/cluster/policy_adapter_cilium.go`
- Test: `internal/service/cluster/policy_adapter_cilium_test.go`

- [ ] **Step 1: 写失败测试锁定 DSL->Cilium 字段映射**
Run: `go test ./internal/service/cluster/... -run "CiliumAdapter|ToCiliumPolicy"`
Expected: 初始失败。

- [ ] **Step 2: 实现 endpointSelector/toPorts/http/dns/fqdn 映射**
Run: `go test ./internal/service/cluster/... -run "CiliumAdapter|ToCiliumPolicy"`
Expected: 通过。

- [ ] **Step 3: 加入不支持字段阻断（serviceAccount/order）**
Run: `go test ./internal/service/cluster/... -run "CiliumAdapter|UnsupportedField"`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/policy_adapter_cilium.go internal/service/cluster/policy_adapter_cilium_test.go && git commit -m "Implement cilium adapter translation and unsupported-field guards"`
Expected: commit 成功。

Acceptance:
- Cilium 路径满足 Phase 2 主线能力。

### Task 4: Calico 适配器（选择器语法与优先级）

**Files:**
- Create: `internal/service/cluster/policy_adapter_calico.go`
- Test: `internal/service/cluster/policy_adapter_calico_test.go`

- [ ] **Step 1: 写失败测试锁定 selector 语法转换与 order 映射**
Run: `go test ./internal/service/cluster/... -run "CalicoAdapter|Selector|Order"`
Expected: 初始失败。

- [ ] **Step 2: 实现 matchLabels/matchExpressions 到 Calico selector 转换**
Run: `go test ./internal/service/cluster/... -run "CalicoAdapter|Selector|Order"`
Expected: 通过。

- [ ] **Step 3: 对不支持字段（fqdn/L7）输出阻断或降级告警**
Run: `go test ./internal/service/cluster/... -run "CalicoAdapter|SemanticGap|Warning"`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/policy_adapter_calico.go internal/service/cluster/policy_adapter_calico_test.go && git commit -m "Add calico adapter with selector conversion and semantic-gap handling"`
Expected: commit 成功。

Acceptance:
- Calico 适配满足 spec 的兼容矩阵。

### Task 5: Flannel 适配器（能力缺口阻断）

**Files:**
- Create: `internal/service/cluster/policy_adapter_flannel.go`
- Test: `internal/service/cluster/policy_adapter_flannel_test.go`

- [ ] **Step 1: 写失败测试覆盖 netpol 开关、L7 阻断与提示文案**
Run: `go test ./internal/service/cluster/... -run "FlannelAdapter|Netpol|L7"`
Expected: 初始失败。

- [ ] **Step 2: 实现标准 K8s NP 映射与错误码返回**
Run: `go test ./internal/service/cluster/... -run "FlannelAdapter|Netpol|L7"`
Expected: 通过。

- [ ] **Step 3: 校验“无策略引擎时禁止发布”护栏**
Run: `go test ./internal/service/cluster/... -run "FlannelAdapter|PublishBlocked"`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/policy_adapter_flannel.go internal/service/cluster/policy_adapter_flannel_test.go && git commit -m "Enforce flannel guardrails to block non-enforceable policy releases"`
Expected: commit 成功。

Acceptance:
- Flannel 路径不会产生“策略下发成功但不生效”的假闭环。

---

## Chunk 3: Release Workflow & Governance Integration

### Task 6: 发布流水（仿真->审批->下发->回滚）

**Files:**
- Create: `internal/service/cluster/policy_release.go`
- Modify: `internal/service/cluster/approval_policy.go`
- Test: `internal/service/cluster/policy_release_test.go`

- [ ] **Step 1: 写失败测试覆盖状态机转换与审批门禁**
Run: `go test ./internal/service/cluster/... -run "PolicyRelease|StateMachine|Approval"`
Expected: 初始失败。

- [ ] **Step 2: 实现 release_id/previous_stable_version 与状态流转**
Run: `go test ./internal/service/cluster/... -run "PolicyRelease|StateMachine|Approval"`
Expected: 通过。

- [ ] **Step 3: 加入 apply_failed 自动回滚与 rollback_applied 记录**
Run: `go test ./internal/service/cluster/... -run "PolicyRelease|Rollback|ApplyFailed"`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/policy_release.go internal/service/cluster/approval_policy.go internal/service/cluster/policy_release_test.go && git commit -m "Implement policy release state machine with guarded rollback flow"`
Expected: commit 成功。

Acceptance:
- 发布与回滚可追踪且状态语义稳定。

### Task 7: 路由/处理器与治理链路打通

**Files:**
- Create: `internal/service/cluster/handler_policy.go`
- Modify: `internal/service/cluster/routes.go`
- Modify: `internal/service/cluster/handler_operations.go`
- Modify: `internal/service/cluster/repository.go`
- Test: `internal/service/cluster/handler_policy_test.go`

- [ ] **Step 1: 新增 policy/release/cni-info 路由并补 handler 失败测试**
Run: `go test ./internal/service/cluster/... -run "HandlerPolicy|Route|CNIInfo"`
Expected: 初始失败。

- [ ] **Step 2: 实现接口并统一返回 envelope（state/code/approval/audit_id）**
Run: `go test ./internal/service/cluster/... -run "HandlerPolicy|Route|CNIInfo"`
Expected: 通过。

- [ ] **Step 3: 验证审批与审计事件在操作中心可追踪**
Run: `go test ./internal/service/cluster/... -run "PolicyAudit|OperationHistory"`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/handler_policy.go internal/service/cluster/routes.go internal/service/cluster/handler_operations.go internal/service/cluster/repository.go internal/service/cluster/handler_policy_test.go && git commit -m "Expose policy and release APIs with governance-grade operation envelopes"`
Expected: commit 成功。

Acceptance:
- API 可驱动前端闭环，并复用现有治理体系。

---

## Chunk 4: Frontend Productization

### Task 8: 前端 API 层与类型归一化

**Files:**
- Modify: `web/src/api/modules/cluster.ts`
- Create: `web/src/api/modules/cluster.policy.test.ts`

- [ ] **Step 1: 写失败测试锁定 policy/release/simulation 响应解码**
Run: `cd web && npx vitest run src/api/modules/cluster.policy.test.ts`
Expected: 初始失败。

- [ ] **Step 2: 增加 API 方法与类型归一化（含错误码、warning、blocking issues）**
Run: `cd web && npx vitest run src/api/modules/cluster.policy.test.ts`
Expected: 通过。

- [ ] **Step 3: 回归 cluster 既有契约测试**
Run: `cd web && npx vitest run src/api/modules/cluster.operations.test.ts`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add web/src/api/modules/cluster.ts web/src/api/modules/cluster.policy.test.ts && git commit -m "Extend cluster api module for policy simulation and release workflows"`
Expected: commit 成功。

Acceptance:
- 前端 API 层支持完整策略与发布操作。

### Task 9: Cluster Policy Center 页面与交互

**Files:**
- Create: `web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx`
- Create: `web/src/components/ClusterPolicy/PolicyTopologyPanel.tsx`
- Create: `web/src/components/ClusterPolicy/PolicySimulationDiffPanel.tsx`
- Create: `web/src/components/ClusterPolicy/PolicyReleasePanel.tsx`
- Modify: `web/src/ProtectedApp.tsx`

- [ ] **Step 1: 建立页面骨架与路由接入，先写渲染失败测试**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx -t "renders policy center shell"`
Expected: 初始失败。

- [ ] **Step 2: 实现拓扑视图、仿真 diff、发布面板基础交互**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx`
Expected: 通过。

- [ ] **Step 3: 覆盖关键交互：仿真阻断、审批待处理、回滚成功反馈**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx --testTimeout=60000`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.tsx web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx web/src/components/ClusterPolicy/PolicyTopologyPanel.tsx web/src/components/ClusterPolicy/PolicySimulationDiffPanel.tsx web/src/components/ClusterPolicy/PolicyReleasePanel.tsx web/src/ProtectedApp.tsx && git commit -m "Add policy center UI with simulation diff and guarded release actions"`
Expected: commit 成功。

Acceptance:
- 页面具备策略编排核心体验和高风险保护提示。

### Task 10: 详情页/操作中心回链与指标桥接

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`
- Create: `internal/service/cluster/policy_metrics.go`
- Test: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx`

- [ ] **Step 1: 增加 policy release 的操作中心筛选与深链展示测试**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx --testTimeout=60000`
Expected: 初始失败。

- [ ] **Step 2: 实现 policy release 事件回链、release_id 展示与查询过滤**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx --testTimeout=60000`
Expected: 通过。

- [ ] **Step 3: 后端增加关键指标（命中/拒绝/发布耗时/仿真耗时/翻译错误）**
Run: `go test ./internal/service/cluster/... -run "PolicyMetrics|Observability"`
Expected: 通过。

- [ ] **Step 4: 提交**
Run: `git add web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx internal/service/cluster/policy_metrics.go && git commit -m "Link policy releases into operation center and expose policy observability metrics"`
Expected: commit 成功。

Acceptance:
- 操作可追踪、审计可见、指标可观测。

---

## Chunk 5: Hardening, Docs, and Release Controls

### Task 11: 验收回归与上线控制

**Files:**
- Create: `docs/runbooks/cluster-policy-release-and-rollback.md`
- Modify: `docs/superpowers/specs/2026-04-05-k8s-phase2-networkpolicy-design.md`
- Modify: `docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md`

- [ ] **Step 1: 执行后端聚焦回归（cluster + governance）**
Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster/... ./internal/service/governance/...`
Expected: 通过。

- [ ] **Step 2: 执行前端聚焦回归（policy center + operation center + cluster api）**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx src/api/modules/cluster.policy.test.ts --testTimeout=60000`
Expected: 通过。

- [ ] **Step 3: 更新 runbook、路线图状态、Phase 2 验收记录**
Run: `rg -n "Phase 2|NetworkPolicy|Gateway API|Flannel" docs/runbooks docs/superpowers/specs -S`
Expected: 文档可检索到对应内容。

- [ ] **Step 4: 全量门禁检查并记录非范围阻塞**
Run: `go test ./... && cd web && npm run test && npm run build`
Expected: 若非本阶段既有失败存在，必须在发布记录中明确标注，不可隐瞒。

- [ ] **Step 5: 提交**
Run: `git add docs/runbooks/cluster-policy-release-and-rollback.md docs/superpowers/specs/2026-04-05-k8s-phase2-networkpolicy-design.md docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md && git commit -m "Document phase-2 rollout controls and policy release runbook"`
Expected: commit 成功。

Acceptance:
- Phase 2 具备上线门槛与回滚预案。

---

## Execution Sequencing

- Lane A（Backend Core）：Task 1/2/3/4/5
- Lane B（Workflow/Governance）：Task 6/7
- Lane C（Frontend）：Task 8/9/10
- Lane D（Quality/Docs）：Task 11

并行规则：
- 依赖链：Task 1 -> Task 2 -> Task 3/4/5 -> Task 6/7 -> Task 8/9/10 -> Task 11
- Task 3/4/5 可并行，但必须在 Task 6 前完成适配器契约冻结。
- Task 8 可与 Task 6 并行启动 mock/类型层，但真实联调必须等待 Task 7。

## Definition of Done

- [ ] Cilium 路径读写 + 仿真 + 发布 + 回滚完整可用。
- [ ] Calico 路径达到兼容矩阵要求并可审计追踪。
- [ ] Flannel 无策略引擎时发布阻断与修复建议可用。
- [ ] 高风险冲突阻断、CRITICAL 风险拦截、审批门禁全部有效。
- [ ] 操作中心可按 `release_id` 与策略对象追踪全链路。
- [ ] 关键指标（命中/拒绝/发布/仿真/翻译错误）可观测。
- [ ] 回滚 runbook 可执行，灰度发布策略明确。

## Scope Guardrails

- 不引入 Service Mesh 深度治理。
- 不扩展到全量 L7 流量编排平台。
- 不在本阶段做跨云自动迁移。
- 不允许跳过仿真和审批直接发布策略。
