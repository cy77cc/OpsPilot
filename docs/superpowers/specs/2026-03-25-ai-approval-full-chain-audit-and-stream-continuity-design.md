# AI 审批全链路审计与流连续性改造设计

日期：2026-03-25  
状态：Proposed  
范围：AI Chat 审批链路 + Deployment 审批链路 + CICD 审批链路（设计阶段，不改代码）

## 1. 背景与问题

当前审批能力在不同模块中均可用，但语义与链路实现存在割裂。用户反馈的核心问题是：

1. AI 工具调用触发审批 interrupt 后，SSE 事件流提前结束。
2. 用户无法在同一会话流中自然接收审批后的恢复与后续执行结果。

全量审计后，确认还存在以下共性问题：

1. 审批事件命名存在新旧并存（例如 `ai.approval.decided` 与 `approval_decided`），消费侧容易错配。
2. AI 审批采用 submit-only + worker 恢复模型，但 chat SSE 与 worker 事件桥接不完整，体感“中断后无后续”。
3. Deployment/CICD 审批状态机独立实现，缺少与 AI 审批一致的领域事件语义与统一审计视角。

## 2. 目标

### 2.1 主目标

1. AI 审批链路实现“同会话流不断开”：触发审批后保持连接，审批后在同连接内继续输出恢复与执行结果。
2. 三条审批链路（AI/Deployment/CICD）语义统一：请求、决策、执行、终态可串联可审计。
3. 事件契约兼容升级：允许旧事件兼容读取，内部以 canonical event type 统一。

### 2.2 非目标

1. 本阶段不强制合并 Deployment/CICD 与 AI 的数据库表结构。
2. 本阶段不改审批策略算法（仅改语义与传输一致性）。
3. 本阶段不进行 UI 大规模重构，仅要求状态消费一致。

## 3. 统一审批语义模型

定义跨域逻辑状态（领域层）：

1. `pending`
2. `decided`（`approved | rejected | expired`）
3. `executing`
4. `terminal`（`success | failed | cancelled`）

映射规则：

1. AI: `waiting_approval -> approved/rejected/expired -> resuming -> completed|resume_failed_*|cancelled`
2. Deployment/CICD: `pending_approval -> approved/rejected -> applying -> applied|failed|rollback`

约束：

1. `approved` 事实不可逆，不得回退到待审批。
2. 每次状态转换必须有事件和审计记录。

## 4. 方案选择

采用“逻辑单会话、物理双阶段流（方案B）”：

1. 对用户保持“同一会话连续流”体验。
2. 后端维持 submit-only + worker 恢复模型。
3. 在 chat SSE 与恢复事件间补齐桥接与 keepalive，避免连接提前结束。

原因：

1. 比“强持有单连接并跨进程唤醒”风险低。
2. 比“前端重连编排”体验更完整。
3. 与现有架构兼容成本最低。

## 5. AI 链路改造设计（重点）

### 5.1 Chat SSE 行为

当 projector 进入 `waiting_approval`：

1. 输出 `tool_approval` 与 `run_state(waiting_approval)`。
2. **不立即 return**，进入挂起等待态。
3. 周期性输出 keepalive（或 heartbeat）防止网关超时。
4. 持续监听同 `run_id` 的审批恢复事件。

### 5.2 提交与恢复

1. `submitApproval` 仍只写决策（幂等提交 + outbox）。
2. worker 消费审批决策事件并恢复执行。
3. worker 发布 `ai.run.resuming / ai.run.resumed / ai.run.resume_failed / ai.run.completed`。
4. chat SSE 在同连接消费上述事件并继续推送 delta/tool_result。

### 5.3 事件命名统一

内部 canonical event type：

1. `ai.approval.requested`
2. `ai.approval.decided`
3. `ai.approval.expired`
4. `ai.run.resuming`
5. `ai.run.resumed`
6. `ai.run.resume_failed`
7. `ai.run.completed`

兼容策略：

1. 消费侧短期兼容旧名（如 `approval_decided`）。
2. 写入侧逐步收敛到 canonical 名称。

### 5.4 断线恢复

1. 继续支持 `last_event_id` 回放。
2. 若 cursor 过期，返回显式错误码并要求前端刷新 projection。
3. 前端即使重连，也必须保持审批状态“已批准事实不可逆”。

## 6. Deployment/CICD 审批链路对齐设计

### 6.1 短期（不动主写模型）

1. 保留现有审批表和流程。
2. 增加领域事件映射层，将 `pending_approval/approved/rejected/applying/...` 映射到统一语义事件。
3. 审计字段补齐统一键：`ticket_id/run_id/release_id/actor/decision/decision_at/result`。

### 6.2 中期

1. 统一审批中心读模型（按 domain 聚合视图）。
2. 支持跨域追踪同一变更链路（AI 指令触发 Deployment/CICD 审批）。

## 7. 风险与缓解

1. 长连接占用增长  
缓解：等待态低频 heartbeat、连接超时策略、run 级并发限流。

2. 事件重复/乱序  
缓解：`event_id` 幂等去重；同 `run_id` 使用 `sequence` 有序消费。

3. 新旧事件并存导致行为漂移  
缓解：统一 canonical 写入，消费端兼容窗口明确并设下线时间。

## 8. 验收标准

1. AI 审批触发后，chat SSE 不中断，直到审批后续状态可见。
2. 审批通过后，用户在同连接看到 `resuming -> resumed/completed`。
3. 审批拒绝/过期后，看到明确终态，不出现“悬空等待”。
4. Deployment/CICD 可输出统一语义的审批事件与审计字段。
5. 旧事件兼容期间，无回归失败。

## 9. 测试计划

### 9.1 AI

1. interrupt 后连接持续性测试（含 heartbeat）。
2. submit 后 worker 恢复事件在同连接可见。
3. 旧/新事件名混用兼容测试。
4. 断线 `last_event_id` 回放一致性测试。

### 9.2 Deployment/CICD

1. 审批状态迁移映射测试。
2. 审计字段完整性测试。
3. 与 AI 指令触发路径的追踪一致性测试。

### 9.3 端到端

1. AI 请求 -> 工具审批 -> 用户决策 -> 恢复执行 -> 终态可见。
2. 发布审批（Deployment/CICD）-> 决策 -> 执行 -> 终态与审计一致。

## 10. 分期计划

1. Phase 1：AI 链路不断流闭环（SSE 挂起/keepalive/恢复桥接、事件名统一）。
2. Phase 2：Deployment/CICD 语义映射与审计字段对齐。
3. Phase 3：统一审批中心读模型与跨域追踪。

