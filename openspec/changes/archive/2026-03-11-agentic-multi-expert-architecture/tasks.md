# Tasks: Agentic Multi-Expert Architecture

## Stage 1: Core Boundary And Contracts

目标：建立新的 AI 编排主边界，定义统一契约，为后续 Rewrite / Planner / Executor / 前端对接提供稳定宿主。

### 1.1 Orchestrator Host
- [x] 定义 `internal/ai/gateway_contract.go`
- [x] 定义 `RunRequest / ResumeRequest / RuntimeContext / StreamEvent`
- [x] 实现 `internal/ai/orchestrator.go` 顶层编排入口
- [x] 明确 `internal/service/ai` 只负责 route / auth / transport shell
- [x] 清理旧 handler 假设，按新后端边界重组入口

### 1.2 Runtime State
- [x] 定义 `ExecutionState`
- [x] 定义 `StepState`
- [x] 定义 `PendingApproval`
- [x] 统一 `trace_id / session_id / plan_id / step_id` 贯穿模型
- [x] 明确 `Run(...)` / `Resume(...)` 的共享状态边界

### 1.3 Event Foundation
- [x] 创建 `internal/ai/events/`
- [x] 定义统一 `EventMeta`
- [x] 明确高层事件与底层调试事件的边界

### 1.4 Rollout Foundation
- [x] 定义新编排链路的 feature flag / rollout 开关
- [x] 定义灰度启用与回滚门槛
- [x] 定义旧链路兼容与回退路径

## Stage 2: Rewrite Stage

目标：增加入口 Rewrite，将用户口语化输入规整为可规划的半结构化任务表达。

### 2.1 Rewrite Contract
- [x] 定义 `rewrite.Output`
- [x] 定义 `normalized_goal / operation_mode / resource_hints / domain_hints / ambiguity_flags / narrative`
- [x] 约束 Rewrite 输出为半结构化协议
- [x] 明确 Rewrite 不负责最终 resolve / permission / execution plan

### 2.2 Rewrite Runtime
- [x] 创建 `internal/ai/rewrite/` 目录
- [x] 实现 `internal/ai/rewrite/rewrite.go`
- [x] 实现 `internal/ai/rewrite/prompt.go`
- [x] 选择 Eino `v0.8.0` 中适合的 Agent / middleware 装配方式
- [x] 输出面向前端的 `rewrite_result` 事件

## Stage 3: Planner

目标：让 Planner 在 Rewrite 之后稳定地产出半结构化决策与执行计划。

### 3.1 Planner Contract
- [x] 定义 `PlannerDecision`
- [x] 定义 `ExecutionPlan`
- [x] 定义 `PlanStep`
- [x] 为 `PlannerDecision` 和 `ExecutionPlan` 增加 `narrative`
- [x] 明确 `clarify/reject/direct_reply/plan` 四类决策
- [x] 约束 `mode/risk` 进入 `PlanStep`

### 3.2 Planner Tools
- [x] 实现 `resolve_service`
- [x] 实现 `resolve_cluster`
- [x] 实现 `resolve_host`
- [x] 实现 `check_permission`
- [x] 实现 `get_user_context`
- [x] 统一 `ResolveResult / ResolveCandidate / ResolveStatus`
- [x] 约束 `resolve_*` 复用 inventory 数据源
- [x] 约束 `get_user_context` 返回标准化上下文

### 3.3 Planner Runtime
- [x] 创建 `internal/ai/planner/` 目录
- [x] 实现 `internal/ai/planner/planner.go`
- [x] 实现 `internal/ai/planner/prompt.go`
- [x] 实现 Planner 结构化输出解析
- [x] 输出 `planner_state / plan_created / clarify_required`

## Stage 4: Experts As Agent Tools

目标：完成各领域专家隔离，并统一通过 Agent Tool 接入执行层。

### 4.1 Expert Registry
- [x] 创建 `internal/ai/experts/registry.go`
- [x] 定义 `Expert` 接口
- [x] 定义 `AsTool()` 导出约定
- [x] 统一专家注册与目录输出

### 4.2 HostOpsExpert
- [x] 创建 `internal/ai/experts/hostops/`
- [x] 实现 HostOpsExpert
- [x] 迁移 `os_* / host_*` 相关工具
- [x] 明确只读与写入类工具的风险元数据

### 4.3 K8sExpert
- [x] 创建 `internal/ai/experts/k8s/`
- [x] 实现 K8sExpert
- [x] 迁移 `k8s_*` 相关工具
- [x] 明确工具风险元数据

### 4.4 ServiceExpert
- [x] 创建 `internal/ai/experts/service/`
- [x] 实现 ServiceExpert
- [x] 迁移 `service_* / deployment_* / credential_* / config_*`
- [x] 明确工具风险元数据

### 4.5 DeliveryExpert
- [x] 创建 `internal/ai/experts/delivery/`
- [x] 实现 DeliveryExpert
- [x] 迁移 `cicd_* / job_*`
- [x] 明确工具风险元数据

### 4.6 ObservabilityExpert
- [x] 创建 `internal/ai/experts/observability/`
- [x] 实现 ObservabilityExpert
- [x] 迁移 `monitor_* / topology_* / audit_*`
- [x] 明确工具风险元数据

### 4.7 Planner Support Tool Cleanup
- [x] 将 `host_list_inventory` 归入 Planner support tools
- [x] 将 `service_list_inventory` 归入 Planner support tools
- [x] 将 `cluster_list_inventory` 归入 Planner support tools
- [x] 将 `permission_check` 归入 Planner support tools
- [x] 评估 `user_list / role_list` 是否仅保留在 Planner support tools
- [x] 将 `service_deploy / host_batch` 收敛为兼容入口

## Stage 5: Executor Runtime

目标：建立确定性执行层，管理 step 调度、状态机、审批和恢复。

### 5.1 Runtime Core
- [x] 创建 `internal/ai/executor/` 目录
- [x] 实现 `executor.go`
- [x] 实现 `scheduler.go`
- [x] 定义 `executor.Request / executor.Result / executor.ResumeRequest`
- [x] 实现代码驱动的 DAG 调度
- [x] 使用稳定 `step_id` 管理依赖和状态

### 5.2 Step State Machine
- [x] 定义 `pending/ready/running/waiting_approval/completed/failed/blocked/cancelled`
- [x] 实现状态流转校验
- [x] 实现 blocked dependency 处理
- [x] 实现单 step 恢复流程
- [x] 定义 `StepError` 与错误码

### 5.3 Approval And Retry
- [x] 定义 `ApprovalDecision`
- [x] 明确 `readonly/low`、`medium`、`high` 风险策略
- [x] 实现审批前持久化
- [x] 实现审批后恢复
- [x] 实现重试逻辑
- [x] 约束非幂等写工具不得自动重试
- [x] 为 resume / approval 补充幂等键或等价去重机制
- [x] 明确“重复批准请求不得重复执行副作用”

### 5.4 Step Output
- [x] 定义 `StepResult`
- [x] 定义 `Evidence`
- [x] 为 Step 输出增加 `summary`
- [x] 为前端事件增加 `user_visible_summary`

## Stage 6: Summarizer

目标：在执行完成后生成用户可读总结，并决定是否需要补充调查。

### 6.1 Summary Contract
- [x] 定义 `SummaryOutput`
- [x] 包含 `summary / conclusion / next_actions / need_more_investigation / narrative`
- [x] 约束 `summary` 作为最终结构化结论，而不是正文流式输出替代物

### 6.2 Summarizer Runtime
- [x] 创建 `internal/ai/summarizer/` 目录
- [x] 实现 `summarizer.go`
- [x] 实现 `prompt.go`
- [x] 定义 `NeedMoreInvestigation` 判定规则
- [x] 定义 `ReplanHint` 或等价重规划提示结构
- [x] 实现 `Summary -> Replan` 回路契约
- [x] 实现最大迭代次数控制
- [x] 输出 `summary` 事件

## Stage 7: Event Stream And ThoughtChain

目标：建立面向前端的高层事件语义，并用 ThoughtChain 承载阶段过程。

### 7.1 Event Schema
- [x] 定义 `rewrite_result`
- [x] 定义 `planner_state`
- [x] 定义 `plan_created`
- [x] 定义 `step_update`
- [x] 定义 `approval_required`
- [x] 定义 `clarify_required`
- [x] 定义 `replan_started`
- [x] 定义 `delta`
- [x] 定义 `summary`
- [x] 定义 `done / error`

### 7.2 Frontend Contract
- [x] 定义 ThoughtChain 阶段模型：`rewrite / plan / execute / user_action / summary`
- [x] 为每个阶段定义标题、描述、内容、状态映射
- [x] 约束前端主体验消费高层事件
- [x] 保留 `tool_call / tool_result` 作为补充信息，而不是主流程
- [x] 明确 `delta` 用于正文流式输出，`summary` 用于结构化结论

### 7.3 ThoughtChain UI
- [x] 在 AI 面板引入 `ThoughtChain`
- [x] 将 `rewrite_result` 映射到 `rewrite`
- [x] 将 `plan_created` 映射到 `plan`
- [x] 将 `step_update` 聚合到 `execute`
- [x] 将 `approval_required / clarify_required` 映射到 `user_action`
- [x] 明确区分 `clarify` 与 `approval` 的标题、说明、CTA、恢复语义
- [x] 将 `replan_started` 映射为新一轮规划提示或迭代状态
- [x] 将 `summary` 映射到 `summary`
- [x] 保持正文回答独立渲染，不与 ThoughtChain 混淆

## Stage 8: Resume API And Gateway Alignment

目标：将审批和恢复模型从旧 checkpoint 语义收敛到 plan-step 语义，同时保持网关映射清晰。

### 8.1 Resume Model
- [x] 将恢复接口收敛到 `session_id + plan_id + step_id`
- [x] 明确前端审批请求模型
- [x] 明确 rejected/cancelled 的用户可见语义
- [x] 明确重复恢复请求的幂等响应语义

### 8.2 Gateway Alignment
- [x] 对齐 `/api/v1/ai/chat`
- [x] 定义规范的 step-resume 接口，并将 `/api/v1/ai/approval/respond` 映射到该语义
- [x] 为 `/api/v1/ai/adk/resume` 定义兼容策略，避免继续暴露旧 checkpoint 心智模型
- [x] 统一 route 层与 orchestrator host 的请求映射

### 8.3 Model Guardrails
- [x] 补充运行时和 prompt 约束，避免模型在 Rewrite / Planner / Expert / Summarizer 阶段跑偏

#### Rewrite Guardrails
- [x] 明确 Rewrite MUST NOT 伪造资源 ID、权限结果、执行结论
- [x] 明确 Rewrite 遇到歧义时保留 `ambiguity_flags`，而不是擅自消歧
- [x] 定义进入 `clarify` 的歧义阈值或等价判定规则

#### Planner Guardrails
- [x] 明确 Planner 在 unresolved / ambiguous 资源场景 MUST 输出 `clarify`
- [x] 明确 Planner MUST 使用结构化字段表达 `mode/risk/depends_on`
- [x] 明确 Planner MUST NOT 通过 narrative 隐式塞入执行要求

#### Expert Guardrails
- [x] 明确 Experts MUST 只调用本领域工具
- [x] 明确 Experts MUST NOT 越权调用 Planner support tools
- [x] 明确 Experts 输出结论时区分“观察事实”和“推断判断”

#### Summarizer Guardrails
- [x] 明确 Summarizer MUST 基于 `StepResult/Evidence` 生成结论
- [x] 明确 Summarizer MUST 标记不确定性，不得将推断表述为已证实事实
- [x] 明确 Summarizer 在证据不足时输出 `need_more_investigation=true`

## Stage 9: Testing

目标：验证新架构的契约、执行链路和前端体验。

### 9.1 Unit Tests
- [x] Rewrite 输出协议测试
- [x] Planner 决策协议测试
- [x] Resolve 工具测试
- [x] Executor 状态机测试
- [x] Approval / Resume 测试
- [x] Resume 幂等测试
- [x] Summarizer 判定测试
- [x] Event schema 测试

### 9.2 Integration Tests
- [x] 端到端主链路测试
- [x] 澄清场景测试
- [x] 审批恢复测试
- [x] 灰度开关与回滚路径测试
- [x] 重启后 `waiting_approval` 恢复测试
- [x] 多专家协作测试
- [x] Replan 事件与前端感知测试
- [x] Summary 输出测试
- [x] ThoughtChain 事件对接测试

### 9.3 Evaluation
- [x] 定义 Rewrite 质量评估指标
- [x] 定义 Planner clarify 率 / plan 可执行率指标
- [x] 定义 Resume 成功率 / 重复恢复拦截率指标
- [x] 定义 ThoughtChain 事件完整率与前端渲染一致性指标

## Stage 10: Cleanup And Migration

目标：清理旧假设、保留兼容入口、补全文档，完成迁移收口。

### 10.1 Cleanup
- [x] 删除旧 proposal/design 不再适配的实现假设
- [x] 清理旧 handler 依赖残留
- [x] 清理旧单体 Agent 主链路的过时绑定
- [x] 清理与新事件模型冲突的旧 SSE 假设

### 10.2 Migration Docs
- [x] 更新 API 文档
- [x] 更新架构文档
- [x] 补充前端 ThoughtChain 对接说明
- [x] 补充旧链路到新链路的迁移说明
