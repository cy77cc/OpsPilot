## MODIFIED Requirements

### Requirement: Agentic Multi-Expert Architecture

AI 助手 MUST 使用四层 Agentic 架构：Planner → Executor → Expert → Summarizer，通过 Eino ADK 的 `Transfer` 与 `AgentAsTool` 特性实现多领域专家协作。

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    四层 Agentic Multi-Expert 架构                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Planner Agent ──► Executor Agent ──► Domain Experts ──► Summarizer Agent │
│                                                                             │
│   - Planner: 意图理解 + 资源解析 + 权限检查 + 计划制定                        │
│   - Executor: 计划执行 + 依赖调度 + 审批恢复 + 结果收集                        │
│   - Experts: 领域任务执行 (HostOps/K8s/Service/Delivery/Observability)       │
│   - Summarizer: 结果汇总 + 完整性判断                                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Acceptance Criteria:**
- [ ] Planner Agent 具备 resolve_service/resolve_cluster/resolve_host/check_permission 工具
- [ ] Planner Agent MUST 输出结构化 `PlannerDecision`，覆盖 `clarify/reject/plan/direct_reply`
- [ ] Executor Agent 支持依赖分析、并行/串行调度、审批恢复和 step 级状态流转
- [ ] 至少实现 HostOpsExpert, K8sExpert, ServiceExpert, DeliveryExpert, ObservabilityExpert 五个执行专家
- [ ] Summarizer Agent 支持结果汇总和补充调查判断
- [ ] 支持流式输出 (SSE)
- [ ] 支持用户澄清交互

#### Scenario: Planner resolves resources and creates plan
- **GIVEN** 用户请求 "排查 payment-api 响应慢"
- **WHEN** Planner 处理请求
- **THEN** Planner MUST 调用 resolve_service 获取 service_id
- **AND** Planner MUST 调用 check_permission 验证权限
- **AND** Planner MUST 输出包含已解析资源的结构化执行计划

#### Scenario: Planner requests clarification for ambiguous input
- **GIVEN** 用户请求 "检查 payment 服务"
- **WHEN** resolve_service 返回多个匹配结果
- **THEN** Planner MUST 直接回复用户请求澄清
- **AND** 不进入 Executor 执行阶段

#### Scenario: Executor schedules experts based on dependencies
- **GIVEN** Planner 输出包含依赖关系的计划
- **WHEN** Executor 执行计划
- **THEN** 无依赖的步骤 MUST 并行执行
- **AND** 有依赖的步骤 MUST 等待依赖完成后执行
- **AND** 失败的步骤 MUST 重试最多 3 次
- **AND** 进入审批的步骤 MUST 持久化为 `waiting_approval`

#### Scenario: Summarizer determines investigation completeness
- **GIVEN** 所有专家执行完成
- **WHEN** Summarizer 汇总结果
- **THEN** Summarizer MUST 判断是否需要补充调查
- **AND** 如需补充且未超过最大迭代次数，MUST 回到 Planner 重新规划

#### Scenario: Approval resumes a single blocked step
- **GIVEN** 某个 mutating step 因审批进入 `waiting_approval`
- **WHEN** 用户完成审批并调用恢复流程
- **THEN** Executor MUST 仅恢复对应 `step_id`
- **AND** MUST NOT 重跑已经 `completed` 的 steps

### Requirement: Planner Structured Output

Planner MUST 通过结构化协议向 Orchestrator 输出决策，而不是依赖自然语言正文解析。

**Acceptance Criteria:**
- [ ] Planner MUST 输出 `PlannerDecision`，类型至少覆盖 `clarify/reject/plan/direct_reply`
- [ ] `plan` 类型 MUST 包含 `ExecutionPlan`
- [ ] `ExecutionPlan` MUST 包含稳定 `plan_id`
- [ ] `PlanStep` MUST 包含稳定 `step_id`
- [ ] `PlanStep` MUST 显式声明 `expert/intent/task`
- [ ] Planner 在 `clarify` 场景 MUST 给出候选项或明确补充要求

#### Scenario: Planner emits structured clarify decision
- **GIVEN** `resolve_service` 返回多个高相似候选
- **WHEN** Planner 无法唯一确认目标资源
- **THEN** Planner MUST 输出 `type=clarify`
- **AND** MUST NOT 输出可直接执行的 `ExecutionPlan`

### Requirement: Planner Tools

Planner Agent MUST 提供以下工具用于资源解析和权限检查：

| 工具 | 说明 |
|------|------|
| `resolve_service` | 根据服务名称/关键词解析服务ID |
| `resolve_cluster` | 根据集群名称/环境解析集群ID |
| `resolve_host` | 根据主机名/IP解析主机ID |
| `check_permission` | 检查用户对资源的操作权限 |
| `get_user_context` | 获取用户当前上下文信息 |

**Acceptance Criteria:**
- [ ] resolve_service 支持精确匹配和模糊匹配
- [ ] resolve_cluster 支持按名称和环境过滤
- [ ] resolve_host 支持主机名、IP、关键词匹配
- [ ] check_permission 返回权限状态和原因
- [ ] get_user_context 返回当前页面和选中资源
- [ ] `resolve_*` MUST 复用 `*_list_inventory` 作为底层候选来源，而不是重复实现底层查询
- [ ] `resolve_*` MUST 返回 `exact/ambiguous/missing` 三类结构化状态
- [ ] `*_list_inventory`、`permission_check`、`user_list`、`role_list` SHOULD 归入 Planner support tools

#### Scenario: Resolve uses inventory as source of truth
- **GIVEN** `service_list_inventory` 可返回候选服务列表
- **WHEN** Planner 调用 `resolve_service`
- **THEN** `resolve_service` MUST 基于 inventory 候选做评分与消歧
- **AND** MUST 返回结构化 `ResolveResult`

### Requirement: Domain Expert Isolation

每个领域专家 MUST 只持有本领域的工具，避免工具集膨胀。

| 专家 | 工具数量 | 示例工具 |
|------|---------|---------|
| HostOpsExpert | ~11 | os_get_*, host_* |
| K8sExpert | ~6 | k8s_* |
| ServiceExpert | ~13 | service_*, deployment_*, credential_*, config_* |
| DeliveryExpert | ~6 | cicd_*, job_* |
| ObservabilityExpert | ~7 | monitor_*, topology_*, audit_* |

**Acceptance Criteria:**
- [ ] 每个专家的工具集 MUST 与其领域职责匹配
- [ ] 领域执行工具 MUST 按专家隔离，公共基础能力 MAY 以 helper/middleware/library 形式共享
- [ ] 专家 MUST 通过 AgentAsTool 被 Executor 调用
- [ ] `*_list_inventory`、`permission_check`、`user_list`、`role_list` SHOULD 优先作为 Planner 支撑工具而不是 execution expert 主工具
- [ ] `service_deploy`、`host_batch` SHOULD 仅作为迁移兼容入口，MUST NOT 作为长期标准 expert tool contract

#### Scenario: Planner support tools are not mounted on experts
- **GIVEN** `cluster_list_inventory` 用于资源解析
- **WHEN** 新架构装配 experts
- **THEN** `cluster_list_inventory` SHOULD 挂载到 Planner support tools
- **AND** SHOULD NOT 作为 `K8sExpert` 的主工具集暴露

### Requirement: Tool Risk And Approval Policy

所有暴露给 Expert 的工具 MUST 带有结构化风险元数据，并由运行时决定 review/approval，而不是完全依赖 prompt。

**Acceptance Criteria:**
- [ ] 每个 expert tool MUST 声明 `mode` 和 `risk`
- [ ] `readonly` + `low` 工具 MUST 可直接执行
- [ ] `medium` 风险工具 MUST 进入 review/edit 或按策略触发审批
- [ ] `high` 风险工具 MUST 进入审批流程
- [ ] Planner MUST NOT 持有 `mutating` 工具
- [ ] `mutating` 工具 MUST 显式声明幂等性与副作用策略

#### Scenario: High risk mutating tool requires approval
- **GIVEN** Executor 调用 `service_deploy_apply`
- **WHEN** 运行时识别该工具为 `mutating/high`
- **THEN** 该 step MUST 进入审批
- **AND** MUST 在审批通过后才允许继续执行

### Requirement: Executor State And Recovery

Executor MUST 使用确定性的 step 状态机和持久化执行状态，支持审批与中断恢复。

**Acceptance Criteria:**
- [ ] step 状态至少覆盖 `pending/ready/running/waiting_approval/retrying/blocked/failed/completed/cancelled`
- [ ] `waiting_approval` 状态 MUST 持久化到 `ExecutionState`
- [ ] `Resume(...)` MUST 只恢复一个被阻断的 step
- [ ] `blocked` step MUST NOT 自动恢复，除非发生重新规划
- [ ] 总超时或用户取消后，所有未完成 step MUST 进入 `cancelled`

### Requirement: Execution Control

执行过程 MUST 受以下参数控制：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `max_retry` | 3 | 单个步骤最大重试次数 |
| `max_iterations` | 5 | Planner → Summarizer 最大循环次数 |
| `expert_timeout` | 60s | 单个专家超时 |
| `total_timeout` | 300s | 总超时 |

**Acceptance Criteria:**
- [ ] 超过 max_retry 后 MUST 返回友好的错误消息
- [ ] 超过 max_iterations 后 MUST 终止并提示用户
- [ ] 超时后 MUST 取消执行并返回错误

### Requirement: Runtime Context And Execution State

Gateway、Orchestrator、Planner、Executor 之间 MUST 使用标准化运行时上下文和执行状态，而不是透传原始前端 payload。

**Acceptance Criteria:**
- [ ] Gateway MUST 生成标准化 `RuntimeContext`
- [ ] Orchestrator MUST 负责初始化和写入 `ExecutionState`
- [ ] Planner MUST 只读 `RuntimeContext` 和 `SessionValues`
- [ ] Executor MUST 负责更新 `StepState/StepResult/PendingApproval`
- [ ] `Run(...)` 与 `Resume(...)` MUST 共享同一 `trace_id`

### Requirement: SSE Event Stream

SSE 事件流 MUST 支持以下事件类型：

| 事件 | 说明 |
|------|------|
| `meta` | 会话元信息 |
| `delta` | 内容片段 |
| `planner_state` | 规划阶段状态 |
| `plan_created` | 计划创建 |
| `step_start` | 步骤开始 |
| `step_result` | 步骤结果 |
| `expert_progress` | 专家执行进度 |
| `tool_call` | 工具调用 |
| `tool_result` | 工具结果 |
| `approval_required` | 需要审批 |
| `done` | 执行完成 |
| `error` | 错误 |

**Acceptance Criteria:**
- [ ] 新事件 MUST 兼容现有前端
- [ ] 所有关键事件 MUST 携带 `session_id/plan_id/trace_id/iteration/timestamp`
- [ ] `plan_created` 事件 MUST 包含完整的计划信息
- [ ] `step_result` 事件 MUST 包含 `step_id` 与结果状态
- [ ] `expert_progress` 事件 MUST 包含专家名称和状态
- [ ] `approval_required` 事件 MUST 包含 `step_id`
- [ ] 同一 `step_id` 的事件顺序 MUST 可判定
