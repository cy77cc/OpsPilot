# AI 链路级 Contract/Security/Performance 全量治理设计

- 日期: 2026-03-24
- 作者: Codex
- 状态: Draft (已与用户口头确认方案结构，待用户审阅文档)
- 适用周期: 6-8 周

## 1. 背景与问题陈述

当前 AI 模块已经具备 Plan/Execute/Replan、审批中断与恢复、SSE 流式交互、历史投影与懒加载能力，但在链路级稳定性上存在系统性风险：

1. 接口契约未完全单一事实源化，字段命名、可空语义与事件协议在前后端存在漂移风险。
2. 错误分层与前端错误态反馈不够精确，存在“业务错误被泛化”为服务错误、UI 吞错或误导用户状态的问题。
3. 审批链路在前端防重入与后端幂等语义上仍有收敛空间，易出现重复提交和冲突噪声。
4. 会话列表/历史回放存在性能和可扩展性隐患（N+1、过量返回、大会话内存压力）。

## 2. 目标与非目标

### 2.1 目标

1. 建立 Contract-First 治理面，REST + SSE + 错误码语义单一事实源。
2. 形成“业务终态 vs 运行时终态”一致模型，审批拒绝/过期不再等同 runtime 崩溃。
3. 完成双端幂等与防重复提交闭环，审批流程可追踪、可恢复、可观测。
4. 落地大会话性能治理：接口摘要化、懒加载标准化、回放一致性验证。
5. 建立持续治理机制：contract diff 门禁、兼容性回归、SLO 监控。

### 2.2 非目标

1. 不重写现有 AI 业务编排框架（ADK runtime 不做替换）。
2. 不在本轮引入新业务能力（例如新的 agent 类型或新工具域）。
3. 不做与 AI 主链路无关的全仓库级重构。

## 3. 方案选型

## 3.1 候选方案

### 方案 A（推荐）: 契约先行 + 四轨并行
- 先冻结并发布 AI contract，再并行推进接口、错误、安全、性能四个维度。
- 优点: 返工最低，跨端稳定性最好，适合复杂链路。
- 缺点: 前 1-2 周主要建设治理基础，功能体感提升相对滞后。

### 方案 B: 先性能后契约
- 先做接口性能与懒加载，再回收契约。
- 优点: 短期体感改善快。
- 缺点: 容易因后续契约调整导致再次返工。

### 方案 C: 先安全后其余
- 先完成审批安全与幂等，再逐步做契约和性能。
- 优点: 安全收益立刻可见。
- 缺点: 契约和体验债务持续存在，收口周期拉长。

## 3.2 最终决策

采用 **方案 A**，以 Contract-First 为主轴，分阶段推进四维整治。

## 4. 目标架构（Contract-First 治理面）

## 4.1 Contract Package（单一事实源）

定义并版本化以下内容：

1. REST schema: `/ai/sessions`, `/ai/runs/:id`, `/ai/approvals/*`, `/ai/run-contents/*`。
2. SSE schema: 事件名、payload、`event_id`、续传语义、兼容规则。
3. Error schema: auth/permission/business/server 分层与可恢复标记。

输出物：

1. 后端响应校验器（出站校验）。
2. 前端 TS 类型 + 运行时 validator（入站校验）。
3. Contract 测试夹具（consumer/provider 合约测试）。

## 4.2 执行态与审批态分离语义

1. 审批业务态: `pending/approved/rejected/expired`。
2. 运行时态: `running/resuming/resumed/completed/completed_with_tool_errors/resume_failed_*`。
3. 约束: `rejected/expired` 为业务终态，不能被错误归类为 runtime crash。

## 5. 模块拆分与边界

## 5.1 contract（新增治理层）
- 职责: 维护协议与错误语义。
- 边界: 业务模块不得绕过 contract 自由拼 JSON。

## 5.2 run-runtime（后端执行态）
- 职责: 内部事件标准化与 run 生命周期推进。
- 边界: 仅输出公共事件，不泄露内部编排细节。

## 5.3 approval-domain（后端审批态）
- 职责: 风险策略、审批任务状态机、outbox、恢复 worker。
- 边界: 审批业务状态与运行状态清晰分层。

## 5.4 ai-query-api（后端读模型）
- 职责: 会话摘要、run projection、content lazy fetch。
- 边界: 列表接口默认摘要化，大字段必须引用化加载。

## 5.5 ai-client-runtime（前端状态机）
- 职责: SSE 消费、滚动状态机、审批 UI 状态、历史回放。
- 边界: 只消费合同内事件；未知事件仅观测降级，不驱动主状态。

## 5.6 observability-governance（双端治理）
- 职责: contract 门禁、兼容监控、错误看板、回放一致性告警。

## 6. 关键数据流与状态机约束

## 6.1 会话执行主链

1. `POST /ai/chat` 以 `session_id + client_request_id` 做幂等建壳。
2. runtime 标准化公共事件，附稳定 `event_id`。
3. 写入 `event_log` 作为事实源。
4. projector 生成 `run_projection`（读模型）。
5. 大内容写 `content_store`，projection 只保留引用。
6. SSE 对外仅发公共协议事件。

## 6.2 审批中断链

1. 高风险 tool call 命中策略，创建 `approval_task(pending)`。
2. 推送 `tool_approval`（`approval_id/preview/timeout`）。
3. 前端进入 `waiting-approval`，按钮进入 single-flight 锁定。
4. 用户提交决策写 outbox：
   - approved -> `approval_decided`
   - rejected/expired -> 业务终态
5. worker 异步推进：`ai.run.resuming -> ai.run.resumed -> ai.run.completed | ai.run.resume_failed`。

## 6.3 不可违反约束

1. `waiting_approval` 必须有真实审批实体映射。
2. 每个审批决策必须可追踪到 task + outbox + run event。
3. `resume_failed_retryable` 不得导致 session 作废。
4. unknown event 仅观测，不能推进核心 UI 状态。
5. `event_id` 在单 run 内严格单调，`last_event_id` 可精确续传。

## 7. 分阶段里程碑（6-8 周）

## 7.1 Phase A（第 1-2 周）合同冻结与止血

1. 统一命名与可空语义（REST + SSE）。
2. 修复 SSE 实时 `id` 透传与续传断链。
3. 修复审批失败前端卡死与重复提交。

交付:
- `ai-contract v1`
- 跨端兼容矩阵
- P0 回归用例集

## 7.2 Phase B（第 3-4 周）错误分层与安全闭环

1. 后端拆分错误语义（未授权/无权限/冲突/不存在/参数错误/服务错误）。
2. 前端建立错误映射与统一交互策略。
3. 审批双端幂等闭环（前端防重入 + 后端状态机守卫）。

交付:
- 错误语义手册
- 审批幂等测试集
- 审计链路补全

## 7.3 Phase C（第 5-6 周）性能与大会话治理

1. `GET /ai/sessions` 去 N+1，改摘要列表。
2. projection/content 懒加载分页化、字段裁剪。
3. 前端长会话滚动状态机与内存压测。

交付:
- 性能基线报告（P95/内存）
- 容量与降级策略

## 7.4 Phase D（第 7-8 周）治理机制固化

1. CI 门禁：contract diff + consumer contract tests。
2. 新事件灰度兼容策略与 unknown-event 观测面板。
3. 端到端演练：审批中断/恢复/超时/拒绝/重放一致性。

交付:
- 发布准入清单
- 运维 Runbook
- SLO/SLA 监控指标

## 8. 测试与验收标准（Definition of Done）

1. 契约一致性: 前后端协议来源唯一且 CI 可验证。
2. 错误冒泡: 401/403/404/409/422/5xx 可区分且 UI 行为确定。
3. 审批安全: 前端 single-flight + 后端幂等守卫 + 越权阻断。
4. 性能: 会话列表 P95 < 300ms；详情首屏 P95 < 500ms（中等数据量基线）。
5. 回放一致性: 断线续传结果与首播一致；审批链路历史回放一致。

## 9. 风险与缓解

1. 风险: 历史事件/响应与新 contract 不兼容。
- 缓解: 提供兼容适配层与版本化解析器，灰度开关发布。

2. 风险: 审批链路并发状态竞争导致重复事件。
- 缓解: 幂等键 + 状态机乐观/悲观锁策略 + outbox 去重约束。

3. 风险: 性能优化影响旧页面依赖。
- 缓解: 先新增摘要接口，旧接口过渡期双写/双读，逐步迁移。

## 10. 变更清单（本设计对应）

1. Contract 定义与校验链路（REST/SSE/error）。
2. Approval 与 Run 状态机语义收敛。
3. Session 列表与历史投影性能治理。
4. CI/监控/发布准入治理机制。

## 11. 下一步

1. 用户审阅并确认本设计文档。
2. 进入 `writing-plans`，拆分为可执行实施计划（按 Phase 与风险优先级）。
