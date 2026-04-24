# Agent-First AIOps 设计（1000+ 节点，SSH 弱依赖）

## 1. 背景与目标

当前主机数据采集与 AI 排障都高度依赖 SSH，存在以下问题：

- 连接稳定性差：跨区域、跨网络环境下 SSH 连接易抖动，批量任务失败率高
- 可扩展性不足：1000+ 节点规模下，基于 SSH 的拉取与远程执行难以稳定扩展
- 安全与审计压力大：AI 直接或间接走 SSH 执行命令，最小权限、行为约束与审计闭环困难
- 执行模型不统一：采集、诊断、修复分散，缺少统一的策略裁决与状态机

本设计目标：

- 采集链路从 SSH 主导升级为 Agent 主导（SSH 仅应急）
- AI 排障从 SSH 命令驱动升级为结构化动作驱动（Policy + Orchestrator）
- 支撑 1000+ 节点、多区域、多租户的可用性、可治理性和可审计性

## 2. 约束与原则

已确认约束：

- SSH 定位：弱依赖（主链路去 SSH，保留 break-glass）
- 部署方式：可在被管服务器安装轻量 Agent
- AI 动作策略：分级放开（低风险自动，高风险审批）
- 目标规模：1000+ 节点，跨区域扩展

设计原则：

- Agent-First：默认所有观测与动作都走 Agent 通道
- Policy-First：所有动作先裁决再执行，AI 不直连执行面
- Evidence-First：诊断与动作都要可回放证据
- 安全优先：策略系统或关键依赖异常时默认 deny
- 可演进：先逻辑拆分、后物理拆分，分阶段替换 SSH 主链路

## 3. 总体架构

### 3.1 五层架构

1. Node Agent 层  
每台主机部署 systemd 常驻 Agent，采集 metrics/log/process/inventory，并执行受控动作。

2. Ingestion & Stream 层  
Agent 数据通过 mTLS 主动回连进入 gateway，再进入消息总线，分流至时序、日志、事件存储。

3. Control Plane 层  
负责节点注册、证书生命周期、配置下发、动作编排与任务状态管理。

4. AI Diagnosis & Remediation 层  
AI 做证据驱动诊断，输出结构化 Action Plan，经策略裁决后执行。

5. Break-Glass 层（SSH）  
仅在 Agent 失联或能力缺口时启用，要求 JIT 授权、全审计、超时回收。

### 3.2 高层数据路径

- 观测路径：Agent -> Gateway -> Stream -> Telemetry Stores -> AI/UI
- 动作路径：AI Runtime -> Action Orchestrator -> Policy Engine -> Agent/SSH Connector -> Audit Ledger
- 审计路径：所有请求、判定、审批、执行与验证结果写入统一审计账本

## 4. 核心组件边界

### 4.1 agent-gateway

职责：

- 处理 Agent 长连接、mTLS 双向认证、心跳与命令下发通道
- 做连接管理与速率控制，不承载业务策略

接口：

- `RegisterAgent`
- `ReportHeartbeat`
- `PushTelemetry`
- `ReceiveAction`

### 4.2 asset-registry

职责：

- 管理节点身份、租户归属、环境、分组、标签与能力清单
- 提供统一资产事实给 AI 与编排层

接口：

- `GetNodeProfile(node_id)`
- `ListNodesByScope(tenant, env, labels)`
- `UpdateNodeCapabilities(node_id, capabilities)`

### 4.3 telemetry-pipeline

职责：

- 标准化 metrics/log/events，分区路由并落库
- 管理重试、背压、死信与幂等写入

接口：

- `IngestTelemetry(batch)`
- `ReplayDeadLetter(topic, partition, offset_range)`

### 4.4 action-orchestrator

职责：

- 统一动作入口与状态机
- 执行 `Plan -> Policy Check -> Dispatch -> Verify -> Rollback/Complete`

接口：

- `SubmitActionPlan(plan)`
- `ApproveAction(action_id, approver)`
- `QueryActionStatus(action_id)`
- `CancelAction(action_id)`

### 4.5 policy-engine

职责：

- 风险分级（L1/L2/L3）、审批要求、变更冻结窗、租户隔离规则
- 给 AI 与 Orchestrator 提供统一裁决，避免双轨判定

接口：

- `Evaluate(plan, context) -> {decision, reason, required_approvals}`

### 4.6 ai-runtime

职责：

- 聚合证据、生成诊断与结构化动作建议
- 不持有主机 shell 直连权限

接口：

- `Diagnose(incident_context) -> diagnosis_report`
- `ProposeActions(diagnosis_report) -> action_plan`

### 4.7 audit-ledger

职责：

- 记录 AI 版本、输入证据、策略结论、审批链与执行结果
- 为合规与复盘提供不可抵赖轨迹

接口：

- `AppendAuditRecord(record)`
- `QueryAuditTrail(scope, time_range)`

## 5. 端到端流程

### 5.1 采集主链路

1. Agent 周期采集与事件触发采集
2. 数据经 Gateway 入总线
3. Pipeline 标准化后写入时序/日志/事件存储
4. AI 与运维界面统一读取观测事实源

### 5.2 AI 排障主链路

1. 触发源：告警、工单或人工提问
2. AI 拉取指标、日志、拓扑与变更记录进行诊断
3. 生成结构化 Action Plan（禁止原始 shell 文本直传执行）
4. Orchestrator 调用 Policy：
   - L1：自动执行
   - L2/L3：审批后执行
5. 执行后自动验证 SLI，失败触发回滚或升级人工
6. 输出复盘摘要并归档审计

### 5.3 SSH 兜底链路

触发条件：

- Agent 失联
- Agent 不具备目标动作能力
- 灾难恢复场景需紧急介入

控制要求：

- 必须由 Orchestrator 间接调用 SSH Connector
- JIT 凭证、命令白名单、会话审计、TTL 自动销权
- 产生“为何走 SSH”的审计原因码

## 6. 异常处理与可靠性设计

### 6.1 节点连接状态机

- `suspect -> unreachable -> lost` 三段式状态，不因瞬时抖动立即判死
- 区域批量失联时触发保护模式：暂停高风险自动动作

### 6.2 消息可靠性

- 采用至少一次投递
- 消费幂等键：`node_id + action_id + seq`
- 落库失败进入 DLQ，提供自动重试与人工重放入口

### 6.3 动作状态机

状态：

- `pending`
- `approved`
- `running`
- `verifying`
- `succeeded`
- `failed`
- `rolled_back`

规则：

- 超时、冲突、前置条件失败时 fail-fast
- 高风险动作无回滚策略则禁止自动执行

### 6.4 控制面高可用

- gateway/orchestrator/policy/audit 无状态化，多副本部署
- 元数据存储做主从容灾与定期恢复演练
- policy 不可用时默认 deny，避免越权执行

### 6.5 多租户隔离

- 数据按 tenant/env/region 分区隔离
- 审批、策略、审计在租户命名空间内独立
- AI 检索与动作作用域必须携带租户上下文并强校验

## 7. 安全与治理

### 7.1 身份与信任

- Agent 身份基于短周期证书与轮换机制
- 节点注册采用受控引导令牌并绑定租户与环境

### 7.2 权限模型

- AI 只拥有“提出动作计划”权限，不拥有直接执行权限
- 执行权限归 Orchestrator，且需通过 Policy 判定

### 7.3 审批模型

- L1 自动
- L2 单人审批
- L3 双人审批或指定角色审批
- 支持变更冻结窗与紧急豁免流程（豁免必须审计）

## 8. 测试与验收

### 8.1 测试分层

- 单元测试：Policy 判定、动作状态机、幂等消费、审批分支
- 集成测试：注册、证书轮换、动作下发、回执、DLQ 重放
- 混沌测试：消息堆积、区域抖动、批量失联、组件降级
- 安全测试：越权、跨租户、重放、过期证书、SSH 滥用

### 8.2 验收指标

- 采集可用性 >= 99.9%
- L1 动作成功率 >= 98%
- 自动动作误触发率 <= 0.1%
- SSH 使用占比在迁移后 3 个月内降至 < 10%

## 9. 分阶段迁移计划

### Phase 0（2-4 周）：控制面与审计基础

- 上线 agent-gateway、asset-registry、audit-ledger 基础能力
- SSH 仍为主链路，新增审计统一口径

### Phase 1（4-6 周）：采集链路切主

- Agent 接管 metrics/log 主采集
- SSH 采集降级为兜底

### Phase 2（4-8 周）：AI 受控动作闭环

- AI 输出结构化 Action Plan
- L1 自动，L2/L3 审批
- 引入执行后 SLI 验证与回滚

### Phase 3（持续）：规模化与治理优化

- 多区域扩展、容量治理、成本优化
- 策略迭代与误报误触发收敛
- SSH break-glass 常态化稽核

## 10. 非目标（当前阶段不做）

- 不做“完全去 SSH”的硬切换
- 不做 AI 全自动高风险动作（无审批）
- 不在本阶段引入跨云统一 CMDB 大改造

## 11. 风险与缓解

- Agent 部署推进慢  
  缓解：先覆盖核心业务集群，按环境分批灰度

- 策略误配导致动作阻塞或误放行  
  缓解：策略版本化、灰度发布、回滚与审计对账

- 消息链路拥塞影响实时性  
  缓解：分区扩容、背压策略、优先级队列与降采样

- 团队沿用 SSH 运维习惯难改  
  缓解：提供可观测收益对比看板，将 SSH 使用率纳入治理指标

## 12. 决策结论

- 采集：从 SSH 拉取转为 Agent 主动上报
- 排障：从 AI+SSH 命令执行转为 AI+结构化动作编排
- SSH：保留为应急通道，但纳入强策略与全审计

该设计可在不打断现网运维的前提下，逐步把 SSH 从主路径降为兜底路径，并为 1000+ 节点规模建立统一、可审计、可扩展的 AIOps 执行基础。
