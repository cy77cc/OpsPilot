# Tasks: Agentic Multi-Expert Architecture

## Phase 1: 基础框架

### 1.1 目录结构与配置
- [ ] 创建 `internal/ai/planner/` 目录
- [ ] 创建 `internal/ai/executor/` 目录
- [ ] 创建 `internal/ai/experts/` 目录及子目录
- [ ] 创建 `internal/ai/summarizer/` 目录
- [ ] 实现 `internal/ai/config.go` 配置加载

### 1.2 专家注册表
- [ ] 实现 `internal/ai/experts/registry.go` 专家注册表
- [ ] 定义 `Expert` 接口和基础结构
- [ ] 定义 `ExpertAgent.AsTool()` 接口，统一 `AgentAsTool` 导出能力
- [ ] 实现 `ExpertCatalog()` 方法生成专家目录

### 1.3 Planner 基础框架
- [ ] 实现 `internal/ai/planner/planner.go` Planner Agent
- [ ] 实现 `internal/ai/planner/prompt.go` 系统提示词
- [ ] 定义 `planner.Request` / `planner.Response` / `planner.Interface`
- [ ] 定义 `ExecutionPlan` / `PlanStep` / `ClarificationResponse` 结构化契约
- [ ] 定义 `PlannerDecision` 协议，覆盖 `clarify/reject/plan/direct_reply`
- [ ] 约束 `PlanStep.Intent` 的枚举集合和 `PlanStep.Input` 的最小输入边界
- [ ] 实现 Planner 结构化输出解析，不依赖自然语言计划正文

## Phase 2: Planner 工具

### 2.1 资源解析工具
- [ ] 实现 `resolve_service` 工具
  - 支持精确匹配和模糊匹配
  - 返回服务列表（ID、名称、环境、状态）
- [ ] 实现 `resolve_cluster` 工具
  - 支持按名称、环境解析
- [ ] 实现 `resolve_host` 工具
  - 支持主机名、IP、关键词匹配
- [ ] 统一 `ResolveResult` / `ResolveCandidate` / `ResolveStatus` 协议
- [ ] 明确 `resolve_*` 复用 `*_list_inventory` 的分层关系，不重复实现底层查询
- [ ] 定义 `planner/resolve.go` 中的 `ResolveServiceInput` / `ResolveClusterInput` / `ResolveHostInput`
- [ ] 约束三个 resolve 工具统一复用 inventory 工具并返回 `ResolveResult`
- [ ] 定义统一 `ResolveScore` 评分模型和分项权重
- [ ] 定义 `exact/ambiguous/missing` 的阈值规则
- [ ] 定义澄清候选的裁剪、排序和文案规则
- [ ] 为 `service/cluster/host` 三类资源补充差异化匹配规则

### 2.2 权限检查工具
- [ ] 实现 `check_permission` 工具
  - 检查用户对资源的操作权限
  - 返回权限状态和原因
- [ ] 定义 `planner/permission.go` 中的 `PermissionCheckInput` / `PermissionCheckResult`
- [ ] 约束 `check_permission` 仅做预检查，不生成审批 token

### 2.3 上下文工具
- [ ] 实现 `get_user_context` 工具
  - 获取用户当前页面
  - 获取选中的资源
  - 获取运行时上下文
- [ ] 定义 `planner/context.go` 中的 `UserContext` 结构
- [ ] 约束 `get_user_context` 返回标准化上下文，而不是直接透传原始网关 payload

### 2.4 Planner 工具装配
- [ ] 将 Planner 工具层拆分为 `tools.go` / `resolve.go` / `context.go` / `permission.go`
- [ ] 约束 Planner 内部默认调用顺序：`get_user_context -> resolve_* -> check_permission`

## Phase 3: 领域专家迁移

### 3.1 HostOpsExpert
- [ ] 实现 `internal/ai/experts/hostops/expert.go`
- [ ] 实现 `internal/ai/experts/hostops/prompt.go` 提示词
- [ ] 按统一骨架实现 HostOpsExpert 的 `Name/Description/AsTool`
- [ ] 明确 HostOpsExpert prompt contract（角色/边界/工具/决策原则/输出格式）
- [ ] 迁移工具:
  - [ ] os_get_cpu_mem
  - [ ] os_get_disk_fs
  - [ ] os_get_net_stat
  - [ ] os_get_process_top
  - [ ] os_get_journal_tail
  - [ ] os_get_container_runtime
  - [ ] host_exec
  - [ ] host_batch_exec_preview
  - [ ] host_batch_exec_apply
  - [ ] host_batch_status_update
  - [ ] host_ssh_exec_readonly

### 3.2 K8sExpert
- [ ] 实现 `internal/ai/experts/k8s/expert.go`
- [ ] 实现 `internal/ai/experts/k8s/prompt.go` 提示词
- [ ] 按统一骨架实现 K8sExpert 的 `Name/Description/AsTool`
- [ ] 明确 K8sExpert prompt contract（角色/边界/工具/决策原则/输出格式）
- [ ] 迁移工具:
  - [ ] k8s_query
  - [ ] k8s_events
  - [ ] k8s_logs
  - [ ] k8s_list_resources
  - [ ] k8s_get_events
  - [ ] k8s_get_pod_logs

### 3.3 ServiceExpert
- [ ] 实现 `internal/ai/experts/service/expert.go`
- [ ] 实现 `internal/ai/experts/service/prompt.go` 提示词
- [ ] 按统一骨架实现 ServiceExpert 的 `Name/Description/AsTool`
- [ ] 明确 ServiceExpert prompt contract（角色/边界/工具/决策原则/输出格式）
- [ ] 迁移工具:
  - [ ] service_status
  - [ ] service_get_detail
  - [ ] service_deploy_preview
  - [ ] service_deploy_apply
  - [ ] service_catalog_list
  - [ ] service_category_tree
  - [ ] service_visibility_check
  - [ ] deployment_target_list
  - [ ] deployment_target_detail
  - [ ] deployment_bootstrap_status
  - [ ] credential_list
  - [ ] credential_test
  - [ ] config_app_list
  - [ ] config_item_get
  - [ ] config_diff

### 3.4 DeliveryExpert
- [ ] 实现 `internal/ai/experts/delivery/expert.go`
- [ ] 实现 `internal/ai/experts/delivery/prompt.go` 提示词
- [ ] 按统一骨架实现 DeliveryExpert 的 `Name/Description/AsTool`
- [ ] 明确 DeliveryExpert prompt contract（角色/边界/工具/决策原则/输出格式）
- [ ] 迁移工具:
  - [ ] cicd_pipeline_list
  - [ ] cicd_pipeline_status
  - [ ] cicd_pipeline_trigger
  - [ ] job_list
  - [ ] job_execution_status
  - [ ] job_run

### 3.5 ObservabilityExpert
- [ ] 实现 `internal/ai/experts/observability/expert.go`
- [ ] 实现 `internal/ai/experts/observability/prompt.go` 提示词
- [ ] 按统一骨架实现 ObservabilityExpert 的 `Name/Description/AsTool`
- [ ] 明确 ObservabilityExpert prompt contract（角色/边界/工具/决策原则/输出格式）
- [ ] 迁移工具:
  - [ ] monitor_alert
  - [ ] monitor_metric
  - [ ] monitor_alert_rule_list
  - [ ] monitor_alert_active
  - [ ] monitor_metric_query
  - [ ] topology_get
  - [ ] audit_log_search

## Phase 3.6: Planner 支撑工具归位

- [ ] 将 `host_list_inventory` 归入 Planner 资源解析工具
- [ ] 将 `service_list_inventory` 归入 Planner 资源解析工具
- [ ] 将 `cluster_list_inventory` 归入 Planner 资源解析工具
- [ ] 将 `permission_check` 归入 Planner 权限预检查工具
- [ ] 评估 `user_list` / `role_list` 是否仅保留在 Planner 辅助工具集中
- [ ] 定义 Planner 的 expert selection policy，覆盖运行态异常/发布失败/配置问题/权限审计问题
- [ ] 将旧兼容入口 `service_deploy` 从专家标准工具集移出，仅保留迁移兼容层
- [ ] 将旧兼容入口 `host_batch` 从专家标准工具集移出，仅保留迁移兼容层

## Phase 4: Executor

### 4.1 核心实现
- [ ] 实现 `internal/ai/executor/executor.go` Executor Agent
- [ ] 定义 `executor.Request` / `executor.Result` / `executor.ResumeRequest` / `executor.Interface`
- [ ] 实现 `internal/ai/executor/scheduler.go` 调度器
- [ ] 实现代码驱动的 DAG 调度
- [ ] 使用稳定 `step_id` 管理依赖、事件、重试和恢复
- [ ] 实现 `ExpertTaskInput` / `StepResult` / `Evidence` 统一契约
- [ ] 定义 `StepState` 状态机：`pending/ready/running/waiting_approval/retrying/blocked/failed/completed/cancelled`
- [ ] 定义统一 `StepError` 模型和错误码集合
- [ ] 明确审批恢复状态 `ApprovalResumeState` 和 step 级恢复流程

### 4.2 错误处理
- [ ] 实现重试逻辑 (max_retry=3)
- [ ] 实现超时控制
- [ ] 实现用户友好的错误消息
- [ ] 实现 blocked dependency 处理，避免下游步骤误执行
- [ ] 定义 `ToolMeta` 统一元数据：`mode/risk/idempotent/approval_policy/mutates_resources/produces_evidence`
- [ ] 为所有迁入 Expert 的 write 类工具补齐 `ToolMeta`
- [ ] 定义 `ApprovalDecision` 并在 Executor/tool middleware 中统一判定
- [ ] 定义 `read/write` × `low/medium/high` 风险矩阵与默认审批策略
- [ ] 约束 `Idempotent=false` 的写工具不得被 Executor 自动重试
- [ ] 为具备副作用的工具补充 `request_id` 或等价去重键支持
- [ ] 为批量执行类工具明确 `preview -> approval -> apply` 三段式契约
- [ ] 定义 `ToolExecutionReceipt` 并约束 Expert 将副作用摘要归并进 `StepResult`
- [ ] 对齐现有 `ToolModeReadonly/ToolModeMutating` 与 `ToolRiskLow/Medium/High`，避免设计与旧实现枚举不一致
- [ ] 明确 `ToolRiskMedium=review/edit`、`ToolRiskHigh=approval` 的运行时兼容语义
- [ ] 按 expert 维度整理现有工具的 `mode/risk/approval_policy` 映射表
- [ ] 标记迁移兼容工具 `service_deploy`、`host_batch` 的下线策略

## Phase 5: Summarizer

### 5.1 核心实现
- [ ] 实现 `internal/ai/summarizer/summarizer.go` Summarizer Agent
- [ ] 实现 `internal/ai/summarizer/prompt.go` 系统提示词
- [ ] 定义 `summarizer.Request` / `summarizer.Response` / `summarizer.Interface`
- [ ] 定义 `SummarizerInput` / `SummarizerDecision` 结构化输出
- [ ] 实现缺失事实 `MissingFacts` 与重规划提示 `NextPlanHints`
- [ ] 明确 Summarizer 的判定规则和 `NeedMoreInvestigation` 触发条件

### 5.2 循环控制
- [ ] 实现迭代计数
- [ ] 实现最大迭代限制

## Phase 6: Orchestrator

### 6.1 编排入口
- [ ] 重构 `internal/ai/orchestrator.go` 顶层编排
- [ ] 定义 `ai.RunRequest` / `ai.RunResult` / `ai.ResumeRequest` / `ai.Orchestrator`
- [ ] 实现完整流程:
  - Planner → Executor → Expert → Summarizer
- [ ] 实现迭代循环
- [ ] 实现用户澄清处理
- [ ] 实现审批/中断后的 `Resume(...)` 流程
- [ ] 明确正常执行、澄清分支、审批恢复、重规划、失败终止的时序实现

### 6.2 SSE 事件
- [ ] 实现 `planner_state` 事件 (`clarifying/planning/replanning`)
- [ ] 实现 `plan_created` 事件
- [ ] 实现 `step_start` 事件
- [ ] 实现 `step_result` 事件
- [ ] 实现 `expert_progress` 事件
- [ ] 定义统一 `EventMeta` 元字段 (`session_id/plan_id/trace_id/iteration/timestamp`)
- [ ] 定义 `approval_required` / `error` / `done` 的 payload schema
- [ ] 约束同一 `step_id` 的事件顺序
- [ ] 保持兼容现有事件 (`delta`, `tool_call`, etc.)

## Phase 7: 测试与集成

### 7.1 单元测试
- [ ] Planner 工具测试
- [ ] Executor 调度器测试
- [ ] 计划解析器测试
- [ ] DAG 拓扑排序测试
- [ ] Planner decision 协议测试
- [ ] StepState 状态流转测试
- [ ] Summarizer 判定规则测试

### 7.2 集成测试
- [ ] 端到端流程测试
- [ ] 用户澄清场景测试
- [ ] 多专家协作测试
- [ ] 错误重试测试
- [ ] 迭代循环测试
- [ ] AgentAsTool 专家调用链路测试
- [ ] 计划恢复与审批中断恢复测试
- [ ] Orchestrator `Run(...)` / `Resume(...)` 双入口测试
- [ ] 灰度开关和 fallback 路径测试

### 7.3 兼容性测试
- [ ] SSE 事件流兼容验证
- [ ] 前端集成测试
- [ ] SSE 事件顺序和 payload schema 验证

## Phase 7.4: 配置与状态
- [ ] 定义运行时配置结构：Planner / Executor / Experts / Summary / Rollout
- [ ] 支持 per-expert `enabled/max_step/timeout/model`
- [ ] 定义 `ExecutionState` 持久化结构
- [ ] 在 step 状态变化、step 完成、waiting_approval 时持久化状态
- [ ] 定义标准化 `RuntimeContext` 结构，避免直接透传前端原始 payload
- [ ] 明确 `GatewayRuntime -> Orchestrator -> Planner` 的上下文注入边界
- [ ] 明确 `SessionSnapshot` / `ExecutionState` / `SessionValues` 的读写边界
- [ ] 定义统一 `TraceContext` 并约束 `Run/Resume/Planner/Executor/Expert/Tool` 全链路透传
- [ ] 定义编排层、模型层、工具层的最小遥测埋点
- [ ] 定义新 orchestrator 的核心指标与灰度对比口径
- [ ] 定义 `DebugRecord` 和原始 payload 引用策略，避免日志直接落完整 prompt/tool output
- [ ] 定义敏感信息脱敏规则，覆盖 `RawOutput/Evidence/SSE/DebugRecord`
- [ ] 定义 `ConversationHistory/PlanningContext/ExecutionSummary/RawArtifactsRef` 四层上下文模型
- [ ] 为 Planner/Expert/Summarizer 定义软硬 token 预算和超限裁剪策略
- [ ] 定义 `StepDigest` / `IterationDigest`，用于重规划和长会话压缩
- [ ] 定义固定的上下文裁剪优先级，避免不同轮次出现非确定性 prompt 形态

## Phase 8: 文档与清理

### 8.1 文档
- [ ] 更新 API 文档
- [ ] 更新架构文档
- [ ] 编写迁移指南
- [ ] 补充“旧链路 -> 新链路”模块映射说明
- [ ] 补充“逐文件迁移表”，覆盖 `agent.go` / `gateway.go` / `orchestrator.go` / `router/` / `graph/` / `approval/` / `tools/`
- [ ] 补充删除门槛和回滚策略说明

### 8.2 清理
- [ ] 将 `internal/ai/agent.go` 降级为 compat runner，并停止承载新编排逻辑
- [ ] 删除 `internal/ai/agent.go` 中全量工具池绑定和单体 `react.Agent` 主链路
- [ ] 重写 `internal/ai/orchestrator.go` 为新主编排入口，并移除旧占位/过时逻辑
- [ ] 调整 `internal/ai/gateway.go`，使其优先接入新 orchestrator 而不是旧单体 Agent
- [ ] 收敛 `internal/ai/gateway.go`：只负责 session/runtime context/SSE/approval，不承载推理逻辑
- [ ] 评估 `internal/ai/router/` 是否仍有保留价值；如无则删除
- [ ] 评估 `internal/ai/graph/` 是否仍有保留价值；如无则删除
- [ ] 若保留 `router/`，将其职责收缩为轻量预分类；若不保留，删除对应调用方
- [ ] 若保留 `graph/`，仅抽取可复用校验/执行能力；若不保留，删除旧 ActionGraph 及其测试
- [ ] 保留 `approval/`、`aspect/`、`state/`、`rag/`、`tools/` 作为新架构支撑层，并完成新接口适配
- [ ] 删除旧 4 expert 命名、映射表和过时文档片段
- [ ] 清理未使用的导入
- [ ] 更新 memory/MEMORY.md

### 8.3 验收后删除
- [ ] 在新 orchestrator 成为默认入口后，移除旧 fallback 开关
- [ ] 删除已无调用方的 compat 适配代码
- [ ] 删除旧 `router + graph + single-agent` 组合链路的集成测试，替换为新 orchestrator 集成测试
- [ ] 删除旧链路专属测试，保留或重写为新架构测试
