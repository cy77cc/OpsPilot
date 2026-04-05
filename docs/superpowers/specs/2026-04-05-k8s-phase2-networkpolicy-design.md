# K8s Phase 2 设计：多 CNI NetworkPolicy 可视化编排

## 1. 背景与目标

Phase 1 已完成集群详情页“可操作闭环”。Phase 2 聚焦网络与策略治理，目标是交付“多 CNI NetworkPolicy 可视化编排”能力，并达到“读写 + 仿真校验”的生产可用门槛。

本阶段明确约束：

- 流量治理方向：`Gateway API` 为主，`Ingress` 仅兼容过渡。
- 网络策略方向：`Cilium` 为主，兼容 `Calico` 与 `Flannel`。
- 发布策略：安全与效率均衡（双指标达标）。
- 架构策略：先中心化控制，预留后续集群侧执行器下沉接口。

## 2. 范围与非目标

### 2.1 本阶段范围

1. 多 CNI 策略可视化：命名空间、工作负载与策略关联拓扑。
2. 策略读写闭环：创建、编辑、发布、回滚、审计追踪。
3. 发布前仿真：冲突检测、影响面分析、风险评分。
4. CNI 适配层：Cilium/Calico/Flannel 的能力矩阵与翻译校验。
5. 与现有治理体系整合：RBAC、审批、审计、操作中心。

### 2.2 非目标

1. 不在本阶段交付 Service Mesh 深度治理（灰度/熔断/限流）。
2. 不在本阶段交付完整 L7 流量治理编排。
3. 不在本阶段交付跨云全自动策略迁移。

## 3. 总体架构

采用“统一策略模型 + 多插件适配层 + 中心化仿真引擎”的方案。

分层如下：

1. `Policy API`：对外提供策略与发布能力接口。
2. `Policy Core`：统一 DSL、规则归一化、能力标签判定。
3. `CNI Adapter`：Cilium/Calico/Flannel 翻译与兼容校验。
4. `Simulation Engine`：冲突检测、影响面与风险评估。
5. `Governance`：RBAC、审批、审计与操作链路追踪。
6. `UI`：策略可视化、编排、仿真结果对比、发布/回滚操作。

控制面演进：

- 当前：中心服务执行仿真与发布。
- 预留：后续引入集群侧执行器时复用统一执行接口，不破坏上层产品体验。

## 4. 组件设计与职责

### 4.1 policy-definition-service

- 负责统一策略 DSL 的草稿、版本、生效态管理。
- 提供模板能力，确保跨 CNI 策略描述一致。

### 4.2 policy-simulation-service

- 对比 `base_version` 与 `candidate_version`。
- 输出阻断项、告警项、影响面摘要与风险等级。

### 4.3 policy-adapter-service

- `cilium-adapter`：支持 Cilium 原生扩展语义映射。
- `calico-adapter`：支持原生 + Calico 增强语义映射。
- `flannel-adapter`：识别能力缺口并生成阻断/建议。

### 4.4 policy-release-service

- 管理发布流水：仿真 -> 审批 -> 下发 -> 回滚。
- 统一结果状态并关联审计记录与操作中心链接。

### 4.5 policy-visualization-ui

- 提供拓扑视图、策略编辑器、仿真 diff、风险说明与发布界面。

### 4.6 policy-observability-bridge

- 汇聚策略命中、拒绝流量摘要与关键告警，支持按 `release_id` 追踪。

## 5. 关键数据流与状态机

### 5.1 策略变更主流程

1. 用户编辑策略草稿。
2. 执行仿真（强制门禁）。
3. 仿真通过后发起发布申请。
4. 审批通过后执行下发。
5. 记录审计并回链操作中心。
6. 成功生效或失败回滚。

### 5.2 仿真流程

输入：

- `base_version`
- `candidate_version`
- 目标集群 CNI 能力信息

输出：

- `blocking_issues`
- `warnings`
- `impact_summary`
- `risk_score`

### 5.3 发布状态机

- `draft`
- `simulation_passed`
- `approval_required`
- `applying`
- `applied`
- `simulation_failed`
- `approval_rejected`
- `apply_failed`
- `rollback_applied`

## 6. 跨 CNI 支持策略

1. `Cilium`：Phase 2 主验收线，完整支持读写与仿真校验。
2. `Calico`：支持读写与仿真，增强能力按兼容矩阵落地。
3. `Flannel`：默认仅支持可视化；若未接入策略引擎，策略发布必须阻断并给出治理建议。

该策略用于避免“策略可下发但实际不生效”的假闭环。

## 7. 错误处理与风险控制

### 7.1 阻断级错误（必须停止发布）

1. 高风险冲突将导致关键命名空间流量误阻断。
2. 语义转译后高风险不等价。
3. 审批未通过或审批令牌无效。
4. 下发校验失败。

### 7.2 告警级问题（允许继续但需确认）

1. CNI 能力降级。
2. 影响面扩大但在白名单范围内。
3. 非关键命名空间覆盖变化。

### 7.3 回滚机制

1. 每次发布生成 `release_id` 与 `previous_stable_version`。
2. 发布失败或健康检查异常支持一键回滚。
3. 回滚全链路必须审计可追溯。

### 7.4 护栏

1. 禁止绕过仿真直接发布。
2. 高风险命名空间启用二次审批。
3. Flannel 能力缺口按配置执行强阻断。

## 8. 测试与验收标准

### 8.1 测试分层

1. 单元测试：DSL、翻译、冲突检测、评分规则。
2. 集成测试：三类 CNI 发布流水。
3. 端到端测试：策略编辑到操作中心追踪闭环。
4. 兼容测试：Gateway API 资源识别与关联展示。

### 8.2 验收门槛（均衡）

1. 安全正确性：
- 高风险冲突 0 漏拦截。
- 跨 CNI 语义降级必须显式提示。

2. 交付效率：
- 仿真、审批、发布时延均可观测并可追溯。

3. 闭环能力：
- 每次发布均有 `release_id`、审计记录、可回滚版本。
- 操作中心可按策略/集群/操作者/时间追踪。

### 8.3 CNI 分级验收

1. Cilium：读写+仿真全通过。
2. Calico：读写+仿真通过，增强项按矩阵验收。
3. Flannel：只读可视化通过；无策略引擎时发布阻断通过。

## 9. 里程碑建议（Phase 2 内）

1. M1：统一 DSL 与 Cilium 适配 + 基础仿真引擎。
2. M2：Calico 适配 + 发布流水与审批审计打通。
3. M3：Flannel 能力缺口阻断 + 可视化与操作中心联动。
4. M4：回归、验收、灰度上线准备。

## 10. 关键参考（官方文档）

1. Kubernetes NetworkPolicy：
   - https://kubernetes.io/docs/concepts/services-networking/network-policies/
2. Gateway API：
   - https://gateway-api.sigs.k8s.io/
3. Cilium Policy：
   - https://docs.cilium.io/en/stable/network/kubernetes/policy.html
4. Calico Policy：
   - https://docs.tigera.io/calico/latest/reference/resources/networkpolicy
   - https://docs.tigera.io/calico/latest/reference/resources/globalnetworkpolicy
5. Flannel 与策略引擎协作背景：
   - https://docs.rke2.io/networking/basic_network_options
   - https://docs.tigera.io/calico/latest/getting-started/kubernetes/flannel/install-for-flannel
