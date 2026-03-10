## MODIFIED Requirements

### Requirement: Agentic Multi-Expert Orchestration Host

AI 助手 MUST 通过稳定的 AI Orchestrator Host 承载完整执行链路：

```text
Gateway / API
    -> AI Orchestrator Host
        -> Rewrite
        -> Planner
        -> Executor Runtime
            -> Expert Agents as Tools
        -> Summarizer
```

约束：

- Gateway 只负责 transport / auth / session shell
- `internal/ai` MUST 作为 AI 编排主边界
- `Executor` MUST 作为 runtime 负责确定性执行，而不是再次退化为大一统自治 Agent

**Acceptance Criteria:**
- [ ] AI Orchestrator Host MUST 成为唯一稳定入口
- [ ] Rewrite / Planner / Executor Runtime / Summarizer MUST 有明确宿主边界
- [ ] Experts MUST 通过 Agent Tool 接入，而不是把全部平台工具重新挂到 Executor
- [ ] Gateway MUST NOT 承担 AI 编排语义

#### Scenario: gateway delegates execution to AI Orchestrator Host
- **GIVEN** 用户请求 `/api/v1/ai/chat`
- **WHEN** Gateway 接收请求
- **THEN** Gateway MUST 将标准化请求委托给 `internal/ai` 中的 Orchestrator Host
- **AND** Gateway MUST NOT 在 handler 层重写编排语义

### Requirement: Rewrite Stage For Colloquial Input

系统 MUST 在正式规划前增加 `Rewrite` 阶段，用于将用户口语化输入改写为稳定的任务表达。

`Rewrite` 输出 MUST 采用半结构化协议：

- 结构化字段用于下游执行
- `narrative` 用于解释和消歧

**Acceptance Criteria:**
- [ ] Rewrite MUST 输出 `normalized_goal`
- [ ] Rewrite MUST 输出 `operation_mode`
- [ ] Rewrite MAY 输出 `resource_hints`
- [ ] Rewrite MAY 输出 `domain_hints`
- [ ] Rewrite MUST 输出自然语言 `narrative`
- [ ] Rewrite MUST NOT 负责最终资源解析
- [ ] Rewrite MUST NOT 持有 mutating tool

#### Scenario: colloquial request is normalized before planning
- **GIVEN** 用户请求 “帮我看看 payment-api 最近是不是有点慢，顺便查下是不是刚发版”
- **WHEN** Rewrite 阶段处理该输入
- **THEN** 系统 MUST 输出归一化后的调查目标
- **AND** MUST 提取 `service_name=payment-api` 作为资源 hint
- **AND** MUST 将任务模式标记为 `investigate`
- **AND** MUST 输出解释该改写含义的 `narrative`

### Requirement: Planner MUST Use Semi-Structured Output

Planner MUST 通过半结构化协议向 Orchestrator 输出决策，而不是依赖自然语言正文解析。

Planner 输出中的结构化字段用于运行时执行，`narrative` 用于解释和消歧。

**Acceptance Criteria:**
- [ ] Planner MUST 输出 `PlannerDecision`
- [ ] `PlannerDecision` MUST 覆盖 `clarify/reject/direct_reply/plan`
- [ ] `plan` 类型 MUST 包含 `ExecutionPlan`
- [ ] `ExecutionPlan` MUST 包含稳定 `plan_id`
- [ ] `PlanStep` MUST 包含稳定 `step_id`
- [ ] `PlanStep` MUST 显式声明 `expert/intent/task/mode/risk`
- [ ] `ExecutionPlan` MUST 包含阶段级 `narrative`
- [ ] `clarify/reject/direct_reply` MUST 包含 `narrative`

#### Scenario: planner emits structured plan with narrative
- **GIVEN** Rewrite 已将用户目标规整为调查任务
- **WHEN** Planner 完成资源解析与规划
- **THEN** Planner MUST 输出结构化 `ExecutionPlan`
- **AND** MUST 为计划补充自然语言 `narrative`
- **AND** MUST 为每个 step 提供机器可消费字段
- **AND** MUST NOT 依赖自然语言正文中的隐式语义驱动执行

#### Scenario: planner requests clarification for ambiguous resource
- **GIVEN** `resolve_service` 返回多个高相似候选
- **WHEN** Planner 无法唯一确定目标资源
- **THEN** Planner MUST 输出 `type=clarify`
- **AND** MUST 提供澄清消息或候选项
- **AND** MUST NOT 进入执行阶段

### Requirement: Planner Support Tools

Planner MUST 提供用于资源解析和权限预检查的 support tools。

| 工具 | 说明 |
|------|------|
| `resolve_service` | 根据服务名称/关键词解析服务 ID |
| `resolve_cluster` | 根据集群名称/环境解析集群 ID |
| `resolve_host` | 根据主机名/IP解析主机 ID |
| `check_permission` | 检查用户对资源的操作权限 |
| `get_user_context` | 获取标准化运行时上下文 |

**Acceptance Criteria:**
- [ ] `resolve_*` MUST 复用 inventory 候选来源
- [ ] `resolve_*` MUST 返回 `exact/ambiguous/missing` 结构化状态
- [ ] `check_permission` MUST 仅做预检查
- [ ] `get_user_context` MUST 返回标准化上下文，而不是透传原始前端 payload
- [ ] `*_list_inventory`、`permission_check`、`user_list`、`role_list` SHOULD 作为 Planner support tools

#### Scenario: planner uses inventory-backed resolve
- **GIVEN** `service_list_inventory` 可提供服务候选列表
- **WHEN** Planner 调用 `resolve_service`
- **THEN** `resolve_service` MUST 基于 inventory 结果评分与消歧
- **AND** MUST 返回结构化 resolve 结果

### Requirement: Expert Agents MUST Be Isolated And Toolized

每个领域专家 MUST 只持有本领域工具，并通过 Agent Tool 接入执行层。

专家包括：

- HostOpsExpert
- K8sExpert
- ServiceExpert
- DeliveryExpert
- ObservabilityExpert

**Acceptance Criteria:**
- [ ] 每个专家的工具集 MUST 与领域职责匹配
- [ ] 领域执行工具 MUST 按专家隔离
- [ ] 专家 MUST 通过 Agent Tool 暴露给 Executor Runtime
- [ ] Planner support tools SHOULD NOT 暴露在 Experts 主工具集中
- [ ] `service_deploy`、`host_batch` SHOULD 仅保留为迁移兼容入口

#### Scenario: executor invokes expert through agent tool
- **GIVEN** Planner 产出一个 `expert=service` 的 step
- **WHEN** Executor Runtime 调度该 step
- **THEN** Executor MUST 通过 ServiceExpert Agent Tool 调用该专家
- **AND** MUST NOT 直接重新挂载全部 service 工具到 Executor

### Requirement: Executor Runtime MUST Own Deterministic Execution

Executor Runtime MUST 负责确定性执行，而不是依赖模型隐式完成调度。

职责包括：

- step 依赖分析
- 并行/串行调度
- 审批等待
- Resume 恢复
- 超时与重试
- step 状态流转

**Acceptance Criteria:**
- [ ] step 状态至少覆盖 `pending/ready/running/waiting_approval/completed/failed/blocked/cancelled`
- [ ] 无依赖 steps MUST 可并行执行
- [ ] 有依赖 steps MUST 等待上游完成
- [ ] `waiting_approval` MUST 持久化到 `ExecutionState`
- [ ] `blocked` step MUST NOT 自动恢复
- [ ] `Resume(...)` MUST 只恢复一个被阻断 step

#### Scenario: executor schedules plan deterministically
- **GIVEN** Planner 输出包含依赖关系的 `ExecutionPlan`
- **WHEN** Executor Runtime 执行该计划
- **THEN** Executor MUST 根据依赖关系调度 step
- **AND** 无依赖 steps MUST 可并行运行
- **AND** 上游失败的下游 steps MUST 进入 `blocked`

#### Scenario: approval resumes one step by plan-step identity
- **GIVEN** 某个 mutating step 进入 `waiting_approval`
- **WHEN** 用户提交 `session_id + plan_id + step_id` 的恢复请求
- **THEN** Executor MUST 仅恢复对应 step
- **AND** MUST NOT 重跑已完成步骤

### Requirement: Resume MUST Be Idempotent

恢复请求 MUST 具备幂等语义，避免重复批准或重复恢复导致副作用重复执行。

**Acceptance Criteria:**
- [ ] 同一 `plan_id + step_id + approval_decision` 的重复恢复请求 MUST 不得重复执行写操作
- [ ] 对 mutating step，运行时 MUST 支持去重键、等价 request_id 或同等级别的重复保护
- [ ] 前端重复提交、网络重试或服务重启后重放 MUST 不得导致重复 apply

#### Scenario: duplicate approval response does not re-run write action
- **GIVEN** 某个高风险写步骤已经因审批通过而恢复执行
- **WHEN** 同一批准请求再次到达
- **THEN** 运行时 MUST 识别该请求为重复恢复
- **AND** MUST NOT 再次执行该写步骤

### Requirement: Tool Risk And Approval Policy

所有由 Experts 暴露的执行工具 MUST 带有结构化风险元数据，并由运行时决定审批策略。

**Acceptance Criteria:**
- [ ] 每个 expert tool MUST 声明 `mode`
- [ ] 每个 expert tool MUST 声明 `risk`
- [ ] `readonly + low` 工具 MUST 可直接执行
- [ ] `medium` 风险工具 MUST 支持 review/edit 或按策略审批
- [ ] `high` 风险工具 MUST 进入审批流程
- [ ] Planner MUST NOT 持有 mutating tool

#### Scenario: high-risk mutating tool requires approval
- **GIVEN** Executor Runtime 调用 `service_deploy_apply`
- **WHEN** 运行时判定该工具为 `mutating/high`
- **THEN** 对应 step MUST 进入 `waiting_approval`
- **AND** MUST 在审批通过后才允许继续执行

### Requirement: Runtime Context MUST Be Standardized

Gateway、Orchestrator、Rewrite、Planner、Executor 之间 MUST 使用标准化 `RuntimeContext`，而不是透传原始前端 payload。

**Acceptance Criteria:**
- [ ] Gateway MUST 生成标准化 `RuntimeContext`
- [ ] Rewrite MUST 消费标准化上下文
- [ ] Planner MUST 只读 `RuntimeContext`
- [ ] Executor MUST 使用标准化上下文与 `ExecutionState`
- [ ] `Run(...)` 与 `Resume(...)` MUST 共享同一 `trace_id`

#### Scenario: runtime context is normalized before entering rewrite
- **GIVEN** 前端发送页面场景和当前选中资源
- **WHEN** Gateway 构造运行请求
- **THEN** Gateway MUST 将其规范化为 `RuntimeContext`
- **AND** MUST NOT 将原始 payload 直接透传到下游所有阶段

### Requirement: ExecutionState MUST Be Durable And Recoverable

`ExecutionState` MUST 具备明确的持久化真源和恢复边界，以支持审批等待、重启恢复和事件一致性。

**Acceptance Criteria:**
- [ ] 系统 MUST 明确 `ExecutionState` 的持久化介质
- [ ] step 状态变化、进入 `waiting_approval`、step 完成时 MUST 持久化关键状态
- [ ] 服务重启后，处于 `waiting_approval` 的执行 MUST 可恢复
- [ ] 事件输出与 `ExecutionState` MUST 保持可对齐关系

#### Scenario: waiting approval state survives process restart
- **GIVEN** 某个步骤处于 `waiting_approval`
- **WHEN** AI 服务发生重启
- **THEN** 系统 MUST 能从持久化状态恢复该执行上下文
- **AND** 用户后续仍可按 `plan_id + step_id` 完成恢复

### Requirement: ThoughtChain-Oriented Event Stream

系统 MUST 输出面向前端 ThoughtChain 的高层事件语义，而不是仅输出底层工具日志。

推荐高层事件包括：

- `meta`
- `rewrite_result`
- `planner_state`
- `plan_created`
- `step_update`
- `approval_required`
- `clarify_required`
- `replan_started`
- `delta`
- `summary`
- `done`
- `error`

**Acceptance Criteria:**
- [ ] 所有关键事件 MUST 携带 `session_id/trace_id/timestamp`
- [ ] `plan_created` MUST 携带 `plan_id`
- [ ] `step_update` MUST 携带 `step_id`
- [ ] `approval_required` MUST 携带 `plan_id + step_id`
- [ ] 重规划开始时 SHOULD 输出 `replan_started` 或等价高层事件
- [ ] 前端主流程 SHOULD 以高层事件驱动 `ThoughtChain`
- [ ] 最终用户可读正文 SHOULD 支持 `delta` 流式输出
- [ ] 底层 `tool_call/tool_result` MAY 作为调试或补充信息存在，但 MUST NOT 成为主体验

#### Scenario: frontend renders ThoughtChain from high-level events
- **GIVEN** 用户发起一次复杂调查任务
- **WHEN** 后端逐阶段执行 Rewrite / Plan / Execute / Summary
- **THEN** 前端 MUST 能根据高层事件更新 ThoughtChain
- **AND** 用户 MUST 能感知 AI 当前处于哪个阶段

#### Scenario: frontend can perceive replan iteration
- **GIVEN** Summarizer 判定当前证据不足，需要重新规划
- **WHEN** Orchestrator 回到 Planner
- **THEN** 后端 SHOULD 输出 `replan_started` 或等价事件
- **AND** 前端 MUST 能感知这是新一轮规划，而不是静默覆盖上一轮状态

### Requirement: Summary MUST Be Separate From Process Display

系统 MUST 将阶段过程展示与最终回答区分开。

其中：

- `ThoughtChain` 用于展示过程
- 最终正文回答用于展示用户可读输出
- `summary` 用于展示最终结构化结论

**Acceptance Criteria:**
- [ ] 最终用户可读回答 MAY 通过 `delta` 流式输出
- [ ] Summarizer MUST 输出用户可读 `summary`
- [ ] `summary` MUST 可独立渲染为结构化结论视图
- [ ] 阶段过程 MUST NOT 取代最终回答

#### Scenario: user sees both process and final answer
- **GIVEN** 一次跨专家协作的调查任务已完成
- **WHEN** 前端接收到最终总结
- **THEN** 用户 MUST 能看到 ThoughtChain 过程节点
- **AND** MUST 同时看到独立的最终回答与 `summary`

### Requirement: Summarizer MUST Support Replan Decision

Summarizer MUST 基于步骤结果和证据判断当前结论是否充分，并在证据不足时触发重规划决策。

**Acceptance Criteria:**
- [ ] Summarizer MUST 输出 `need_more_investigation`
- [ ] Summarizer SHOULD 输出下一轮调查提示或等价 `ReplanHint`
- [ ] 当 `need_more_investigation=true` 时，Orchestrator MUST 能回到 Planner
- [ ] 系统 MUST 受最大迭代次数限制

#### Scenario: summarizer requests replan when evidence is insufficient
- **GIVEN** 多个步骤已执行完成但仍无法形成充分结论
- **WHEN** Summarizer 汇总结果
- **THEN** Summarizer MUST 输出 `need_more_investigation=true`
- **AND** MUST 提供下一轮调查提示
- **AND** Orchestrator MUST 在未超过最大迭代次数时回到 Planner

### Requirement: Clarify And Approval MUST Be Distinct User Actions

系统 MAY 在同一阶段容器中展示澄清与审批，但 MUST 在语义和交互上区分这两类用户动作。

**Acceptance Criteria:**
- [ ] `clarify_required` MUST 表示“信息不足，需要用户补充”
- [ ] `approval_required` MUST 表示“信息足够，但执行前需要授权”
- [ ] 前端 MAY 将二者映射到统一阶段容器
- [ ] 前端在标题、描述、CTA、恢复语义上 MUST 区分二者

#### Scenario: frontend distinguishes clarify from approval
- **GIVEN** 一次请求因资源歧义需要用户补充
- **WHEN** 后端输出 `clarify_required`
- **THEN** 前端 MUST 将其显示为补充信息动作
- **AND** MUST NOT 将其显示为审批授权动作

### Requirement: Model Guardrails MUST Prevent Execution Drift

系统 MUST 为 Rewrite、Planner、Experts、Summarizer 定义约束，避免模型在执行链路中擅自补全、越权或夸大结论。

**Acceptance Criteria:**
- [ ] Rewrite MUST NOT 伪造资源 ID、权限结果或执行结论
- [ ] Rewrite 在歧义场景 SHOULD 保留 `ambiguity_flags`
- [ ] 系统 SHOULD 定义进入 `clarify` 的弱/强歧义阈值或等价规则
- [ ] Planner 在 unresolved/ambiguous 资源场景 MUST 输出 `clarify`，而不是猜测目标
- [ ] Planner MUST NOT 通过 `narrative` 隐式表达关键执行字段
- [ ] Experts MUST 只调用本领域工具
- [ ] Summarizer MUST 基于 `StepResult/Evidence` 形成结论
- [ ] Summarizer MUST 区分观察事实与推断判断
- [ ] 在证据不足时，Summarizer MUST NOT 将推断表述为已证实事实

#### Scenario: planner refuses to guess ambiguous target
- **GIVEN** Rewrite 仅提供模糊资源 hint
- **WHEN** Planner 无法唯一解析目标
- **THEN** Planner MUST 输出 `clarify`
- **AND** MUST NOT 基于最高分候选自动继续执行

#### Scenario: summarizer preserves uncertainty
- **GIVEN** StepResult 仅提供部分证据，无法证明发布就是根因
- **WHEN** Summarizer 生成结论
- **THEN** Summarizer MUST 将该判断表述为推断或待确认结论
- **AND** MUST NOT 将其描述为已证实事实

### Requirement: Rollout MUST Support Gradual Enablement And Rollback

新架构 MUST 通过渐进式发布方式启用，并具备明确回滚策略。

**Acceptance Criteria:**
- [ ] 系统 MUST 支持 feature flag、灰度开关或等价 rollout 控制
- [ ] 新链路 MUST 能与旧链路并存一段时间
- [ ] 系统 MUST 定义回滚条件与执行路径

#### Scenario: orchestrator rollout can be disabled safely
- **GIVEN** 新编排链路在灰度期间出现严重问题
- **WHEN** 运维关闭 rollout 开关
- **THEN** 请求 MUST 能回退到旧兼容路径或安全降级路径
- **AND** 不得要求前端同步切换协议后才能恢复服务
