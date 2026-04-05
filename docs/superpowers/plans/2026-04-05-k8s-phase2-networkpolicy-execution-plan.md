# K8s Phase 2 NetworkPolicy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **DOCUMENTATION-FIRST POLICY:** Before implementing any new component or framework integration, you MUST query official documentation first using Context7 MCP or vendor docs. Search order: (1) Context7 docs, (2) GitHub official repo, (3) Exa web search. Document findings in task comments.

**Goal:** 交付"多 CNI NetworkPolicy 可视化编排"Phase 2 能力，达到"读写 + 仿真校验 + 审批审计 + 回滚追踪"闭环。

**Architecture:** 基于统一 DSL + CNI 适配层（Cilium/Calico/Flannel）+ 中心化仿真引擎。发布链路复用现有治理能力（RBAC、审批、审计、操作中心），并保持后续下沉集群侧执行器的接口兼容。Flannel 在无策略引擎时强制阻断发布，避免"可下发但不生效"。

**Tech Stack:** Go (Gin/GORM/client-go), React + TypeScript + Ant Design, Vitest, existing governance (approval/audit/policy), Prometheus metrics.

**Key Documentation References:**
- Kubernetes NetworkPolicy: https://kubernetes.io/docs/concepts/services-networking/network-policies/
- Cilium NetworkPolicy: https://docs.cilium.io/en/stable/network/kubernetes/policy.html
- Calico NetworkPolicy: https://docs.tigera.io/calico/latest/reference/resources/networkpolicy
- Flannel NetPol: https://github.com/flannel-io/flannel/blob/master/Documentation/netpol.md
- Gateway API: https://gateway-api.sigs.k8s.io/

---

## File Structure

### Backend
- Create: `internal/model/policy_release.go`
- Create: `internal/model/policy_definition.go`
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
- Create: `internal/database/migrations/20260405_create_policy_releases.sql`
- Test: `internal/service/cluster/policy_*_test.go`
- Test: `internal/service/governance/approval/service_test.go`
- Test: `internal/service/governance/audit/service_test.go`

### Frontend
- Create: `web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx`
- Create: `web/src/components/ClusterPolicy/PolicyTopologyPanel.tsx`
- Create: `web/src/components/ClusterPolicy/PolicySimulationDiffPanel.tsx`
- Create: `web/src/components/ClusterPolicy/PolicyReleasePanel.tsx`
- Create: `web/src/e2e/policy-release-flow.test.ts`
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

## Chunk 0: Documentation Research & Schema Design

### Task 0: 数据库 Schema 设计与文档调研

**PREREQUISITE:** Before starting, query documentation for K8s NetworkPolicy, Cilium, Calico data models.

**Files:**
- Create: `internal/model/policy_release.go`
- Create: `internal/model/policy_definition.go`
- Create: `internal/database/migrations/20260405_create_policy_releases.sql`
- Create: `docs/research/001-cni-policy-data-models.md`

- [ ] **Step 1: 调研 K8s NetworkPolicy、CiliumNetworkPolicy、Calico NetworkPolicy 的 CRD 结构**
Run: Use Context7 MCP to query `/kubernetes/website`, `/cilium/cilium`, `/projectcalico/calico` for policy spec structures
Expected: 产出数据模型对比文档 `docs/research/001-cni-policy-data-models.md`

- [ ] **Step 2: 设计 PolicyRelease 和 PolicyDefinition 的 GORM 模型**
Run: `go test ./internal/model/... -run "PolicyReleaseModel|PolicyDefinitionModel"`
Expected: 初始失败

- [ ] **Step 3: 编写迁移脚本并验证前向/回滚兼容**
Run: 执行迁移脚本，验证 `up` 和 `down` 均可执行
Expected: 无错误

- [ ] **Step 4: 提交**
Run: `git add internal/model/ internal/database/migrations/20260405_create_policy_releases.sql docs/research/001-cni-policy-data-models.md && git commit -m "Design policy data models and migrations based on CNI research"`
Expected: commit 成功

Acceptance:
- 数据模型覆盖 design.md 中 PolicyRelease 所有字段
- 迁移脚本可回滚
- 文档记录 CNI 策略数据模型差异

---

## Chunk 1: Contract & Simulation Baseline

### Task 1: 固化统一 DSL、错误码与状态机契约

**PREREQUISITE:** Review design.md Section 4.1.1 (DSL Design) before implementation.

**Files:**
- Create: `internal/service/cluster/policy_definition.go`
- Modify: `internal/service/cluster/types_policy.go`
- Test: `internal/service/cluster/policy_definition_test.go`

- [ ] **Step 1: 定义 DSL 结构体、状态枚举、错误码常量**
Run: `rg -n "PolicyRelease|simulation|approval_required|FLANNEL_" internal/service/cluster -S`
Expected: 新增类型与常量可被检索

- [ ] **Step 2: 写失败测试锁定必须字段和默认行为**
Run: `go test ./internal/service/cluster/... -run "PolicyDefinition|PolicyState|PolicyErrorCode"`
Expected: 初始失败（字段或默认值未满足）

- [ ] **Step 3: 实现最小通过逻辑并补齐序列化兼容**
Run: `go test ./internal/service/cluster/... -run "PolicyDefinition|PolicyState|PolicyErrorCode"`
Expected: 通过

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/policy_definition.go internal/service/cluster/types_policy.go internal/service/cluster/policy_definition_test.go && git commit -m "Freeze policy DSL and release-state contracts for phase-2 rollout"`
Expected: commit 成功

Acceptance:
- DSL 字段、状态机和错误码具备稳定后续开发基线
- 支持 JSON/YAML 双向序列化

### Task 2: 搭建仿真引擎（冲突检测 + 风险评分）

**PREREQUISITE:** Review design.md Section 4.2 (Simulation Engine) and conflict detection rules.

**Files:**
- Create: `internal/service/cluster/policy_simulation.go`
- Test: `internal/service/cluster/policy_simulation_test.go`

- [ ] **Step 1: 为冲突检测和风险评分写失败测试（含关键命名空间阻断）**
Run: `go test ./internal/service/cluster/... -run "Simulation|RiskScore|BlockingConflict"`
Expected: 初始失败

- [ ] **Step 2: 实现冲突检测、影响面计算、风险分级**
Run: `go test ./internal/service/cluster/... -run "Simulation|RiskScore|BlockingConflict"`
Expected: 通过

- [ ] **Step 3: 覆盖 CRITICAL 阈值阻断与 warning 路径**
Run: `go test ./internal/service/cluster/... -run "Critical|Warning|ImpactSummary"`
Expected: 通过

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/policy_simulation.go internal/service/cluster/policy_simulation_test.go && git commit -m "Add simulation engine with blocking conflict rules and risk scoring"`
Expected: commit 成功

Acceptance:
- 仿真可输出 `blocking_issues/warnings/impact_summary/risk_score`
- 风险评分算法符合 design.md 公式

---

## Chunk 2: Multi-CNI Adapter Lane

### Task 3: Cilium 适配器（主验收线）

**PREREQUISITE:** MUST query Cilium documentation before implementation.
Run: Use Context7 MCP to query `/cilium/cilium` or `/websites/cilium_io_en_stable` for:
- CiliumNetworkPolicy CRD spec (apiVersion: cilium.io/v2)
- endpointSelector, toPorts, toFQDNs fields
- L7 HTTP/DNS rules syntax

**Files:**
- Create: `internal/service/cluster/policy_adapter_cilium.go`
- Test: `internal/service/cluster/policy_adapter_cilium_test.go`
- Create: `docs/research/002-cilium-network-policy-spec.md`

- [ ] **Step 1: 调研 Cilium NetworkPolicy CRD 结构并记录**
Expected: 产出 `docs/research/002-cilium-network-policy-spec.md`

- [ ] **Step 2: 写失败测试锁定 DSL->Cilium 字段映射**
Run: `go test ./internal/service/cluster/... -run "CiliumAdapter|ToCiliumPolicy"`
Expected: 初始失败

- [ ] **Step 3: 实现 endpointSelector/toPorts/http/dns/fqdn 映射**
Run: `go test ./internal/service/cluster/... -run "CiliumAdapter|ToCiliumPolicy"`
Expected: 通过

- [ ] **Step 4: 加入不支持字段阻断（serviceAccount/order）**
Run: `go test ./internal/service/cluster/... -run "CiliumAdapter|UnsupportedField"`
Expected: 通过

- [ ] **Step 5: 提交**
Run: `git add internal/service/cluster/policy_adapter_cilium.go internal/service/cluster/policy_adapter_cilium_test.go docs/research/002-cilium-network-policy-spec.md && git commit -m "Implement cilium adapter translation and unsupported-field guards"`
Expected: commit 成功

Acceptance:
- Cilium 路径满足 Phase 2 主线能力
- 文档记录 Cilium CRD 字段映射表

### Task 4: Calico 适配器（选择器语法与优先级）

**PREREQUISITE:** MUST query Calico documentation before implementation.
Run: Use Context7 MCP to query `/projectcalico/calico` or `/websites/tigera_io_calico` for:
- NetworkPolicy and GlobalNetworkPolicy spec
- selector expression syntax (==, in, has, !)
- order, serviceAccountSelector, doNotTrack fields

**Files:**
- Create: `internal/service/cluster/policy_adapter_calico.go`
- Test: `internal/service/cluster/policy_adapter_calico_test.go`
- Create: `docs/research/003-calico-network-policy-spec.md`

- [ ] **Step 1: 调研 Calico NetworkPolicy CRD 结构并记录**
Expected: 产出 `docs/research/003-calico-network-policy-spec.md`

- [ ] **Step 2: 写失败测试锁定 selector 语法转换与 order 映射**
Run: `go test ./internal/service/cluster/... -run "CalicoAdapter|Selector|Order"`
Expected: 初始失败

- [ ] **Step 3: 实现 matchLabels/matchExpressions 到 Calico selector 转换**
Run: `go test ./internal/service/cluster/... -run "CalicoAdapter|Selector|Order"`
Expected: 通过

- [ ] **Step 4: 对不支持字段（fqdn/L7）输出阻断或降级告警**
Run: `go test ./internal/service/cluster/... -run "CalicoAdapter|SemanticGap|Warning"`
Expected: 通过

- [ ] **Step 5: 提交**
Run: `git add internal/service/cluster/policy_adapter_calico.go internal/service/cluster/policy_adapter_calico_test.go docs/research/003-calico-network-policy-spec.md && git commit -m "Add calico adapter with selector conversion and semantic-gap handling"`
Expected: commit 成功

Acceptance:
- Calico 适配满足 spec 的兼容矩阵
- 选择器语法转换函数覆盖所有 K8s LabelSelector 操作符

### Task 5: Flannel 适配器（能力缺口阻断）

**PREREQUISITE:** MUST query Flannel documentation before implementation.
Run: Use Context7 MCP to query `/flannel-io/flannel` for:
- Network policy controller enablement (netpol.enabled)
- Supported K8s NetworkPolicy features
- Limitations

**Files:**
- Create: `internal/service/cluster/policy_adapter_flannel.go`
- Test: `internal/service/cluster/policy_adapter_flannel_test.go`
- Create: `docs/research/004-flannel-network-policy-limitations.md`

- [ ] **Step 1: 调研 Flannel 网络策略能力并记录**
Expected: 产出 `docs/research/004-flannel-network-policy-limitations.md`

- [ ] **Step 2: 写失败测试覆盖 netpol 开关、L7 阻断与提示文案**
Run: `go test ./internal/service/cluster/... -run "FlannelAdapter|Netpol|L7"`
Expected: 初始失败

- [ ] **Step 3: 实现标准 K8s NP 映射与错误码返回**
Run: `go test ./internal/service/cluster/... -run "FlannelAdapter|Netpol|L7"`
Expected: 通过

- [ ] **Step 4: 校验"无策略引擎时禁止发布"护栏**
Run: `go test ./internal/service/cluster/... -run "FlannelAdapter|PublishBlocked"`
Expected: 通过

- [ ] **Step 5: 提交**
Run: `git add internal/service/cluster/policy_adapter_flannel.go internal/service/cluster/policy_adapter_flannel_test.go docs/research/004-flannel-network-policy-limitations.md && git commit -m "Enforce flannel guardrails to block non-enforceable policy releases"`
Expected: commit 成功

Acceptance:
- Flannel 路径不会产生"策略下发成功但不生效"的假闭环
- 错误码文案包含修复建议（如 helm 命令）

---

## Chunk 3: Release Workflow & Governance Integration

### Task 6: 发布流水（仿真->审批->下发->回滚）

**PREREQUISITE:** Review design.md Section 4.4 (Release Service) and Section 5.3 (State Machine).

**Files:**
- Create: `internal/service/cluster/policy_release.go`
- Modify: `internal/service/cluster/approval_policy.go`
- Test: `internal/service/cluster/policy_release_test.go`

- [ ] **Step 1: 写失败测试覆盖状态机转换与审批门禁**
Run: `go test ./internal/service/cluster/... -run "PolicyRelease|StateMachine|Approval"`
Expected: 初始失败

- [ ] **Step 2: 实现 release_id/previous_stable_version 与状态流转**
Run: `go test ./internal/service/cluster/... -run "PolicyRelease|StateMachine|Approval"`
Expected: 通过

- [ ] **Step 3: 加入 apply_failed 自动回滚与 rollback_applied 记录**
Run: `go test ./internal/service/cluster/... -run "PolicyRelease|Rollback|ApplyFailed"`
Expected: 通过

- [ ] **Step 4: 补充回滚端到端测试和审批令牌校验测试**
Run: `go test ./internal/service/cluster/... -run "RollbackEndToEnd|ApprovalTokenValidation"`
Expected: 通过

- [ ] **Step 5: 提交**
Run: `git add internal/service/cluster/policy_release.go internal/service/cluster/approval_policy.go internal/service/cluster/policy_release_test.go && git commit -m "Implement policy release state machine with guarded rollback flow"`
Expected: commit 成功

Acceptance:
- 发布与回滚可追踪且状态语义稳定
- 回滚端到端测试覆盖
- 审批令牌失效场景测试覆盖

### Task 7: 路由/处理器与治理链路打通

**Files:**
- Create: `internal/service/cluster/handler_policy.go`
- Modify: `internal/service/cluster/routes.go`
- Modify: `internal/service/cluster/handler_operations.go`
- Modify: `internal/service/cluster/repository.go`
- Test: `internal/service/cluster/handler_policy_test.go`

- [ ] **Step 1: 新增 policy/release/cni-info 路由并补 handler 失败测试**
Run: `go test ./internal/service/cluster/... -run "HandlerPolicy|Route|CNIInfo"`
Expected: 初始失败

- [ ] **Step 2: 实现接口并统一返回 envelope（state/code/approval/audit_id）**
Run: `go test ./internal/service/cluster/... -run "HandlerPolicy|Route|CNIInfo"`
Expected: 通过

- [ ] **Step 3: 验证审批与审计事件在操作中心可追踪**
Run: `go test ./internal/service/cluster/... -run "PolicyAudit|OperationHistory"`
Expected: 通过

- [ ] **Step 4: 提交**
Run: `git add internal/service/cluster/handler_policy.go internal/service/cluster/routes.go internal/service/cluster/handler_operations.go internal/service/cluster/repository.go internal/service/cluster/handler_policy_test.go && git commit -m "Expose policy and release APIs with governance-grade operation envelopes"`
Expected: commit 成功

Acceptance:
- API 可驱动前端闭环，并复用现有治理体系
- CNI 能力发现接口 `GET /api/v1/clusters/{cluster_id}/cni-info` 可用

---

## Chunk 4: Frontend Productization

### Task 8: 前端 API 层与类型归一化

**PREREQUISITE:** Review design.md Section 10 (API Endpoints Index).

**Files:**
- Modify: `web/src/api/modules/cluster.ts`
- Create: `web/src/api/modules/cluster.policy.test.ts`

- [ ] **Step 1: 写失败测试锁定 policy/release/simulation 响应解码**
Run: `cd web && npx vitest run src/api/modules/cluster.policy.test.ts`
Expected: 初始失败

- [ ] **Step 2: 增加 API 方法与类型归一化（含错误码、warning、blocking issues）**
Run: `cd web && npx vitest run src/api/modules/cluster.policy.test.ts`
Expected: 通过

- [ ] **Step 3: 回归 cluster 既有契约测试**
Run: `cd web && npx vitest run src/api/modules/cluster.operations.test.ts`
Expected: 通过

- [ ] **Step 4: 提交**
Run: `git add web/src/api/modules/cluster.ts web/src/api/modules/cluster.policy.test.ts && git commit -m "Extend cluster api module for policy simulations and release workflows"`
Expected: commit 成功

Acceptance:
- 前端 API 层支持完整策略与发布操作
- 响应类型覆盖 blocking_issues/warnings/impact_summary

### Task 9: Cluster Policy Center 页面与交互

**PREREQUISITE:** Review design.md Section 4.5 (Visualization UI).

**Files:**
- Create: `web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.tsx`
- Create: `web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx`
- Create: `web/src/components/ClusterPolicy/PolicyTopologyPanel.tsx`
- Create: `web/src/components/ClusterPolicy/PolicySimulationDiffPanel.tsx`
- Create: `web/src/components/ClusterPolicy/PolicyReleasePanel.tsx`
- Modify: `web/src/ProtectedApp.tsx`

- [ ] **Step 1: 建立页面骨架与路由接入，先写渲染失败测试**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx -t "renders policy center shell"`
Expected: 初始失败

- [ ] **Step 2: 实现拓扑视图、仿真 diff、发布面板基础交互**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx`
Expected: 通过

- [ ] **Step 3: 覆盖关键交互：仿真阻断、审批待处理、回滚成功反馈**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx --testTimeout=60000`
Expected: 通过

- [ ] **Step 4: 提交**
Run: `git add web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.tsx web/src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx web/src/components/ClusterPolicy/PolicyTopologyPanel.tsx web/src/components/ClusterPolicy/PolicySimulationDiffPanel.tsx web/src/components/ClusterPolicy/PolicyReleasePanel.tsx web/src/ProtectedApp.tsx && git commit -m "Add policy center UI with simulation diff and guarded release actions"`
Expected: commit 成功

Acceptance:
- 页面具备策略编排核心体验和高风险保护提示

### Task 9.5: 前端集成测试（E2E lite）

**PREREQUISITE:** Review existing E2E patterns in the codebase.

**Files:**
- Create: `web/src/e2e/policy-release-flow.test.ts`

- [ ] **Step 1: 调研现有 E2E 测试框架和模式**
Run: `rg -l "e2e|playwright|vitest.*e2e" web/`
Expected: 找到现有 E2E 测试文件

- [ ] **Step 2: 草稿创建 → 仿真 → 发布申请流程测试**
Run: `cd web && npx vitest run src/e2e/policy-release-flow.test.ts -t "complete policy release flow"`
Expected: 通过

- [ ] **Step 3: 仿真阻断 UI 反馈测试**
Run: `cd web && npx vitest run src/e2e/policy-release-flow.test.ts -t "simulation blocking UI feedback"`
Expected: 通过

- [ ] **Step 4: 回滚操作与成功反馈测试**
Run: `cd web && npx vitest run src/e2e/policy-release-flow.test.ts -t "rollback success feedback"`
Expected: 通过

- [ ] **Step 5: 提交**
Run: `git add web/src/e2e/policy-release-flow.test.ts && git commit -m "Add E2E lite tests for policy release flow"`
Expected: commit 成功

Acceptance:
- 端到端覆盖核心发布流程
- UI 交互反馈测试通过

### Task 10: 详情页/操作中心回链与指标桥接

**PREREQUISITE:** Review Prometheus metrics best practices and existing metrics patterns.

**Files:**
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx`
- Modify: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx`
- Create: `internal/service/cluster/policy_metrics.go`
- Test: `web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx`

- [ ] **Step 1: 增加 policy release 的操作中心筛选与深链展示测试**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx --testTimeout=60000`
Expected: 初始失败

- [ ] **Step 2: 实现 policy release 事件回链、release_id 展示与查询过滤**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx --testTimeout=60000`
Expected: 通过

- [ ] **Step 3: 后端增加关键指标（命中/拒绝/发布耗时/仿真耗时/翻译错误）**
Run: `go test ./internal/service/cluster/... -run "PolicyMetrics|Observability"`
Expected: 通过

**指标定义:**
| 指标名称 | 类型 | Buckets/Labels |
|---------|------|---------------|
| `policy_hit_total` | Counter | Labels: policy_name, action, direction, namespace |
| `policy_deny_total` | Counter | Labels: policy_name, namespace |
| `policy_release_duration_seconds` | Histogram | Buckets: 0.5, 1, 2, 5, 10; Labels: phase |
| `simulation_evaluation_duration_seconds` | Histogram | Buckets: 0.1, 0.5, 1, 2 |
| `cni_adapter_translation_errors_total` | Counter | Labels: cni_type, error_code |

- [ ] **Step 4: 提交**
Run: `git add web/src/pages/Deployment/Infrastructure/ClusterDetailPage.tsx web/src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.tsx internal/service/cluster/policy_metrics.go && git commit -m "Link policy releases into operation center and expose policy observability metrics"`
Expected: commit 成功

Acceptance:
- 操作可追踪、审计可见、指标可观测
- Prometheus 可 scrape 到所有定义指标
- Grafana 仪表板可展示核心指标

### Task 10.5: Gateway API 关联展示

**PREREQUISITE:** MUST query Gateway API documentation before implementation.
Run: Use Context7 MCP to query `/kubernetes-sigs/gateway-api` or `/websites/gateway-api_sigs_k8s_io` for:
- HTTPRoute, GRPCRoute spec
- backendRefs, matches, filters fields
- Relationship to NetworkPolicy

**Files:**
- Create: `web/src/components/ClusterPolicy/GatewayPolicyAssociationPanel.tsx`
- Create: `docs/research/005-gateway-api-spec.md`

- [ ] **Step 1: 调研 Gateway API HTTPRoute/GRPCRoute 结构并记录**
Expected: 产出 `docs/research/005-gateway-api-spec.md`

- [ ] **Step 2: 实现 Gateway API 资源与 NetworkPolicy 关联展示组件**
Run: `cd web && npx vitest run -t "GatewayApiAssociation"`
Expected: 通过

- [ ] **Step 3: 提交**
Run: `git add web/src/components/ClusterPolicy/GatewayPolicyAssociationPanel.tsx docs/research/005-gateway-api-spec.md && git commit -m "Add Gateway API association panel for NetworkPolicy context"`
Expected: commit 成功

Acceptance:
- HTTPRoute/GRPCRoute 与 NetworkPolicy 关联关系可视化
- 支持从 Gateway 资源跳转到关联策略

---

## Chunk 5: Hardening, Docs, and Release Controls

### Task 11: 验收回归与上线控制

**Files:**
- Create: `docs/runbooks/cluster-policy-release-and-rollback.md`
- Modify: `docs/superpowers/specs/2026-04-05-k8s-phase2-networkpolicy-design.md`
- Modify: `docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md`

- [ ] **Step 1: 执行后端聚焦回归（cluster + governance）**
Run: `GOCACHE=/tmp/go-build-cache go test ./internal/service/cluster/... ./internal/service/governance/...`
Expected: 通过

- [ ] **Step 2: 执行前端聚焦回归（policy center + operation center + cluster api）**
Run: `cd web && npx vitest run src/pages/Deployment/Infrastructure/ClusterPolicyCenterPage.test.tsx src/pages/Deployment/Infrastructure/ClusterOperationCenterPage.test.tsx src/api/modules/cluster.policy.test.ts --testTimeout=60000`
Expected: 通过

- [ ] **Step 3: 执行 E2E 流程测试**
Run: `cd web && npx vitest run src/e2e/policy-release-flow.test.ts --testTimeout=120000`
Expected: 通过

- [ ] **Step 4: 编写 Runbook（必需章节：前置条件、发布流程、回滚流程、故障排查、升级降级策略）**
Expected: `docs/runbooks/cluster-policy-release-and-rollback.md` 包含所有必需章节

- [ ] **Step 5: 更新 runbook、路线图状态、Phase 2 验收记录**
Run: `rg -n "Phase 2|NetworkPolicy|Gateway API|Flannel" docs/runbooks docs/superpowers/specs -S`
Expected: 文档可检索到对应内容

- [ ] **Step 6: 全量门禁检查并记录非范围阻塞**
Run: `go test ./... && cd web && npm run test && npm run build`
Expected: 若非本阶段既有失败存在，必须在发布记录中明确标注，不可隐瞒

- [ ] **Step 7: 提交**
Run: `git add docs/runbooks/cluster-policy-release-and-rollback.md docs/superpowers/specs/2026-04-05-k8s-phase2-networkpolicy-design.md docs/superpowers/specs/2026-04-04-k8s-management-roadmap-chart.md && git commit -m "Document phase-2 rollout controls and policy release runbook"`
Expected: commit 成功

Acceptance:
- Phase 2 具备上线门槛与回滚预案
- Runbook 可执行，包含故障排查章节

---

## Execution Sequencing

### Parallel Lanes

```
Lane A（Backend Core）：Task 0 -> Task 1 -> Task 2 -> Task 3/4/5 -> Task 6 -> Task 7
Lane B（Frontend）：    Task 8 -> Task 9 -> Task 9.5 -> Task 10 -> Task 10.5
Lane C（Quality/Docs）：                              Task 11
```

### Dependency Rules

| Dependency | Blocks | Reason |
|------------|--------|--------|
| Task 0 (DB Schema) | Task 1, Task 6 | 数据模型是 DSL 和持久化的基础 |
| Task 1 (DSL) | Task 2, Task 3/4/5 | 适配器依赖统一 DSL |
| Task 2 (Simulation Core) | Task 6 | 发布流水依赖仿真引擎 |
| Task 3/4/5 (Adapters) | Task 2 (语义降级) | 仿真需要适配器返回能力缺口 |
| Task 6 (Release) | Task 8/9 (Frontend) | 前端联调需要后端 API |
| Task 7 (Handler) | Task 8/9 (Frontend) | 前端联调需要路由可用 |
| Task 9 (UI) | Task 9.5 (E2E) | E2E 依赖页面完成 |
| Task 10 (Metrics) | Task 11 (Hardening) | 验收需要指标可观测 |

### Optimized Timeline

| Week | Tasks |
|------|-------|
| Week 1 | Task 0 (DB + Docs), Task 1 (DSL) |
| Week 2 | Task 2 (Simulation), Task 3 (Cilium + Docs) |
| Week 3 | Task 4 (Calico + Docs), Task 5 (Flannel + Docs) |
| Week 4 | Task 6 (Release), Task 7 (Handler) |
| Week 5 | Task 8 (Frontend API), Task 9 (UI) |
| Week 6 | Task 9.5 (E2E), Task 10 (Metrics), Task 10.5 (Gateway API) |
| Week 7 | Task 11 (Hardening + Docs + Runbook) |

---

## Definition of Done

- [ ] Cilium 路径读写 + 仿真 + 发布 + 回滚完整可用
- [ ] Calico 路径达到兼容矩阵要求并可审计追踪
- [ ] Flannel 无策略引擎时发布阻断与修复建议可用
- [ ] 高风险冲突阻断、CRITICAL 风险拦截、审批门禁全部有效
- [ ] 操作中心可按 `release_id` 与策略对象追踪全链路
- [ ] 关键指标（命中/拒绝/发布/仿真/翻译错误）可观测且可告警
- [ ] 回滚 runbook 可执行，灰度发布策略明确
- [ ] E2E 测试覆盖核心发布流程
- [ ] Gateway API 关联展示可用
- [ ] 所有研究文档（001-005）已归档

## Scope Guardrails

- 不引入 Service Mesh 深度治理（灰度/熔断/限流）
- 不扩展到全量 L7 流量编排平台
- 不在本阶段做跨云自动迁移
- 不允许跳过仿真和审批直接发布策略
- 不在无文档调研的情况下实现新组件

## Documentation Deliverables

| Doc ID | Title | Location |
|--------|-------|----------|
| 001 | CNI Policy Data Models Comparison | `docs/research/001-cni-policy-data-models.md` |
| 002 | Cilium NetworkPolicy CRD Spec | `docs/research/002-cilium-network-policy-spec.md` |
| 003 | Calico NetworkPolicy CRD Spec | `docs/research/003-calico-network-policy-spec.md` |
| 004 | Flannel NetworkPolicy Limitations | `docs/research/004-flannel-network-policy-limitations.md` |
| 005 | Gateway API HTTPRoute/GRPCRoute Spec | `docs/research/005-gateway-api-spec.md` |
| Runbook | Policy Release and Rollbook Runbook | `docs/runbooks/cluster-policy-release-and-rollback.md` |
